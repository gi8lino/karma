package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic(t *testing.T) {
	t.Parallel()

	t.Run("creates file with default permissions", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "kustomization.yaml")
		require.NoError(t, writeFileAtomic(path, []byte("new\n"), 0o644))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "new\n", string(data))
	})

	t.Run("preserves existing permissions", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
		require.NoError(t, writeFileAtomic(path, []byte("new\n"), 0o644))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})
}
