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
