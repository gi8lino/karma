package processor

import (
	"io"
	"testing"

	"github.com/containeroo/kustomizer/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchSkipPatterns(t *testing.T) {
	t.Parallel()

	proc := New(Options{Skip: []string{"flux/config/**", "flux/*", "plain"}}, logging.New(io.Discard, io.Discard, 0))

	skip, subtree, pattern := proc.matchSkip("flux/config/app", true)
	require.True(t, skip)
	assert.True(t, subtree)
	assert.Equal(t, "flux/config/**", pattern)

	skip, subtree, _ = proc.matchSkip("flux/ignore-me", false)
	require.True(t, skip)
	assert.False(t, subtree)

	skip, _, pattern = proc.matchSkip("plain", false)
	require.True(t, skip)
	assert.Equal(t, "plain", pattern)
}

func TestMergeResourcesOrders(t *testing.T) {
	t.Parallel()

	logger := logging.New(io.Discard, io.Discard, 0)
	proc := New(Options{DirSlash: true, DirFirst: true}, logger)

	final := proc.mergeResources([]string{"https://example.com"}, []string{"b", "a"}, []string{"z", "y"})
	require.Equal(t, []string{"https://example.com", "a/", "b/", "y", "z"}, final)
}
