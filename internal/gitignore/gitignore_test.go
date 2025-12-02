package gitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("returns nil matcher when disabled", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		matcher, err := Load(dir, false)
		require.NoError(t, err)
		assert.Nil(t, matcher)
	})

	t.Run("parses patterns from .gitignore", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignore.yaml\nsubdir/\n"), 0o600))

		matcher, err := Load(dir, true)
		require.NoError(t, err)
		require.NotNil(t, matcher)
		assert.True(t, matcher.Ignored(filepath.Join(dir, "ignore.yaml"), false))
		assert.True(t, matcher.Ignored(filepath.Join(dir, "subdir"), true))
	})
}

func TestNewMatcher(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("child/cache.tmp"), 0o600))
}

func TestMatcherChild(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("child/cache.tmp\n"), 0o600))
	childDir := filepath.Join(dir, "child")
	require.NoError(t, os.Mkdir(childDir, 0o755))

	parent, err := Load(dir, true)
	require.NoError(t, err)
	require.NotNil(t, parent)

	child, err := parent.Child(childDir)
	require.NoError(t, err)
	require.NotNil(t, child)

	t.Run("inherits parent patterns", func(t *testing.T) {
		t.Parallel()
		assert.True(t, child.Ignored(filepath.Join(childDir, "cache.tmp"), false))
	})

	t.Run("allows unique child patterns", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, os.WriteFile(filepath.Join(childDir, ".gitignore"), []byte("child.txt\n"), 0o600))
		childWithPattern, err := child.Child(childDir)
		require.NoError(t, err)
		assert.True(t, childWithPattern.Ignored(filepath.Join(childDir, "child.txt"), false))
	})
}

func TestMatchesPattern(t *testing.T) {
	t.Parallel()

	t.Run("matches exact path", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesPattern("app.yaml", "app.yaml", false))
		assert.False(t, matchesPattern("app.yaml", "other.yaml", false))
	})

	t.Run("handles directory suffixes", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesPattern("config", "config/", true))
		assert.False(t, matchesPattern("config/file", "config/", true))
		assert.False(t, matchesPattern("config", "config/", false))
	})

	t.Run("supports globbing", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesPattern("docs/guide.md", "docs/*.md", false))
		assert.False(t, matchesPattern("docs/guide.md", "src/*.md", false))
	})

	t.Run("fails gracefully on invalid patterns", func(t *testing.T) {
		t.Parallel()
		assert.False(t, matchesPattern("path", "[invalid", false))
	})
}

func TestParseGitignore(t *testing.T) {
	t.Parallel()

	t.Run("skips comments and empty lines", func(t *testing.T) {
		t.Parallel()
		content := "#comment\n\n# another comment\nkeep.yaml"
		patterns, err := parseGitignore(strings.NewReader(content))
		require.NoError(t, err)
		assert.Equal(t, []string{"keep.yaml"}, patterns)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()
		content := "  spaced.yaml  \n\t#ignored\n"
		patterns, err := parseGitignore(strings.NewReader(content))
		require.NoError(t, err)
		assert.Equal(t, []string{"spaced.yaml"}, patterns)
	})
}
