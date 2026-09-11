package gitignore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("returns nil matcher when disabled", func(t *testing.T) {
		t.Parallel()
		matcher, err := Load(t.TempDir(), false)
		require.NoError(t, err)
		assert.Nil(t, matcher)
	})

	t.Run("supports gitignore semantics", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.tmp\n!keep.tmp\ncache/\n**/generated/*.yaml\n"), 0o600))

		matcher, err := Load(root, true)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		assert.True(t, matcher.Ignored(filepath.Join(root, "drop.tmp"), false))
		assert.False(t, matcher.Ignored(filepath.Join(root, "keep.tmp"), false))
		assert.True(t, matcher.Ignored(filepath.Join(root, "cache"), true))
		assert.True(t, matcher.Ignored(filepath.Join(root, "nested", "generated", "app.yaml"), false))
	})

	t.Run("scopes nested gitignore files", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		sub := filepath.Join(root, "sub")
		require.NoError(t, os.Mkdir(sub, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.yaml\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(sub, ".gitignore"), []byte("!keep.yaml\nlocal.tmp\n"), 0o600))

		matcher, err := Load(root, true)
		require.NoError(t, err)

		assert.True(t, matcher.Ignored(filepath.Join(sub, "drop.yaml"), false))
		assert.False(t, matcher.Ignored(filepath.Join(sub, "keep.yaml"), false))
		assert.True(t, matcher.Ignored(filepath.Join(sub, "local.tmp"), false))
		assert.False(t, matcher.Ignored(filepath.Join(root, "local.tmp"), false))
	})
}
