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
			"--suffix",
			"--prefix",
			"-q",
			"foo",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"foo"}, cfg.BaseDirs)
		assert.Equal(t, []string{".img", "dashboards", "patch-*"}, cfg.SkipPatterns)
		require.False(t, cfg.UseGitIgnore)
		require.True(t, cfg.IncludeDot)
		require.True(t, cfg.AddDirSuffix)
		require.True(t, cfg.AddDirPrefix)
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
		require.True(t, cfg.UseGitIgnore)
		require.False(t, cfg.IncludeDot)
		require.False(t, cfg.AddDirSuffix)
		require.False(t, cfg.AddDirPrefix)
		require.False(t, cfg.DryRun)
		require.False(t, cfg.Check)
	})

	t.Run("order flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"--order", "external,files,dirs", "foo"})
		require.NoError(t, err)
		require.Equal(t, []string{"external", "files", "dirs"}, cfg.ResourceOrder)
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

	t.Run("wrong order flag", func(t *testing.T) {
		t.Parallel()
		_, err := Parse("1.0.0", []string{"--order", "foo"})
		require.Error(t, err)
		assert.EqualError(t, err, "invalid value for flag --order: invalid resource order item: foo. allowed are: external, dirs, files.")
	})

	t.Run("empty order flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"--order", "files,dirs,,external", "positional"})
		require.NoError(t, err)
		assert.Equal(t, []string{"files", "dirs", "external"}, cfg.ResourceOrder)
	})
	t.Run("dry run", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"--dry-run", "foo"})
		require.NoError(t, err)
		assert.True(t, cfg.DryRun)
		assert.False(t, cfg.Check)
	})

	t.Run("check", func(t *testing.T) {
		t.Parallel()
		cfg, err := Parse("1.0.0", []string{"--check", "foo"})
		require.NoError(t, err)
		assert.True(t, cfg.Check)
		assert.False(t, cfg.DryRun)
	})

	t.Run("dry run and check conflict", func(t *testing.T) {
		t.Parallel()
		_, err := Parse("1.0.0", []string{"--dry-run", "--check", "foo"})
		require.Error(t, err)
	})

}
