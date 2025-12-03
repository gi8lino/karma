package processor

import (
	"os"
	"path"
	"strings"
)

// SkipMode enumerates how patterns should behave.
type skipMode int

const (
	skipModeExact skipMode = iota
	skipModeGlob
	skipModeSubtree
	skipModeChildren
)

// skipRule represents a parsed skip pattern.
type skipRule struct {
	raw   string
	mode  skipMode
	value string
}

// childDir carries metadata that controls how we recurse into a directory.
type childDir struct {
	name       string // Base name of the directory.
	skipUpdate bool   // True when the child kustomization must remain untouched.
	skipWalk   bool   // True when recursion into the directory should be skipped.
}

// parseSkipRules compiles CLI patterns into skipRule entries.
func parseSkipRules(patterns []string) []skipRule {
	rules := make([]skipRule, 0, len(patterns))
	for _, raw := range patterns {
		rule := skipRule{raw: raw}
		canonical := strings.TrimRight(raw, "/")
		switch {
		case strings.HasSuffix(canonical, "/**"):
			// Keep directories but skip their own kustomization.
			rule.mode = skipModeSubtree
			rule.value = strings.TrimSuffix(canonical, "/**")
		case strings.HasSuffix(raw, "/*"):
			// Skip immediate children but keep the parent listed.
			rule.mode = skipModeChildren
			rule.value = strings.TrimSuffix(canonical, "/*")
		case strings.ContainsAny(raw, "*?[]"):
			// Treat glob patterns as direct skip rules.
			rule.mode = skipModeGlob
			rule.value = raw
		default:
			// Plain literal directories or files.
			rule.mode = skipModeExact
			rule.value = strings.TrimSuffix(raw, "/")
		}
		rules = append(rules, rule)
	}
	return rules
}

// matchSkip determines whether rel matches any configured skip rule.
func matchSkip(rel string, isDir bool, rules []skipRule) (skip bool, mode skipMode, pattern string) {
	for _, rule := range rules {
		switch rule.mode {
		case skipModeSubtree:
			// Subtree skips only affect the directory itself, so children can still be processed.
			if rel == rule.value {
				return true, skipModeSubtree, rule.raw
			}
		case skipModeChildren:
			// Children skips only apply to immediate descendants.
			if isDir && rel == rule.value {
				return true, skipModeChildren, rule.raw
			}
			if matchesChild(rel, rule.value) {
				return true, skipModeChildren, rule.raw
			}
		case skipModeExact:
			// Exact matches drop the resource entirely.
			if rel == rule.value {
				return true, skipModeExact, rule.raw
			}

			// Match by basename in case the pattern is relative.
			if !strings.Contains(rule.value, "/") && path.Base(rel) == rule.value {
				return true, skipModeExact, rule.raw
			}
		case skipModeGlob:
			// Glob patterns work across the full path.
			if matched, err := path.Match(rule.value, rel); err == nil && matched {
				return true, skipModeGlob, rule.raw
			}

			// Also allow glob matches against the basename for non-path patterns.
			if !strings.Contains(rule.value, "/") {
				if matched, err := path.Match(rule.value, path.Base(rel)); err == nil && matched {
					return true, skipModeGlob, rule.raw
				}
			}
		}
	}
	return false, skipModeExact, ""
}

// handleSkipDir records how a skipped directory should adjust the resource lists.
func handleSkipDir(entry os.DirEntry, mode skipMode, dirEntries []string, childDirs []childDir) ([]string, []childDir) {
	name := entry.Name()
	switch mode {
	case skipModeExact:
		// Exact skips drop the directory from the resource list.
		return dirEntries, childDirs
	case skipModeChildren:
		// Keep directories listed but skip their contents.
		dirEntries = append(dirEntries, name)
		childDirs = append(childDirs, childDir{name: name, skipWalk: true})
		return dirEntries, childDirs
	case skipModeSubtree:
		// Keep the directory listed but never rewrite its kustomization.
		dirEntries = append(dirEntries, name)
		childDirs = append(childDirs, childDir{name: name, skipUpdate: true})
		return dirEntries, childDirs
	default:
		// Fallback for unknown modes, keep the current lists unchanged.
		return dirEntries, childDirs
	}
}

// matchesChild reports whether rel is a direct child of prefix.
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
