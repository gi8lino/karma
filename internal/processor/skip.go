package processor

import (
	"os"
	"path"
	"strings"
)

// skipMode enumerates how patterns should behave.
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
	name       string // base name of the directory.
	skipUpdate bool   // true when the child kustomization must remain untouched.
	skipWalk   bool   // true when we should not recurse into the directory.
}

// parseSkipRules compiles CLI patterns into skipRule entries.
func parseSkipRules(patterns []string) []skipRule {
	rules := make([]skipRule, 0, len(patterns))
	for _, raw := range patterns {
		rule := skipRule{raw: raw}
		switch {
		case strings.HasSuffix(raw, "/**"):
			// keep directories but skip their own kustomization.
			rule.mode = skipModeSubtree
			rule.value = strings.TrimSuffix(raw, "/**")
		case strings.HasSuffix(raw, "/*"):
			// skip immediate children but keep the parent listed.
			rule.mode = skipModeChildren
			rule.value = strings.TrimSuffix(raw, "/*")
		case strings.ContainsAny(raw, "*?[]"):
			// treat glob patterns as direct skip rules.
			rule.mode = skipModeGlob
			rule.value = raw
		default:
			// plain literal directories or files.
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
			// subtree skips cover everything below the prefix.
			if matchesPrefix(rel, rule.value) {
				return true, skipModeSubtree, rule.raw
			}
		case skipModeChildren:
			// children skips only apply to immediate descendants.
			if isDir && rel == rule.value {
				return true, skipModeChildren, rule.raw
			}
			if matchesChild(rel, rule.value) {
				return true, skipModeChildren, rule.raw
			}
		case skipModeExact:
			// exact matches drop the resource entirely.
			if rel == rule.value {
				return true, skipModeExact, rule.raw
			}

			// match by basename in case the pattern is relative.
			if !strings.Contains(rule.value, "/") && path.Base(rel) == rule.value {
				return true, skipModeExact, rule.raw
			}
		case skipModeGlob:
			// glob patterns work across the full path.
			if matched, err := path.Match(rule.value, rel); err == nil && matched {
				return true, skipModeGlob, rule.raw
			}

			// also allow glob matches against the basename for non-path patterns.
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
		// exact skips drop the directory from the resource list.
		return dirEntries, childDirs
	case skipModeChildren:
		// keep directories listed but skip their contents.
		dirEntries = append(dirEntries, name)
		childDirs = append(childDirs, childDir{name: name, skipWalk: true})
		return dirEntries, childDirs
	case skipModeSubtree:
		// keep the directory listed but never rewrite its kustomization.
		dirEntries = append(dirEntries, name)
		childDirs = append(childDirs, childDir{name: name, skipUpdate: true})
		return dirEntries, childDirs
	default:
		// fallback for unknown modes, keep the current lists unchanged.
		return dirEntries, childDirs
	}
}

// matchesPrefix reports whether rel lives inside prefix.
func matchesPrefix(rel, prefix string) bool {
	if prefix == "" {
		return true
	}
	// treat the prefix as matching itself before falling back to HasPrefix.
	if rel == prefix {
		return true
	}

	return strings.HasPrefix(rel, prefix+"/")
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
