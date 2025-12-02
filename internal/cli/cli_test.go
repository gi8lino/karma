package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("full options", func(t *testing.T) {
		t.Parallel()

		cfg, err := Parse("1.0.0", []string{
			"-s", ".img,dashboards",
			"-s", "patch-*",
			"--no-gitignore",
			"--include-dot",
			"--no-dir-slash",
			"-q",
			"foo",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"foo"}, cfg.BaseDirs)
		assert.Equal(t, []string{".img", "dashboards", "patch-*"}, cfg.SkipPatterns)
		require.True(t, cfg.NoGitIgnore)
		require.True(t, cfg.IncludeDot)
		require.True(t, cfg.NoDirSlash)
		require.True(t, cfg.Mute)
		assert.Equal(t, -1, cfg.Verbosity, "mute should set verbosity to -1 via finalizer")
	})

	t.Run("verbose flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"-vv", "foo"})
		require.NoError(t, err)
		assert.Equal(t, 2, cfg.Verbosity)
	})

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		cfg, err := Parse("1.0.0", []string{"bar"})
		require.NoError(t, err)
		assert.Equal(t, []string{"bar"}, cfg.BaseDirs)
		assert.Equal(t, []string{}, cfg.SkipPatterns)
		assert.Zero(t, cfg.Verbosity)
		require.False(t, cfg.NoGitIgnore)
		require.False(t, cfg.IncludeDot)
		require.False(t, cfg.NoDirSlash)
	})

	t.Run("order flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"--order", "remote,files,dirs", "foo"})
		require.NoError(t, err)
		require.Equal(t, []string{"remote", "files", "dirs"}, cfg.ResourceOrder)
	})

	t.Run("missing positional", func(t *testing.T) {
		t.Parallel()

		_, err := Parse("1.0.0", []string{})
		require.Error(t, err)
	})

	t.Run("help", func(t *testing.T) {
		t.Parallel()

		_, err := Parse("1.0.0", []string{"--help"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Usage")
	})

	t.Run("mute flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"-q", "foo"})
		require.NoError(t, err)
		assert.True(t, cfg.Mute)
		assert.Equal(t, -1, cfg.Verbosity, "mute should set verbosity to -1 via finalizer")
	})
}
