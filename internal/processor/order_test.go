package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResourceOrder(t *testing.T) {
	t.Parallel()

	t.Run("default order", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"remote", "dirs", "files"}, ParseResourceOrder(""))
	})

	t.Run("partial order appends missing groups", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"remote", "files", "dirs"}, ParseResourceOrder("remote,files"))
	})

	t.Run("dedups invalid entries", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"remote", "dirs", "files"}, ParseResourceOrder("remote,remote,invalid"))
	})
}

func TestNormalizeResourceOrder(t *testing.T) {
	t.Parallel()

	t.Run("empty slice returns default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"remote", "dirs", "files"}, normalizeResourceOrder([]string{}))
	})

	t.Run("ignores unknown entries and trims whitespace", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"  DIRS", "foo", "FILES"})
		assert.Equal(t, []string{"dirs", "files", "remote"}, got)
	})

	t.Run("dedups repeated entries", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"remote", "remote", "dirs"})
		assert.Equal(t, []string{"remote", "dirs", "files"}, got)
	})

	t.Run("maintains custom order when valid", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"files", "remote"})
		assert.Equal(t, []string{"files", "remote", "dirs"}, got)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"remote", "remote", "", "dirs"})
		assert.Equal(t, []string{"remote", "dirs", "files"}, got)
	})
}
