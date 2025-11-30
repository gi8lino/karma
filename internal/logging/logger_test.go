package logging

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerMessageFormatting(t *testing.T) {
	t.Parallel()

	var out, errBuf bytes.Buffer
	logger := New(&out, &errBuf, LevelInfo)

	logger.Skipped("path", "foo", "reason", "bar")
	logger.Processing("dir", "path", "foo")
	logger.ResourceDiff([]string{"cluster.yaml"}, []string{})
	logger.Error("boom", "detail", "broken")

	stripped := stripANSI(out.String())
	assert.Contains(t, stripped, "[SKIPPING]")
	assert.Contains(t, stripped, "-resources:")
	assert.Contains(t, stripped, "+resources:")
	assert.Contains(t, stripped, "[PROCESS ]")

	assert.Contains(t, errBuf.String(), "[ERROR")
}

func stripANSI(input string) string {
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(input, "")
}
