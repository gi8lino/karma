package gitignore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gitignorepkg "github.com/git-pkgs/gitignore"
)

// Matcher decides if a path is ignored by .gitignore rules below a root.
type Matcher interface {
	Ignored(fullPath string, isDir bool) bool
}

type matcher struct {
	root    string
	matcher *gitignorepkg.Matcher
}

// Load creates a matcher rooted at dir. Only .gitignore files below dir are
// loaded so results do not depend on a user's global Git configuration.
func Load(dir string, useGitignore bool) (Matcher, error) {
	if !useGitignore {
		return nil, nil
	}

	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve gitignore root: %w", err)
	}

	compiled := gitignorepkg.New("")
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Name() != ".gitignore" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("resolve gitignore scope: %w", err)
		}
		if relDir == "." {
			relDir = ""
		}
		compiled.AddPatterns(data, filepath.ToSlash(relDir))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if errs := compiled.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("invalid .gitignore pattern: %w", errs[0])
	}

	return &matcher{root: root, matcher: compiled}, nil
}

// Ignored reports whether fullPath is ignored by the loaded rules.
func (m *matcher) Ignored(fullPath string, isDir bool) bool {
	if m == nil {
		return false
	}

	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(m.root, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return m.matcher.MatchPath(filepath.ToSlash(rel), isDir)
}
