package logging

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stripANSI(t *testing.T, input string) string {
	t.Helper()
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(input, "")
}

func TestProcessingLogsWithLevel(t *testing.T) {
	t.Parallel()

	t.Run("processing", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Processing("base", "path", "/tmp")
		assert.Contains(t, stripANSI(t, out.String()), "[PROCESS ]")
	})
}

func TestSkippingLogsWhenEnabled(t *testing.T) {
	t.Parallel()

	t.Run("skipping", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Skipped("path", "flux/config")
		assert.Contains(t, stripANSI(t, out.String()), "[SKIPPING]")
	})
}

func TestUpdatedLogsAlways(t *testing.T) {
	t.Parallel()

	t.Run("updated", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Updated("/tmp/kustomization.yaml")
		assert.Contains(t, stripANSI(t, out.String()), "[UPDATED ]")
	})
}

func TestNoOpLogsWithLevel(t *testing.T) {
	t.Parallel()

	t.Run("noop", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.NoOp("/tmp/kustomization.yaml")
		assert.Contains(t, stripANSI(t, out.String()), "[NO-OP   ]")
	})
}

func TestDebugLogsWhenVerbose(t *testing.T) {
	t.Parallel()

	t.Run("debug", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelDebug)
		logger.Debug("details", "foo", "bar")
		assert.Contains(t, stripANSI(t, out.String()), "[DEBUG   ]")
	})
}

func TestTraceLogsWhenTraceLevel(t *testing.T) {
	t.Parallel()

	t.Run("trace", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelTrace)
		logger.Trace("details", "foo", "bar")
		assert.Contains(t, stripANSI(t, out.String()), "[TRACE   ]")
	})
}

func TestSummaryAlwaysLogs(t *testing.T) {
	t.Parallel()

	t.Run("summary", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Summary(2, 1)
		assert.Contains(t, stripANSI(t, out.String()), "[SUMMARY ]")
	})
}

func TestErrorLogsToErrWriter(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		errBuf := &bytes.Buffer{}
		logger := New(nil, errBuf, LevelError)
		logger.Error("boom", "detail", "failed")
		assert.Contains(t, errBuf.String(), "[ERROR")
	})
}

func TestResourceDiffShowsChanges(t *testing.T) {
	t.Parallel()

	t.Run("resource diff", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.ResourceDiff([]string{"app"}, []string{"app", "new"})
		stripped := stripANSI(t, out.String())
		assert.NotContains(t, stripped, "-resources")
		require.Contains(t, stripped, "+  - new")
	})
}

func TestResourceDiffSkipWhenNoChange(t *testing.T) {
	t.Parallel()

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.ResourceDiff([]string{"app"}, []string{"app"})
		assert.Empty(t, stripANSI(t, out.String()))
	})
}
