package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	cfg, err := Parse("1.0.0", []string{
		"-s", ".img,dashboards",
		"-s", "patch-*",
		"-v",
		"--no-gitignore",
		"--include-dot",
		"--no-dir-slash",
		"--no-dir-first",
		"foo",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo"}, cfg.BaseDirs)
	assert.Equal(t, []string{".img", "dashboards", "patch-*"}, cfg.SkipPatterns)
	assert.Equal(t, 1, cfg.Verbosity)
	assert.True(t, cfg.NoGitIgnore)
	assert.True(t, cfg.IncludeDot)
	assert.True(t, cfg.NoDirSlash)
	assert.True(t, cfg.NoDirFirst)
}
