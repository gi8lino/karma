package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run("not kustomization", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isKustomization("kustomization.txt"))
	})
}

func TestIsYAML(t *testing.T) {
	t.Parallel()

	t.Run("lower yaml", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isYAML("resource.yaml"))
	})

	t.Run("upper yml", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isYAML("RESOURCE.YML"))
	})

	t.Run("not yaml", func(t *testing.T) {
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

	t.Run("not remote", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isRemoteResource("file://local"))
	})
}
