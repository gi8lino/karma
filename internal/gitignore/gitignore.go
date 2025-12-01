package gitignore

import (
	"bufio"
	"errors"
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

type matcher struct {
	dir      string              // directory that owns this matcher.
	parent   *matcher            // parent matcher to inherit patterns.
	patterns []string            // collected patterns from this directory.
	children map[string]*matcher // cached child matchers.
}

// Load creates a matcher rooted at dir; returns nil if useGitignore is false.
func Load(dir string, useGitignore bool) (Matcher, error) {
	if !useGitignore {
		return nil, nil
	}
	return newMatcher(dir, nil)
}

func newMatcher(dir string, parent *matcher) (*matcher, error) {
	m := &matcher{
		dir:      dir,
		parent:   parent,
		children: make(map[string]*matcher),
	}

	// load the .gitignore file if it exists.
	path := filepath.Join(dir, ".gitignore")
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return nil, err
	}
	defer file.Close() // nolint:errcheck

	// parse the file into patterns.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m.patterns = append(m.patterns, line)
	}
	return m, scanner.Err()
}

func (m *matcher) Ignored(fullPath string, isDir bool) bool {
	if m == nil {
		return false
	}

	// compute the relative path to the matcher.
	rel, err := filepath.Rel(m.dir, fullPath)
	if err != nil {
		return m.parent.Ignored(fullPath, isDir)
	}

	// normalize the relative path to a slash-separated string.
	rel = filepath.ToSlash(rel)
	if rel == "." {
		rel = ""
	}

	for _, pattern := range m.patterns {
		// short-circuit when a pattern matches the relative path.
		if matchesPattern(rel, pattern, isDir) {
			return true
		}
	}

	// recurse into the parent matcher if we have one.
	if m.parent != nil {
		return m.parent.Ignored(fullPath, isDir)
	}
	return false
}

func (m *matcher) Child(dir string) (Matcher, error) {
	if m == nil {
		return newMatcher(dir, nil)
	}

	// reuse existing child matchers.
	if child, ok := m.children[dir]; ok {
		return child, nil
	}

	// create a new child matcher.
	child, err := newMatcher(dir, m)
	if err != nil {
		return nil, err
	}
	m.children[dir] = child

	return child, nil
}

// matchesPattern reports whether rel matches the pattern.
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

	if strings.ContainsAny(pattern, "*?[]") {
		matched, err := filepath.Match(pattern, rel)
		if err != nil {
			return false
		}
		return matched
	}
	return rel == pattern
}
