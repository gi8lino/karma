package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gi8lino/karma/internal/gitignore"
	"github.com/gi8lino/karma/internal/logging"
	"github.com/gi8lino/karma/internal/utils"
	"gopkg.in/yaml.v3"
)

// Options describe how the processor behaves for each tree.
type Options struct {
	Skip          []string
	UseGitIgnore  bool
	IncludeDot    bool
	DirSlash      bool
	ResourceOrder []string
}

type ResourceStats struct {
	Reordered int
	Added     int
	Removed   int
	Updated   int
	NoOp      int
}

// Processor walks directories and keeps kustomization resources in sync.
type Processor struct {
	opts      Options
	logger    *logging.Logger
	skipRules []skipRule
}

// New creates a processor with the provided options and logger.
func New(opts Options, logger *logging.Logger) *Processor {
	return &Processor{
		opts:      opts,
		logger:    logger,
		skipRules: parseSkipRules(opts.Skip),
	}
}

// Process walks a directory tree and updates kustomizations incrementally.
func (p *Processor) Process(ctx context.Context, dir string) (ResourceStats, error) {
	return p.walkDir(ctx, dir, dir, nil, false)
}

// walkDir processes the current directory and recurses into children.
func (p *Processor) walkDir(ctx context.Context, dir, base string, parent gitignore.Matcher, skipUpdate bool) (ResourceStats, error) {
	// Load the matcher once so we can reuse it for each directory.
	matcher, err := p.loadMatcher(dir, parent)
	if err != nil {
		return ResourceStats{}, err
	}
	var stats ResourceStats

	// Load the entries once so scanEntries can handle ignores and skip logic.
	dirEntries, fileEntries, subdirs, err := p.scanEntries(dir, base, matcher)
	if err != nil {
		return ResourceStats{}, err
	}

	// Resolve which kustomization file we should touch (yaml or yml).
	kustomizationPath, exists, pathErr := p.pickKustomizationPath(dir)
	if pathErr != nil {
		return ResourceStats{}, pathErr
	}

	// Rewrite the kustomization file if it changed.
	fileStats, err := p.applyKustomization(dir, kustomizationPath, exists, dirEntries, fileEntries, skipUpdate)
	if err != nil {
		return ResourceStats{}, err
	}
	stats.Reordered += fileStats.Reordered
	stats.Added += fileStats.Added
	stats.Removed += fileStats.Removed
	stats.Updated += fileStats.Updated
	stats.NoOp += fileStats.NoOp

	// Recurse into each child unless marked as "skipWalk".
	for _, child := range subdirs {
		if child.skipWalk {
			continue
		}
		childStats, err := p.walkDir(ctx, filepath.Join(dir, child.name), base, matcher, child.skipUpdate)
		if err != nil {
			return ResourceStats{}, err
		}
		stats.Reordered += childStats.Reordered
		stats.Added += childStats.Added
		stats.Removed += childStats.Removed
		stats.Updated += childStats.Updated
		stats.NoOp += childStats.NoOp
	}

	return stats, nil
}

// scanEntries returns the directories, YAML files, and recursion hints for dir.
// The return tuples are:
//
//	dirEntries: resource directories that belong in this kustomization,
//	fileEntries: YAML files within dir that belong in this kustomization,
//	childDirs: metadata that controls how each subdirectory is traversed.
func (p *Processor) scanEntries(
	dir, base string,
	matcher gitignore.Matcher,
) (dirEntries []string, fileEntries []string, childDirs []childDir, err error) {
	// Get all items in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}

	// Walk entries so ignores and skip patterns are applied deterministically.
	for _, entry := range entries {
		if isKustomization(entry.Name()) {
			continue
		}

		// skip hidden entries when configured to ignore dotfiles.
		if !p.opts.IncludeDot && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// compute the relative path for logging.
		fullPath := filepath.Join(dir, entry.Name())
		rel := p.relPath(base, fullPath)

		// Check .gitignore before skip patterns.
		if matcher != nil && matcher.Ignored(fullPath, entry.IsDir()) {
			p.logger.Skipped("path", rel, "reason", "gitignore")
			continue
		}

		// ask the skip matcher whether this resource should be withheld.
		skip, mode, pattern := matchSkip(rel, entry.IsDir(), p.skipRules)
		if skip {
			p.logger.Skipped("path", rel, "reason", "pattern", "pattern", pattern)
			if !entry.IsDir() {
				continue
			}
			// directories may remain listed but we adjust recursion based on skip mode
			dirEntries, childDirs = handleSkipDir(entry, mode, dirEntries, childDirs)
			continue
		}

		// record directories and schedule recursive processing.
		if entry.IsDir() {
			dirEntries = append(dirEntries, entry.Name())
			childDirs = append(childDirs, childDir{name: entry.Name()})
			continue
		}

		// include eligible YAML files in the resource list.
		if isYAML(entry.Name()) {
			fileEntries = append(fileEntries, entry.Name())
		}
	}

	return dirEntries, fileEntries, childDirs, nil
}

// loadMatcher returns the matcher for dir using the parent stack.
func (p *Processor) loadMatcher(dir string, parent gitignore.Matcher) (gitignore.Matcher, error) {
	if !p.opts.UseGitIgnore {
		return nil, nil
	}
	if parent != nil {
		return parent.Child(dir)
	}
	return gitignore.Load(dir, true)
}

// relPath computes a clean slash-separated relative path for logging.
func (p *Processor) relPath(base, full string) string {
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == "." {
		return filepath.Base(full)
	}
	return filepath.ToSlash(rel)
}

// pickKustomizationPath finds an existing file or defaults to yaml.
func (p *Processor) pickKustomizationPath(dir string) (string, bool, error) {
	candidates := []string{"kustomization.yaml", "kustomization.yml"}
	for _, name := range candidates {
		// probe the candidate path to see if the file exists.
		full := filepath.Join(dir, name)
		info, err := os.Stat(full)
		if err == nil {
			if info.IsDir() {
				continue
			}
			return full, true, nil
		}

		// propagate unexpected errors rather than treating them as missing.
		if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
	}
	// If we didn't find a kustomization, create one.
	return filepath.Join(dir, "kustomization.yaml"), false, nil
}

// updateKustomization rewrites the resources section if it changed.
func (p *Processor) updateKustomization(
	path string,
	exists bool,
	dirEntries, fileEntries []string,
) (updated bool, order, final []string, stats ResourceStats, err error) {
	// load or initialize the target YAML document.
	root, seq, order, nodes, err := p.loadKustomization(path, exists)
	if err != nil {
		return false, nil, nil, ResourceStats{}, err
	}

	// build the canonical resource order.
	final = p.mergeResources(order, dirEntries, fileEntries)
	if slices.Equal(final, order) {
		return false, order, final, ResourceStats{}, nil
	}
	added, removed := diffEntries(order, final)
	stats.Added = len(added)
	stats.Removed = len(removed)
	if orderChanged(order, final) {
		stats.Reordered = 1
	}

	// build scalar nodes for each entry.
	content := make([]*yaml.Node, 0, len(final))
	for _, val := range final {
		// reuse existing nodes whenever possible.
		if node, ok := nodes[val]; ok {
			content = append(content, node)
			continue
		}
		content = append(content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: val,
			Tag:   "!!str",
		})
	}
	seq.Content = content

	// encode through a buffer so we can add the document marker.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("close encoder: %w", err)
	}

	// create or truncate the target file before writing the encoded YAML.
	file, err := os.Create(path)
	if err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close() // nolint:errcheck

	// always prepend the canonical document start.
	if _, err := file.WriteString("---\n"); err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("write prefix: %w", err)
	}

	// write the encoded document after the header.
	if _, err := file.Write(buf.Bytes()); err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("write content: %w", err)
	}

	return true, order, final, stats, nil
}

// diffEntries returns the added and removed elements when comparing two resource lists.
func diffEntries(old, new []string) (added, removed []string) {
	counts := make(map[string]int, len(old))
	for _, entry := range old {
		counts[entry]++
	}

	for _, entry := range new {
		if counts[entry] > 0 {
			counts[entry]--
			if counts[entry] == 0 {
				delete(counts, entry)
			}
			continue
		}
		// record newly introduced entries.
		added = append(added, entry)
	}

	for entry, count := range counts {
		// anything left in counts was removed.
		for i := 0; i < count; i++ {
			removed = append(removed, entry)
		}
	}

	return added, removed
}

// orderChanged returns true when the order of entries has changed.
func orderChanged(old, new []string) bool {
	indexes := make(map[string][]int, len(old))

	// map each original entry to all of its positions so we can detect interleaved duplicates.
	for i, entry := range old {
		indexes[entry] = append(indexes[entry], i)
	}

	consumed := make(map[string]int, len(indexes))
	prev := -1
	for _, entry := range new {
		list, ok := indexes[entry]
		if !ok {
			// newly added entry, skip it.
			continue
		}
		idx := consumed[entry]
		if idx >= len(list) {
			// all occurrences of this entry already tracked; ignore extras.
			continue
		}
		pos := list[idx]
		consumed[entry]++
		if prev > pos && prev != -1 {
			// detected a previously seen entry that now appears earlier—order changed.
			return true
		}
		prev = pos
	}

	return false
}

// applyKustomization decides whether to rewrite a kustomization based on skip flags.
func (p *Processor) applyKustomization(
	dir, path string,
	exists bool,
	dirEntries, fileEntries []string,
	skipUpdate bool,
) (ResourceStats, error) {
	if skipUpdate {
		p.logger.Trace("skip-update", "dir", dir)
		return ResourceStats{}, nil
	}

	// rewrite the file unless skipUpdate was requested.
	updatedDir, order, final, stats, err := p.updateKustomization(path, exists, dirEntries, fileEntries)
	if err != nil {
		return ResourceStats{}, err
	}
	// log whether we updated anything.
	if updatedDir {
		stats = p.logUpdate(path, stats, order, final)
		stats.Updated = 1
		return stats, nil
	}

	stats.NoOp = 1
	p.logger.NoOp(path)

	return stats, nil
}

// logUpdate logs the update statistics and diffs.
func (p *Processor) logUpdate(path string, stats ResourceStats, order, final []string) ResourceStats {
	var changeParts []string
	if stats.Reordered > 0 {
		changeParts = append(changeParts, "order")
	}
	if stats.Added > 0 {
		changeParts = append(changeParts, "added")
	}
	if stats.Removed > 0 {
		changeParts = append(changeParts, "removed")
	}
	if len(changeParts) > 0 {
		p.logger.Updated(path, "change", strings.Join(changeParts, "+"))
	} else {
		p.logger.Updated(path)
	}
	p.logger.ResourceDiff(order, final)
	return stats
}

// loadKustomization reads or initializes the YAML document.
func (p *Processor) loadKustomization(
	path string,
	exists bool,
) (root *yaml.Node, seq *yaml.Node, order []string, nodes map[string]*yaml.Node, err error) {
	root = &yaml.Node{}

	if exists {
		// read the existing node tree to preserve comments.
		var data []byte
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		err = yaml.Unmarshal(data, root)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	// ensure the node is treated as a document.
	if root.Kind != yaml.DocumentNode {
		root.Kind = yaml.DocumentNode
	}

	// initialize an empty mapping if the document was empty.
	if len(root.Content) == 0 {
		root.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}

	// normalize the first child to a mapping node.
	if root.Content[0].Kind != yaml.MappingNode {
		root.Content[0].Kind = yaml.MappingNode
	}

	ensureHeader(root.Content[0])

	seq, order, nodes, err = ensureResourcesSeq(root)
	return root, seq, order, nodes, err
}

// ensureResourcesSeq guarantees the resources block exists.
func ensureResourcesSeq(root *yaml.Node) (seq *yaml.Node, order []string, nodes map[string]*yaml.Node, err error) {
	mapNode := root.Content[0]
	for i := 0; i < len(mapNode.Content); i += 2 {
		// iterate key/value pairs, keeping resources when found.
		if i+1 >= len(mapNode.Content) {
			break
		}

		key := mapNode.Content[i]
		if key.Value == "resources" {
			// stop at the first resources entry so we can reuse its sequence.
			seq = mapNode.Content[i+1]
			break
		}
	}

	// create a resources sequence if none exists yet.
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "resources", Tag: "!!str"}
		mapNode.Content = append(mapNode.Content, keyNode, seq)
	}

	// normalize the entry to a sequence node before usage.
	if seq.Kind != yaml.SequenceNode {
		seq.Kind = yaml.SequenceNode
	}

	nodes, order = collectExistingResources(seq)
	return seq, order, nodes, err
}

// ensureHeader injects the canonical header keys at the top when missing.
func ensureHeader(mapNode *yaml.Node) {
	// detect if a header already exists; if so, leave it untouched.
	for i := 0; i < len(mapNode.Content); i += 2 {
		key := mapNode.Content[i].Value
		if key == "apiVersion" || key == "kind" {
			return
		}
	}

	header := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "apiVersion", Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: "kustomize.config.k8s.io/v1beta1", Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: "kind", Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: "Kustomization", Tag: "!!str"},
	}

	// Prepend the header nodes so the header keys appear first in the document.
	mapNode.Content = append(header, mapNode.Content...)
}

// collectExistingResources indexes the existing sequence nodes.
func collectExistingResources(seq *yaml.Node) (nodes map[string]*yaml.Node, order []string) {
	nodes = make(map[string]*yaml.Node, len(seq.Content))
	order = make([]string, 0, len(seq.Content))

	for _, node := range seq.Content {
		// ignore anything that is not a scalar resource entry.
		if node.Kind != yaml.ScalarNode {
			continue
		}

		// record the first occurrence and map the node for reuse.
		if _, exists := nodes[node.Value]; !exists {
			order = append(order, node.Value)
		}

		nodes[node.Value] = node
	}

	return nodes, order
}

// mergeResources produces the canonical ordering for resources.
func (p *Processor) mergeResources(existing []string, dirEntries, fileEntries []string) []string {
	// create a copy of the existing resources.
	dirs := append([]string(nil), dirEntries...)
	files := append([]string(nil), fileEntries...)
	dirs = p.decorateSubdirs(dirs)

	sort.Strings(dirs)
	sort.Strings(files)

	// preserve remote resources from existing order.
	remote := make([]string, 0, len(existing))
	for _, value := range existing {
		if isRemoteResource(value) {
			remote = append(remote, value)
		}
	}
	sort.Strings(remote)

	order := normalizeResourceOrder(p.opts.ResourceOrder)

	final := make([]string, 0, len(remote)+len(dirs)+len(files))
	for _, group := range order {
		switch group {
		case resourceGroupRemote:
			final = append(final, remote...)
		case resourceGroupDirs:
			final = append(final, dirs...)
		case resourceGroupFiles:
			final = append(final, files...)
		}
	}

	return utils.DedupPreserve(final)
}

// decorateSubdirs appends slash suffixes when configured.
func (p *Processor) decorateSubdirs(subdirs []string) []string {
	if !p.opts.DirSlash {
		return subdirs
	}
	out := make([]string, 0, len(subdirs))
	for _, s := range subdirs {
		if trimmed, ok := strings.CutSuffix(s, "/"); ok {
			out = append(out, trimmed+"/")
			continue
		}
		out = append(out, s+"/")
	}
	return out
}
