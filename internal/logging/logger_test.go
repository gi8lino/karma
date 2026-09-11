package logging

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessing(t *testing.T) {
	t.Parallel()

	t.Run("processing", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelInfo)
		logger.Processing("base", "path", "/tmp")
		assert.Contains(t, out.String(), "[PROCESS ]")
		assert.NotContains(t, out.String(), "\x1b[")
	})
}

func TestSkipping(t *testing.T) {
	t.Parallel()

	t.Run("skipping", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelDebug)
		logger.Skipped("path", "flux/config")
		assert.Contains(t, out.String(), "[SKIPPING]")
	})

	t.Run("no skip at info", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelInfo)
		logger.Skipped("path", "flux/config")
		assert.Empty(t, out.String())
	})
}

func TestUpdated(t *testing.T) {
	t.Parallel()

	t.Run("updated", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelInfo)
		logger.Updated("/tmp/kustomization.yaml")
		assert.Contains(t, out.String(), "[UPDATED ]")
	})
}

func TestNoOp(t *testing.T) {
	t.Parallel()

	t.Run("noop", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelDebug)
		logger.NoOp("/tmp/kustomization.yaml")
		assert.Contains(t, out.String(), "[NO-OP   ]")
	})

	t.Run("no noop at verbose", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelVerbose)
		logger.NoOp("/tmp/kustomization.yaml")
		assert.Empty(t, out.String())
	})
}

func TestDebugKV(t *testing.T) {
	t.Parallel()

	t.Run("kv only", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		logger := New(buf, LevelDebug)
		logger.DebugKV("foo", "bar", "baz", "qux")
		got := buf.String()
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
		logger := New(out, LevelTrace)
		logger.Trace("details", "foo", "bar")
		assert.Contains(t, out.String(), "[TRACE   ]")
	})
}

func TestSummary(t *testing.T) {
	t.Parallel()

	t.Run("summary", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelInfo)
		logger.Summary(2, 1, 0, 0, 0)
		assert.Contains(t, out.String(), "[SUMMARY ]")
	})
}

func TestResourceDiff(t *testing.T) {
	t.Parallel()

	t.Run("diffs", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelVerbose)
		logger.ResourceDiff([]string{"app", "old"}, []string{"app", "new"})
		stripped := out.String()
		assert.NotContains(t, out.String(), "\x1b[")
		require.Contains(t, stripped, "+  - \"new\"")
		require.Contains(t, stripped, "-  - \"old\"")
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelVerbose)
		logger.ResourceDiff([]string{}, []string{})
		require.Empty(t, out.String())
	})

	t.Run("info level hides diff", func(t *testing.T) {
		t.Parallel()
		out := &bytes.Buffer{}
		logger := New(out, LevelInfo)
		logger.ResourceDiff([]string{"app"}, []string{"app", "new"})
		assert.Empty(t, out.String())
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

func TestShouldColor(t *testing.T) {
	t.Run("buffer is plain", func(t *testing.T) {
		assert.False(t, shouldColor(&bytes.Buffer{}))
	})

	t.Run("regular file is plain", func(t *testing.T) {
		file, err := os.CreateTemp(t.TempDir(), "log-*")
		require.NoError(t, err)
		t.Cleanup(func() { _ = file.Close() })
		assert.False(t, shouldColor(file))
	})

	t.Run("no color disables ansi", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		assert.False(t, shouldColor(os.Stdout))
	})
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("even key values", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		logger := New(nil, LevelInfo)
		logger.write(buf, "UPDATED", []string{"kustomization", "/tmp/kustomization.yaml"})
		got := buf.String()
		assert.Contains(t, got, "[UPDATED ]")
		assert.Contains(t, got, "kustomization=/tmp/kustomization.yaml")
	})

	t.Run("odd key list writes bare value", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		logger := New(nil, LevelInfo)
		logger.write(buf, "SUMMARY", []string{"updated", "1", "no-op"})
		got := buf.String()
		assert.Contains(t, got, "[SUMMARY ]")
		assert.Contains(t, got, "updated=1")
		assert.Contains(t, got, "no-op")
	})
}
