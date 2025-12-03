package gitignore

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Matcher decides if a path is ignored based on stacked rules.
type Matcher interface {
	// Ignored reports whether the given path matches any loaded patterns.
	Ignored(fullPath string, isDir bool) bool
	// Child loads or reuses the matcher for a subdirectory.
	Child(dir string) (Matcher, error)
}

// Matcher implementation stores the directory-specific state required for path matching.
type matcher struct {
	dir      string              // Directory that owns this matcher.
	parent   *matcher            // Parent matcher to inherit patterns.
	patterns []string            // Collected patterns from this directory.
	children map[string]*matcher // Cached child matchers.
}

// Load creates a matcher rooted at dir; returns nil if useGitignore is false.
func Load(dir string, useGitignore bool) (Matcher, error) {
	if !useGitignore {
		return nil, nil
	}
	return newMatcher(dir, nil)
}

// Creates a matcher rooted at dir; returns nil if dir does not exist.
func newMatcher(dir string, parent *matcher) (*matcher, error) {
	m := &matcher{
		dir:      dir,
		parent:   parent,
		children: make(map[string]*matcher),
	}

	// Load the .gitignore file if it exists.
	path := filepath.Join(dir, ".gitignore")
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return nil, err
	}
	defer file.Close() // nolint:errcheck

	// Parse the file into patterns.
	patterns, err := parseGitignore(file)
	if err != nil {
		return nil, err
	}
	m.patterns = patterns
	return m, nil
}

// Ignored reports whether the given path matches any loaded patterns.
func (m *matcher) Ignored(fullPath string, isDir bool) bool {
	if m == nil {
		return false
	}

	// Compute the relative path to the matcher.
	rel, err := filepath.Rel(m.dir, fullPath)
	if err != nil {
		return m.parent.Ignored(fullPath, isDir)
	}

	// Normalize the relative path to a slash-separated string.
	rel = filepath.ToSlash(rel)
	if rel == "." {
		rel = ""
	}

	for _, pattern := range m.patterns {
		// Short-circuit when a pattern matches the relative path.
		if matchesPattern(rel, pattern, isDir) {
			return true
		}
	}

	// Recurse into the parent matcher if we have one.
	if m.parent != nil {
		return m.parent.Ignored(fullPath, isDir)
	}
	return false
}

// Child loads or reuses the matcher for a subdirectory.
func (m *matcher) Child(dir string) (Matcher, error) {
	if m == nil {
		return newMatcher(dir, nil)
	}

	// Reuse existing child matchers.
	if child, ok := m.children[dir]; ok {
		return child, nil
	}

	// Create a new child matcher.
	child, err := newMatcher(dir, m)
	if err != nil {
		return nil, err
	}
	m.children[dir] = child

	return child, nil
}

// ParseGitignore reads patterns from the provided reader.
func parseGitignore(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var patterns []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}

// MatchesPattern reports whether rel matches the pattern.
func matchesPattern(rel, pattern string, isDir bool) bool {
	if pattern == "" {
		return false
	}
	if strings.HasSuffix(pattern, "/") {
		if !isDir {
			return false
		}
		pattern = strings.TrimSuffix(pattern, "/")
	}
	if pattern == "" {
		return true
	}

	// Check for glob patterns.
	if strings.ContainsAny(pattern, "*?[]") {
		matched, err := filepath.Match(pattern, rel)
		if err != nil {
			return false
		}
		return matched
	}
	return rel == pattern
}
