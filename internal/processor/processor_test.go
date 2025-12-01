package processor

import (
	"io"
	"testing"

	"github.com/gi8lino/kustomizer/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchSkipPatterns(t *testing.T) {
	t.Parallel()

	proc := New(Options{Skip: []string{"flux/config/**", "flux/*", "plain"}}, logging.New(io.Discard, io.Discard, 0))

	skip, mode, pattern := matchSkip("flux/config/app", false, proc.skipRules)
	require.True(t, skip)
	assert.Equal(t, skipModeSubtree, mode)
	assert.Equal(t, "flux/config/**", pattern)

	skip, mode, _ = matchSkip("flux/ignore-me", false, proc.skipRules)
	require.True(t, skip)
	assert.Equal(t, skipModeChildren, mode)

	skip, mode, pattern = matchSkip("plain", false, proc.skipRules)
	require.True(t, skip)
	assert.Equal(t, skipModeExact, mode)
	assert.Equal(t, "plain", pattern)
}

func TestMergeResourcesOrders(t *testing.T) {
	t.Parallel()

	logger := logging.New(io.Discard, io.Discard, 0)
	proc := New(Options{DirSlash: true, DirFirst: true}, logger)

	final := proc.mergeResources([]string{"https://example.com"}, []string{"b", "a"}, []string{"z", "y"})
	require.Equal(t, []string{"https://example.com", "a/", "b/", "y", "z"}, final)
}
