package gitignore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatcherIgnoresPattern(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignore.yaml\nsubdir/\n"), 0o600))

	matcher, err := Load(dir, true)
	require.NoError(t, err)
	require.NotNil(t, matcher)

	require.True(t, matcher.Ignored(filepath.Join(dir, "ignore.yaml"), false))
	require.True(t, matcher.Ignored(filepath.Join(dir, "subdir"), true))
}
