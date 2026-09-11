package processor

import (
	"os"
	"path"
	"strings"
)

// skipMode describes how a matching directory should be handled.
type skipMode int

const (
	skipModeDrop skipMode = iota
	skipModeOpaque
	skipModePreserve
)

// skipRule combines a path pattern with its directory behavior.
type skipRule struct {
	pattern string
	mode    skipMode
}

// childDir carries metadata that controls how we recurse into a directory.
type childDir struct {
	name       string // Base name of the directory.
	skipUpdate bool   // True when the child kustomization must remain untouched.
	skipWalk   bool   // True when recursion into the directory should be skipped.
}

// parseSkipRules builds rules from the three explicit CLI behaviors. Rules are
// checked in this order, so skip wins over opaque, which wins over preserve.
func parseSkipRules(skip, opaque, preserve []string) []skipRule {
	rules := make([]skipRule, 0, len(skip)+len(opaque)+len(preserve))
	for _, pattern := range skip {
		rules = append(rules, skipRule{pattern: cleanSkipPattern(pattern), mode: skipModeDrop})
	}
	for _, pattern := range opaque {
		rules = append(rules, skipRule{pattern: cleanSkipPattern(pattern), mode: skipModeOpaque})
	}
	for _, pattern := range preserve {
		rules = append(rules, skipRule{pattern: cleanSkipPattern(pattern), mode: skipModePreserve})
	}
	return rules
}

func cleanSkipPattern(pattern string) string {
	return strings.Trim(strings.TrimSpace(pattern), "/")
}

// matchSkip determines whether rel matches a configured rule. Opaque and
// preserve apply only to directories; files continue through normal scanning.
func matchSkip(rel string, isDir bool, rules []skipRule) (skip bool, mode skipMode, pattern string) {
	for _, rule := range rules {
		if rule.pattern == "" || (!isDir && rule.mode != skipModeDrop) {
			continue
		}
		if matchesSkipPattern(rel, rule.pattern) {
			return true, rule.mode, rule.pattern
		}
	}
	return false, skipModeDrop, ""
}

// matchesSkipPattern matches patterns containing a slash against the path from
// the processing root. Patterns without a slash match basenames at any depth.
func matchesSkipPattern(rel, pattern string) bool {
	if strings.Contains(pattern, "/") {
		matched, err := path.Match(pattern, rel)
		return err == nil && matched
	}
	matched, err := path.Match(pattern, path.Base(rel))
	return err == nil && matched
}

// handleSkipDir records how a matched directory should adjust traversal.
func handleSkipDir(entry os.DirEntry, mode skipMode, dirEntries []string, childDirs []childDir) ([]string, []childDir) {
	name := entry.Name()
	switch mode {
	case skipModeDrop:
		return dirEntries, childDirs
	case skipModeOpaque:
		dirEntries = append(dirEntries, name)
		childDirs = append(childDirs, childDir{name: name, skipWalk: true})
		return dirEntries, childDirs
	case skipModePreserve:
		dirEntries = append(dirEntries, name)
		childDirs = append(childDirs, childDir{name: name, skipUpdate: true})
		return dirEntries, childDirs
	default:
		return dirEntries, childDirs
	}
}
