package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerMessageFormatting(t *testing.T) {
	t.Parallel()

	var out, errBuf bytes.Buffer
	logger := New(&out, &errBuf, 1)

	logger.Skipped("path", "foo", "reason", "bar")
	logger.Processing("dir", "path", "foo")
	logger.Error("boom", "detail", "broken")

	assert.Contains(t, out.String(), "[SKIPPING] path=foo reason=bar")
	assert.Contains(t, out.String(), "[PROCESS ] kind=dir")
	assert.Contains(t, errBuf.String(), "[ERROR   ] message=boom detail=broken")
}
