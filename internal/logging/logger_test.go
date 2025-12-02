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

func TestProcessing(t *testing.T) {
	t.Parallel()

	t.Run("processing", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Processing("base", "path", "/tmp")
		assert.Contains(t, stripANSI(t, out.String()), "[PROCESS ]")
	})
}

func TestSkipping(t *testing.T) {
	t.Parallel()

	t.Run("skipping", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelDebug)
		logger.Skipped("path", "flux/config")
		assert.Contains(t, stripANSI(t, out.String()), "[SKIPPING]")
	})

	t.Run("no skip at info", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Skipped("path", "flux/config")
		assert.Empty(t, stripANSI(t, out.String()))
	})
}

func TestUpdated(t *testing.T) {
	t.Parallel()

	t.Run("updated", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Updated("/tmp/kustomization.yaml")
		assert.Contains(t, stripANSI(t, out.String()), "[UPDATED ]")
	})
}

func TestNoOp(t *testing.T) {
	t.Parallel()

	t.Run("noop", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelDebug)
		logger.NoOp("/tmp/kustomization.yaml")
		assert.Contains(t, stripANSI(t, out.String()), "[NO-OP   ]")
	})

	t.Run("no noop at verbose", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelVerbose)
		logger.NoOp("/tmp/kustomization.yaml")
		assert.Empty(t, stripANSI(t, out.String()))
	})
}

func TestDebug(t *testing.T) {
	t.Parallel()

	t.Run("debug", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelDebug)
		logger.Debug("details", "foo", "bar")
		assert.Contains(t, stripANSI(t, out.String()), "[DEBUG   ]")
	})
}

func TestDebugKV(t *testing.T) {
	t.Parallel()

	t.Run("kv only", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		logger := New(buf, nil, LevelDebug)
		logger.DebugKV("foo", "bar", "baz", "qux")
		got := stripANSI(t, buf.String())
		assert.Contains(t, got, "foo=bar")
		assert.Contains(t, got, "baz=qux")
		assert.NotContains(t, got, "message=")
	})
}

func TestTrace(t *testing.T) {
	t.Parallel()

	t.Run("trace", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelTrace)
		logger.Trace("details", "foo", "bar")
		assert.Contains(t, stripANSI(t, out.String()), "[TRACE   ]")
	})
}

func TestSummary(t *testing.T) {
	t.Parallel()

	t.Run("summary", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.Summary(2, 1, 0, 0, 0)
		assert.Contains(t, stripANSI(t, out.String()), "[SUMMARY ]")
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		errBuf := &bytes.Buffer{}
		logger := New(nil, errBuf, LevelError)
		logger.Error("boom", "detail", "failed")
		assert.Contains(t, errBuf.String(), "[ERROR")
	})
}

func TestResourceDiff(t *testing.T) {
	t.Parallel()

	t.Run("diffs", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelVerbose)
		logger.ResourceDiff([]string{"app", "old"}, []string{"app", "new"})
		stripped := stripANSI(t, out.String())
		require.Contains(t, stripped, "+  - \"new\"")
		require.Contains(t, stripped, "-  - \"old\"")
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelVerbose)
		logger.ResourceDiff([]string{}, []string{})
		require.Empty(t, out.String())
	})

	t.Run("info level hides diff", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, nil, LevelInfo)
		logger.ResourceDiff([]string{"app"}, []string{"app", "new"})
		assert.Empty(t, stripANSI(t, out.String()))
	})
}

func TestLevelFromVerbosity(t *testing.T) {
	t.Parallel()

	t.Run("mute takes precedence", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, LevelOff, LevelFromVerbosity(-1))
	})

	t.Run("verbosity honors trace", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, LevelTrace, LevelFromVerbosity(3))
	})

	t.Run("verbosity honors debug", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, LevelDebug, LevelFromVerbosity(2))
	})

	t.Run("verbosity uses verbose level for -v", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, LevelVerbose, LevelFromVerbosity(1))
	})

	t.Run("verbosity defaults to info", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, LevelInfo, LevelFromVerbosity(0))
	})
}

func TestDiffStrings(t *testing.T) {
	t.Parallel()

	t.Run("detects added and removed", func(t *testing.T) {
		t.Parallel()
		removed, added := diffStrings([]string{"a", "b"}, []string{"b", "c"})
		assert.Equal(t, []string{"a"}, removed)
		assert.Equal(t, []string{"c"}, added)
	})

	t.Run("respects duplicates", func(t *testing.T) {
		t.Parallel()
		removed, added := diffStrings([]string{"a", "a", "b"}, []string{"a", "c", "a"})
		assert.ElementsMatch(t, []string{"b"}, removed)
		assert.ElementsMatch(t, []string{"c"}, added)
	})
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("even key values", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		logger := New(nil, nil, LevelInfo)
		logger.write(buf, "UPDATED", []string{"kustomization", "/tmp/kustomization.yaml"})
		got := stripANSI(t, buf.String())
		assert.Contains(t, got, "[UPDATED ]")
		assert.Contains(t, got, "kustomization=/tmp/kustomization.yaml")
	})

	t.Run("odd key list writes bare value", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		logger := New(nil, nil, LevelInfo)
		logger.write(buf, "SUMMARY", []string{"updated", "1", "no-op"})
		got := stripANSI(t, buf.String())
		assert.Contains(t, got, "[SUMMARY ]")
		assert.Contains(t, got, "updated=1")
		assert.Contains(t, got, "no-op")
	})
}
