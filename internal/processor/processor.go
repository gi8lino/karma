package processor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gi8lino/kustomizer/internal/gitignore"
	"github.com/gi8lino/kustomizer/internal/logging"
	"github.com/gi8lino/kustomizer/internal/utils"
	"gopkg.in/yaml.v3"
)

// Options describe how the processor behaves for each tree.
type Options struct {
	Skip         []string
	UseGitIgnore bool
	IncludeDot   bool
	DirSlash     bool
	DirFirst     bool
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
func (p *Processor) Process(ctx context.Context, dir string) (updated, noOp int, err error) {
	return p.Walk(ctx, dir)
}

// Walk kicks off a recursive traversal rooted at dir.
func (p *Processor) Walk(ctx context.Context, dir string) (updated, noOp int, err error) {
	matcher, err := gitignore.Load(dir, p.opts.UseGitIgnore)
	if err != nil {
		return 0, 0, err
	}
	return p.walkDir(ctx, dir, dir, matcher)
}

// walkDir processes the current directory and recurses into children.
func (p *Processor) walkDir(ctx context.Context, dir, base string, matcher gitignore.Matcher) (updated, noOp int, err error) {
	dirEntries := make([]string, 0, 16)
	fileEntries := make([]string, 0, 16)
	subdirs := make([]string, 0, 16)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}

	for _, entry := range entries {
		if entry.Name() == "kustomization.yaml" || entry.Name() == "kustomization.yml" {
			continue
		}
		if !p.opts.IncludeDot && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		rel, relErr := filepath.Rel(base, fullPath)
		if relErr != nil || rel == "." {
			rel = entry.Name()
		}
		rel = filepath.ToSlash(rel)

		if matcher != nil && matcher.Ignored(fullPath, entry.IsDir()) {
			p.logger.Skipped("path", rel, "reason", "gitignore")
			continue
		}

		skip, subtree, pattern := p.matchSkip(rel, entry.IsDir())
		if skip {
			p.logger.Skipped("path", rel, "reason", pattern)
			if entry.IsDir() {
				if !subtree {
					subdirs = append(subdirs, entry.Name())
				}
				continue
			}
			continue
		}

		if entry.IsDir() {
			dirEntries = append(dirEntries, entry.Name())
			subdirs = append(subdirs, entry.Name())
			continue
		}
		if isYAML(entry.Name()) {
			fileEntries = append(fileEntries, entry.Name())
		}
	}

	kustomizationPath, exists, pathErr := p.pickKustomizationPath(dir)
	if pathErr != nil {
		return 0, 0, pathErr
	}

	updatedDir, err := p.updateKustomization(kustomizationPath, exists, dirEntries, fileEntries)
	if err != nil {
		return 0, 0, err
	}
	if updatedDir {
		updated++
		p.logger.Updated(kustomizationPath)
	} else {
		noOp++
		p.logger.NoOp(kustomizationPath)
	}

	for _, sub := range subdirs {
		childMatcher := matcher
		if matcher != nil {
			childMatcher, err = matcher.Child(filepath.Join(dir, sub))
			if err != nil {
				return updated, noOp, err
			}
		}
		u, n, err := p.walkDir(ctx, filepath.Join(dir, sub), base, childMatcher)
		if err != nil {
			return updated, noOp, err
		}
		updated += u
		noOp += n
	}

	return updated, noOp, nil
}

// matchSkip decides if a path should be skipped based on configured rules.
func (p *Processor) matchSkip(rel string, isDir bool) (skip, subtree bool, pattern string) {
	for _, rule := range p.skipRules {
		switch rule.mode {
		case skipModeSubtree:
			if matchesPrefix(rel, rule.value) {
				return true, true, rule.raw
			}
		case skipModeChildren:
			if matchesChild(rel, rule.value) {
				return true, false, rule.raw
			}
		case skipModeExact:
			if rel == rule.value {
				return true, true, rule.raw
			}
		case skipModeGlob:
			matched, err := path.Match(rule.value, rel)
			if err == nil && matched {
				return true, true, rule.raw
			}
		}
	}
	return false, false, ""
}

type skipMode int

const (
	skipModeExact skipMode = iota
	skipModeGlob
	skipModeSubtree
	skipModeChildren
)

type skipRule struct {
	raw   string
	mode  skipMode
	value string
}

// parseSkipRules builds skip rules from configured patterns.
func parseSkipRules(patterns []string) []skipRule {
	rules := make([]skipRule, 0, len(patterns))
	for _, raw := range patterns {
		rule := skipRule{raw: raw}
		switch {
		case strings.HasSuffix(raw, "/**"):
			rule.mode = skipModeSubtree
			rule.value = strings.TrimSuffix(raw, "/**")
		case strings.HasSuffix(raw, "/*"):
			rule.mode = skipModeChildren
			rule.value = strings.TrimSuffix(raw, "/*")
		case strings.ContainsAny(raw, "*?[]"):
			rule.mode = skipModeGlob
			rule.value = raw
		default:
			rule.mode = skipModeExact
			rule.value = raw
		}
		rules = append(rules, rule)
	}
	return rules
}

func matchesPrefix(rel, prefix string) bool {
	if prefix == "" {
		return true
	}
	if rel == prefix {
		return true
	}
	return strings.HasPrefix(rel, prefix+"/")
}

func matchesChild(rel, prefix string) bool {
	if prefix == "" {
		return !strings.Contains(rel, "/")
	}
	if !strings.HasPrefix(rel, prefix+"/") {
		return false
	}
	rest := rel[len(prefix)+1:]
	return rest != "" && !strings.Contains(rest, "/")
}

// pickKustomizationPath finds an existing file or defaults to yaml.
func (p *Processor) pickKustomizationPath(dir string) (string, bool, error) {
	candidates := []string{"kustomization.yaml", "kustomization.yml"}
	for _, name := range candidates {
		full := filepath.Join(dir, name)
		info, err := os.Stat(full)
		if err == nil {
			if info.IsDir() {
				continue
			}
			return full, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
	}
	return filepath.Join(dir, "kustomization.yaml"), false, nil
}

// updateKustomization rewrites the resources section if it changed.
func (p *Processor) updateKustomization(path string, exists bool, dirEntries, fileEntries []string) (bool, error) {
	root, seq, order, nodes, err := p.loadKustomization(path, exists)
	if err != nil {
		return false, err
	}

	final := p.mergeResources(order, dirEntries, fileEntries)
	if equalStrings(final, order) {
		return false, nil
	}

	content := make([]*yaml.Node, 0, len(final))
	for _, val := range final {
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

	file, err := os.Create(path)
	if err != nil {
		return false, fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return false, fmt.Errorf("encode: %w", err)
	}
	return true, nil
}

// loadKustomization reads or initializes the YAML document.
func (p *Processor) loadKustomization(path string, exists bool) (*yaml.Node, *yaml.Node, []string, map[string]*yaml.Node, error) {
	root := &yaml.Node{}
	if exists {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if err := yaml.Unmarshal(data, root); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if root.Kind != yaml.DocumentNode {
		root.Kind = yaml.DocumentNode
	}
	if len(root.Content) == 0 {
		root.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	if root.Content[0].Kind != yaml.MappingNode {
		root.Content[0].Kind = yaml.MappingNode
	}

	seq, order, nodes, err := ensureResourcesSeq(root)
	return root, seq, order, nodes, err
}

// ensureResourcesSeq guarantees the resources block exists.
func ensureResourcesSeq(root *yaml.Node) (*yaml.Node, []string, map[string]*yaml.Node, error) {
	mapNode := root.Content[0]
	var seq *yaml.Node
	for i := 0; i < len(mapNode.Content); i += 2 {
		if i+1 >= len(mapNode.Content) {
			break
		}
		key := mapNode.Content[i]
		if key.Value == "resources" {
			seq = mapNode.Content[i+1]
			break
		}
	}
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "resources", Tag: "!!str"}
		mapNode.Content = append(mapNode.Content, keyNode, seq)
	}
	if seq.Kind != yaml.SequenceNode {
		seq.Kind = yaml.SequenceNode
	}
	nodes, order := collectExistingResources(seq)
	return seq, order, nodes, nil
}

// collectExistingResources indexes the existing sequence nodes.
func collectExistingResources(seq *yaml.Node) (map[string]*yaml.Node, []string) {
	nodes := make(map[string]*yaml.Node, len(seq.Content))
	order := make([]string, 0, len(seq.Content))
	for _, node := range seq.Content {
		if node.Kind != yaml.ScalarNode {
			continue
		}
		if _, exists := nodes[node.Value]; !exists {
			order = append(order, node.Value)
		}
		nodes[node.Value] = node
	}
	return nodes, order
}

// mergeResources produces the canonical ordering for resources.
func (p *Processor) mergeResources(existing []string, dirEntries, fileEntries []string) []string {
	dirs := utils.DedupPreserve(dirEntries)
	files := utils.DedupPreserve(fileEntries)
	dirs = p.decorateSubdirs(dirs)
	sort.Strings(dirs)
	sort.Strings(files)

	remote := make([]string, 0, len(existing))
	for _, value := range existing {
		if isRemoteResource(value) {
			remote = append(remote, value)
		}
	}
	sort.Strings(remote)

	final := make([]string, 0, len(remote)+len(dirs)+len(files))
	final = append(final, remote...)
	if p.opts.DirFirst {
		final = append(final, dirs...)
		final = append(final, files...)
	} else {
		all := append(append([]string{}, dirs...), files...)
		sort.Strings(all)
		final = append(final, all...)
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

func isYAML(name string) bool {
	lowered := strings.ToLower(name)
	return strings.HasSuffix(lowered, ".yaml") || strings.HasSuffix(lowered, ".yml")
}

func isRemoteResource(entry string) bool {
	return strings.HasPrefix(entry, "http://") || strings.HasPrefix(entry, "https://")
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
