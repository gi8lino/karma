package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gi8lino/karma/internal/gitignore"
	"github.com/gi8lino/karma/internal/logging"
	"gopkg.in/yaml.v3"
)

// Options describe how the processor behaves for each tree.
type Options struct {
	ResourceOrder          []string
	Skip                   []string
	Opaque                 []string
	Preserve               []string
	PreserveKustomizations []string
	UseGitIgnore           bool
	IncludeDot             bool
	AddDirSuffix           bool
	AddDirPrefix           bool
	DryRun                 bool
}

// ResourceStats holds the results of processing a tree.
type ResourceStats struct {
	Reordered int
	Added     int
	Removed   int
	Updated   int
	NoOp      int
}

// Add adds the other stats to this one.
func (s *ResourceStats) Add(other ResourceStats) {
	s.Reordered += other.Reordered
	s.Added += other.Added
	s.Removed += other.Removed
	s.Updated += other.Updated
	s.NoOp += other.NoOp
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
		skipRules: parseSkipRules(opts.Skip, opts.Opaque, opts.Preserve),
	}
}

// Process walks a directory tree and updates kustomizations incrementally.
func (p *Processor) Process(ctx context.Context, dir string) (ResourceStats, error) {
	matcher, err := gitignore.Load(dir, p.opts.UseGitIgnore)
	if err != nil {
		return ResourceStats{}, err
	}
	return p.walkDir(ctx, dir, dir, matcher, false)
}

// walkDir processes the current directory and recurses into children.
func (p *Processor) walkDir(ctx context.Context, dir, base string, matcher gitignore.Matcher, skipUpdate bool) (ResourceStats, error) {

	// Resolve which kustomization file should be touched (yaml or yml).
	kustomizationPath, exists, pathErr := p.pickKustomizationPath(dir)
	if pathErr != nil {
		return ResourceStats{}, pathErr
	}
	preserveKustomization := p.isPreservedKustomization(base, kustomizationPath)
	component := false
	if exists && !preserveKustomization {
		kind, kindErr := readKustomizeKind(kustomizationPath)
		if kindErr != nil {
			return ResourceStats{}, kindErr
		}
		component = strings.EqualFold(kind, "Component")
	}

	// A preserved Kustomization is outside Karma's ownership, so don't parse it
	// while deciding how to traverse the directory.
	scanKustomizationPath := kustomizationPath
	if preserveKustomization {
		scanKustomizationPath = ""
	}

	// Load the entries once so scanEntries can handle ignores and skip logic.
	dirEntries, fileEntries, subdirs, err := p.scanEntries(ctx, dir, base, matcher, scanKustomizationPath)
	if err != nil {
		return ResourceStats{}, err
	}

	var stats ResourceStats

	// Rewrite the kustomization file if it changed, unless this exact file is preserved.
	fileStats, err := p.applyKustomization(
		dir,
		kustomizationPath,
		exists,
		dirEntries,
		fileEntries,
		skipUpdate || component || preserveKustomization,
	)
	if err != nil {
		return ResourceStats{}, err
	}
	stats.Add(fileStats)

	// Recurse into each child unless marked as "skipWalk".
	for _, child := range subdirs {
		if child.skipWalk {
			continue
		}
		childStats, err := p.walkDir(ctx, filepath.Join(dir, child.name), base, matcher, child.skipUpdate)
		if err != nil {
			return ResourceStats{}, err
		}
		stats.Add(childStats)
	}

	return stats, nil
}

// scanEntries returns the directories, YAML files, and recursion hints for dir.
// The returned slices are:
//
//	dirEntries: resource directories that belong in this kustomization,
//	fileEntries: YAML files within dir that belong in this kustomization,
//	childDirs: metadata that controls how each subdirectory is traversed.
func (p *Processor) scanEntries(
	ctx context.Context,
	dir, base string,
	matcher gitignore.Matcher,
	kustomizationPath string,
) (dirEntries []string, fileEntries []string, childDirs []childDir, err error) {
	referenced, err := referencedLocalEntries(kustomizationPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get all items in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}

	// Walk entries so ignores and skip patterns are applied deterministically.
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}

		fullPath := filepath.Join(dir, entry.Name())
		if isKustomization(entry.Name()) || (kustomizationPath != "" && filepath.Clean(fullPath) == filepath.Clean(kustomizationPath)) {
			continue
		}

		// Skip hidden entries when configured to ignore dotfiles.
		if !p.opts.IncludeDot && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Compute the relative path for logging.
		rel := p.relPath(base, fullPath)

		// Check .gitignore before skip patterns.
		if matcher != nil && matcher.Ignored(fullPath, entry.IsDir()) {
			p.logger.Skipped("path", rel, "reason", "gitignore")
			continue
		}

		// Ask the skip matcher whether this resource should be withheld.
		skip, mode, pattern := matchSkip(rel, entry.IsDir(), p.skipRules)
		if skip {
			p.logger.Skipped("path", rel, "reason", "pattern", "pattern", pattern)
			if !entry.IsDir() {
				continue
			}
			// Directories may remain listed but we adjust recursion based on skip mode
			dirEntries, childDirs = handleSkipDir(entry, mode, dirEntries, childDirs)
			continue
		}

		// Record directories and schedule recursive processing. A directory explicitly
		// referenced by another Kustomization field (for example components) is still
		// traversed, but is not also injected into resources.
		if entry.IsDir() {
			if _, usedElsewhere := referenced[entry.Name()]; !usedElsewhere {
				dirEntries = append(dirEntries, entry.Name())
			} else {
				p.logger.Skipped("path", rel, "reason", "referenced")
			}
			childDirs = append(childDirs, childDir{name: entry.Name()})
			continue
		}

		// Include eligible YAML files in the resource list.
		if isYAML(entry.Name()) {
			if _, usedElsewhere := referenced[entry.Name()]; usedElsewhere {
				p.logger.Skipped("path", rel, "reason", "referenced")
				continue
			}
			fileEntries = append(fileEntries, entry.Name())
		}
	}

	return dirEntries, fileEntries, childDirs, nil
}

// relPath computes a clean slash-separated relative path for logging.
func (p *Processor) relPath(base, full string) string {
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == "." {
		return filepath.Base(full)
	}
	return filepath.ToSlash(rel)
}

// isPreservedKustomization reports whether path is explicitly excluded from
// writes. Entries are exact slash-separated paths relative to the processing
// base directory.
func (p *Processor) isPreservedKustomization(base, file string) bool {
	rel, err := filepath.Rel(base, file)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(filepath.Clean(rel))

	for _, candidate := range p.opts.PreserveKustomizations {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		candidate = filepath.ToSlash(filepath.Clean(candidate))
		candidate = strings.TrimPrefix(candidate, "./")
		if rel == candidate {
			return true
		}
	}
	return false
}

// pickKustomizationPath finds the canonical Kustomize file or defaults to yaml.
func (p *Processor) pickKustomizationPath(dir string) (string, bool, error) {
	var matches []string
	for _, name := range kustomizationNames {
		full := filepath.Join(dir, name)
		info, err := os.Stat(full)
		if err == nil {
			if !info.IsDir() {
				matches = append(matches, full)
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
	}

	switch len(matches) {
	case 0:
		return filepath.Join(dir, "kustomization.yaml"), false, nil
	case 1:
		return matches[0], true, nil
	default:
		return "", false, fmt.Errorf("multiple kustomization files in %s: %s", dir, strings.Join(matches, ", "))
	}
}

// readKustomizeKind extracts the first non-empty kind from a YAML file.
func readKustomizeKind(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var meta struct {
			Kind string `yaml:"kind"`
		}
		if err := dec.Decode(&meta); err != nil {
			if errors.Is(err, io.EOF) {
				return "", nil
			}
			return "", err
		}
		if meta.Kind != "" {
			return meta.Kind, nil
		}
	}
}

// updateKustomization rewrites the resources section if it changed.
func (p *Processor) updateKustomization(
	path string,
	exists bool,
	dirEntries, fileEntries []string,
) (updated bool, order, final []string, stats ResourceStats, err error) {
	// Load or initialize the target YAML document.
	root, seq, order, nodes, structureChanged, err := p.loadKustomization(path, exists)
	if err != nil {
		return false, nil, nil, ResourceStats{}, err
	}

	// Build the canonical resource order.
	final = p.mergeResources(order, dirEntries, fileEntries)
	if exists && !structureChanged && slices.Equal(final, order) {
		return false, order, final, ResourceStats{}, nil
	}
	added, removed := diffEntries(order, final)
	stats.Added = len(added)
	stats.Removed = len(removed)
	if orderChanged(order, final) {
		stats.Reordered = 1
	}

	// Build scalar nodes for each entry.
	content := make([]*yaml.Node, 0, len(final))
	for _, val := range final {
		// Reuse existing nodes whenever possible.
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

	// Encode through a buffer so the document marker can be added.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return false, nil, nil, ResourceStats{}, fmt.Errorf("close encoder: %w", err)
	}

	payload := make([]byte, 0, len(buf.Bytes())+4)
	payload = append(payload, "---\n"...)
	payload = append(payload, buf.Bytes()...)
	if !p.opts.DryRun {
		if err := writeFileAtomic(path, payload, 0o644); err != nil {
			return false, nil, nil, ResourceStats{}, fmt.Errorf("write %s: %w", path, err)
		}
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
		// Record newly introduced entries.
		added = append(added, entry)
	}

	for entry, count := range counts {
		// Anything left in counts was removed.
		for range count {
			removed = append(removed, entry)
		}
	}

	return added, removed
}

// orderChanged returns true when the order of entries has changed.
func orderChanged(old, new []string) bool {
	indexes := make(map[string][]int, len(old))

	// Map each original entry to all of its positions so interleaved duplicates can be detected.
	for i, entry := range old {
		indexes[entry] = append(indexes[entry], i)
	}

	consumed := make(map[string]int, len(indexes))
	prev := -1
	for _, entry := range new {
		list, ok := indexes[entry]
		if !ok {
			// Newly added entry, skip it.
			continue
		}
		idx := consumed[entry]
		if idx >= len(list) {
			// All occurrences of this entry already tracked; ignore extras.
			continue
		}
		pos := list[idx]
		consumed[entry]++
		if prev > pos && prev != -1 {
			// Detected a previously seen entry that now appears earlier—order changed.
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

	// Rewrite the file unless skipUpdate was requested.
	updatedDir, order, final, stats, err := p.updateKustomization(path, exists, dirEntries, fileEntries)
	if err != nil {
		return ResourceStats{}, err
	}
	// Log whether the file was updated.
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
) (root *yaml.Node, seq *yaml.Node, order []string, nodes map[string]*yaml.Node, structureChanged bool, err error) {
	root = &yaml.Node{}

	if exists {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, nil, nil, nil, false, readErr
		}
		if len(bytes.TrimSpace(data)) > 0 {
			if err := yaml.Unmarshal(data, root); err != nil {
				return nil, nil, nil, nil, false, fmt.Errorf("decode %s: %w", path, err)
			}
		}
	}

	if root.Kind == 0 {
		root.Kind = yaml.DocumentNode
		root.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 {
		return nil, nil, nil, nil, false, fmt.Errorf("%s: kustomization must be a single YAML document", path)
	}
	mapNode := root.Content[0]
	if mapNode.Kind != yaml.MappingNode || len(mapNode.Content)%2 != 0 {
		return nil, nil, nil, nil, false, fmt.Errorf("%s: kustomization root must be a YAML mapping", path)
	}

	headerChanged, err := ensureHeader(mapNode)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("%s: %w", path, err)
	}

	var resourcesChanged bool
	seq, order, nodes, resourcesChanged, err = ensureResourcesSeq(root)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("%s: %w", path, err)
	}
	return root, seq, order, nodes, headerChanged || resourcesChanged, nil
}

// ensureResourcesSeq guarantees the resources block exists without coercing
// an existing value into a different YAML type.
func ensureResourcesSeq(root *yaml.Node) (seq *yaml.Node, order []string, nodes map[string]*yaml.Node, changed bool, err error) {
	mapNode := root.Content[0]
	for i := 0; i < len(mapNode.Content); i += 2 {
		key := mapNode.Content[i]
		if key.Kind == yaml.ScalarNode && key.Value == "resources" {
			seq = mapNode.Content[i+1]
			break
		}
	}

	if seq == nil {
		changed = true
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "resources", Tag: "!!str"}
		mapNode.Content = append(mapNode.Content, keyNode, seq)
	}
	if seq.Kind != yaml.SequenceNode {
		return nil, nil, nil, false, fmt.Errorf("resources must be a YAML sequence")
	}

	nodes, order, err = collectExistingResources(seq)
	return seq, order, nodes, changed, err
}

// ensureHeader injects missing canonical header keys and validates existing
// values. Existing values are never silently rewritten.
func ensureHeader(mapNode *yaml.Node) (bool, error) {
	const apiVersion = "kustomize.config.k8s.io/v1beta1"

	hasAPIVersion := false
	hasKind := false
	for i := 0; i < len(mapNode.Content); i += 2 {
		key, value := mapNode.Content[i], mapNode.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			return false, fmt.Errorf("mapping keys must be scalars")
		}
		switch key.Value {
		case "apiVersion":
			hasAPIVersion = true
			if value.Kind != yaml.ScalarNode || value.Value != apiVersion {
				return false, fmt.Errorf("apiVersion must be %q", apiVersion)
			}
		case "kind":
			hasKind = true
			if value.Kind != yaml.ScalarNode || value.Value != "Kustomization" {
				return false, fmt.Errorf("kind must be %q", "Kustomization")
			}
		}
	}

	header := make([]*yaml.Node, 0, 4)
	if !hasAPIVersion {
		header = append(header,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "apiVersion", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: apiVersion, Tag: "!!str"},
		)
	}
	if !hasKind {
		header = append(header,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "kind", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "Kustomization", Tag: "!!str"},
		)
	}
	if len(header) > 0 {
		mapNode.Content = append(header, mapNode.Content...)
		return true, nil
	}
	return false, nil
}

// collectExistingResources indexes scalar resource entries while preserving
// the original sequence for change detection. Keeping duplicates in order means
// the canonical merge can remove them instead of mistaking the file for a no-op.
func collectExistingResources(seq *yaml.Node) (nodes map[string]*yaml.Node, order []string, err error) {
	nodes = make(map[string]*yaml.Node, len(seq.Content))
	order = make([]string, 0, len(seq.Content))

	for _, node := range seq.Content {
		if node.Kind != yaml.ScalarNode {
			return nil, nil, fmt.Errorf("resources entries must be scalar strings")
		}
		order = append(order, node.Value)
		if _, exists := nodes[node.Value]; !exists {
			nodes[node.Value] = node
		}
	}
	return nodes, order, nil
}

// mergeResources produces the canonical ordering for resources.
func (p *Processor) mergeResources(existing []string, dirEntries, fileEntries []string) []string {
	dirs := p.ensureDirPrefix(dirEntries)
	dirs = p.ensureDirSuffix(dirs)
	files := slices.Clone(fileEntries)

	slices.Sort(dirs)
	dirs = slices.Compact(dirs)
	slices.Sort(files)
	files = slices.Compact(files)

	// Preserve external entries Karma cannot positively identify as direct, locally managed
	// resources. This keeps references such as ../base and remote URLs intact.
	external := make([]string, 0, len(existing))
	for _, value := range existing {
		if !isManagedLocalResource(value) {
			external = append(external, value)
		}
	}
	slices.Sort(external)
	external = slices.Compact(external)

	order := normalizeResourceOrder(p.opts.ResourceOrder)

	final := make([]string, 0, len(external)+len(dirs)+len(files))
	for _, group := range order {
		switch group {
		case resourceGroupExternal:
			final = append(final, external...)
		case resourceGroupDirs:
			final = append(final, dirs...)
		case resourceGroupFiles:
			final = append(final, files...)
		}
	}

	return final
}

// ensureDirSuffix appends slash suffixes when configured.
func (p *Processor) ensureDirSuffix(subdirs []string) []string {
	if !p.opts.AddDirSuffix {
		return subdirs
	}
	out := make([]string, 0, len(subdirs))
	for _, s := range subdirs {
		out = append(out, strings.TrimSuffix(s, "/")+"/")
	}
	return out
}

// ensureDirPrefix appends slash prefixes when configured.
func (p *Processor) ensureDirPrefix(subdirs []string) []string {
	if !p.opts.AddDirPrefix {
		return subdirs
	}
	out := make([]string, 0, len(subdirs))
	for _, sub := range subdirs {
		out = append(out, "./"+strings.TrimPrefix(sub, "./"))
	}
	return out
}
