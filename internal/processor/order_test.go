package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultResourceOrder(t *testing.T) {
	t.Parallel()

	t.Run("returns default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"external", "dirs", "files"}, DefaultResourceOrder())
	})
}

func TestParseResourceOrder(t *testing.T) {
	t.Parallel()

	t.Run("default order", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"external", "dirs", "files"}, ParseResourceOrder(""))
	})

	t.Run("partial order appends missing groups", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"external", "files", "dirs"}, ParseResourceOrder("external,files"))
	})

	t.Run("dedups invalid entries", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"external", "dirs", "files"}, ParseResourceOrder("remote,remote,invalid"))
	})
}

func TestNormalizeResourceOrder(t *testing.T) {
	t.Parallel()

	t.Run("empty slice returns default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"external", "dirs", "files"}, normalizeResourceOrder([]string{}))
	})

	t.Run("ignores unknown entries and trims whitespace", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"  DIRS", "foo", "FILES"})
		assert.Equal(t, []string{"dirs", "files", "external"}, got)
	})

	t.Run("dedups repeated entries", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"external", "external", "dirs"})
		assert.Equal(t, []string{"external", "dirs", "files"}, got)
	})

	t.Run("maintains custom order when valid", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"files", "remote"})
		assert.Equal(t, []string{"files", "external", "dirs"}, got)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		got := normalizeResourceOrder([]string{"external", "external", "", "dirs"})
		assert.Equal(t, []string{"external", "dirs", "files"}, got)
	})
}
