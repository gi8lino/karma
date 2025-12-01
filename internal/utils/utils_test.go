package utils_test

import (
	"testing"

	"github.com/gi8lino/karma/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestDedupPreserve(t *testing.T) {
	t.Parallel()
	t.Run("removes duplicates and preserves order", func(t *testing.T) {
		t.Parallel()
		in := []string{"a", "b", "a", "c", "b"}
		got := utils.DedupPreserve(in)
		want := []string{"a", "b", "c"}
		assert.Equal(t, want, got)
	})
}
