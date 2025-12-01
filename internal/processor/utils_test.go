package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsKustomization(t *testing.T) {
	t.Parallel()

	t.Run("yaml", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isKustomization("kustomization.yaml"))
	})

	t.Run("yml", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isKustomization("kustomization.yml"))
	})

	t.Run("notKustomization", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isKustomization("kustomization.txt"))
	})
}

func TestIsYAML(t *testing.T) {
	t.Parallel()

	t.Run("lowerYAML", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isYAML("resource.yaml"))
	})

	t.Run("upperYML", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isYAML("RESOURCE.YML"))
	})

	t.Run("notYAML", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isYAML("resource.txt"))
	})
}

func TestIsRemoteResource(t *testing.T) {
	t.Parallel()

	t.Run("http", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isRemoteResource("http://example.com"))
	})

	t.Run("https", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isRemoteResource("https://example.com"))
	})

	t.Run("notRemote", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isRemoteResource("file://local"))
	})
}

func TestEqualStrings(t *testing.T) {
	t.Parallel()

	t.Run("same", func(t *testing.T) {
		t.Parallel()
		require.True(t, equalStrings([]string{"a", "b"}, []string{"a", "b"}))
	})

	t.Run("differentLength", func(t *testing.T) {
		t.Parallel()
		require.False(t, equalStrings([]string{"a"}, []string{"a", "b"}))
	})

	t.Run("differentContent", func(t *testing.T) {
		t.Parallel()
		require.False(t, equalStrings([]string{"a", "b"}, []string{"b", "a"}))
	})
}
