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
	Ignored(fullPath string, isDir bool) bool
	Child(dir string) (Matcher, error)
}

type matcher struct {
	dir      string
	parent   *matcher
	patterns []string
	children map[string]*matcher
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
	path := filepath.Join(dir, ".gitignore")
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
	rel, err := filepath.Rel(m.dir, fullPath)
	if err != nil {
		return m.parent.Ignored(fullPath, isDir)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		rel = ""
	}

	for _, pattern := range m.patterns {
		if matchesPattern(rel, pattern, isDir) {
			return true
		}
	}
	if m.parent != nil {
		return m.parent.Ignored(fullPath, isDir)
	}
	return false
}

func (m *matcher) Child(dir string) (Matcher, error) {
	if m == nil {
		return newMatcher(dir, nil)
	}
	if child, ok := m.children[dir]; ok {
		return child, nil
	}
	child, err := newMatcher(dir, m)
	if err != nil {
		return nil, err
	}
	m.children[dir] = child
	return child, nil
}

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
