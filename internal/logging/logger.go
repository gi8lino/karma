package logging

import (
	"fmt"
	"io"
	"strings"
)

const (
	colorReset  = "\x1b[0m"
	colorRed    = "\x1b[31m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorBlue   = "\x1b[34m"
	colorPurple = "\x1b[35m"
	colorCyan   = "\x1b[36m"
)

var tagColors = map[string]string{
	"PROCESS":  colorCyan,
	"SKIPPING": colorYellow,
	"UPDATED":  colorGreen,
	"NO-OP":    colorBlue,
	"TRACE":    colorPurple,
	"SUMMARY":  colorGreen,
	"ERROR":    colorRed,
	"DEBUG":    colorPurple,
}

// LogLevel defines how verbose the logger should be.
type LogLevel int

const (
	LevelOff LogLevel = iota
	LevelError
	LevelInfo
	LevelDebug
	LevelTrace
)

// LevelFromVerbosity maps a CLI verbosity counter to an internal log level.
func LevelFromVerbosity(v int) LogLevel {
	switch {
	case v >= 2:
		return LevelTrace
	case v == 1:
		return LevelDebug
	default:
		return LevelInfo
	}
}

// Logger formats CLI output with output streams and a minimum log level.
type Logger struct {
	out      io.Writer
	err      io.Writer
	minLevel LogLevel
}

// New creates a logger that renders on the provided writers.
func New(out, err io.Writer, level LogLevel) *Logger {
	return &Logger{out: out, err: err, minLevel: level}
}

// Flush exists for symmetry with buffered I/O.
func (l *Logger) Flush() {}

// Processing logs the current directory when the console level allows it.
func (l *Logger) Processing(kind string, kv ...string) {
	l.log(LevelInfo, "PROCESS", func() []string {
		return append([]string{"kind", kind}, kv...)
	})
}

// Skipped logs resources that were skipped because of a rule.
func (l *Logger) Skipped(kv ...string) {
	l.log(LevelInfo, "SKIPPING", func() []string {
		return kv
	})
}

// Updated logs that a kustomization was rewritten.
func (l *Logger) Updated(path string, kv ...string) {
	l.log(LevelInfo, "UPDATED", func() []string {
		return append([]string{"kustomization", path}, kv...)
	})
}

// NoOp logs that a kustomization was already in sync.
func (l *Logger) NoOp(path string, kv ...string) {
	l.log(LevelInfo, "NO-OP", func() []string {
		return append([]string{"kustomization", path}, kv...)
	})
}

// Debug logs debug-level output when the log level permits.
func (l *Logger) Debug(msg string, kv ...string) {
	l.log(LevelDebug, "DEBUG", func() []string {
		return append([]string{"message", msg}, kv...)
	})
}

// Trace logs trace-level output when the log level permits.
func (l *Logger) Trace(msg string, kv ...string) {
	l.log(LevelTrace, "TRACE", func() []string {
		return append([]string{"message", msg}, kv...)
	})
}

// Summary prints the overall update statistics.
func (l *Logger) Summary(updated, noOp int) {
	l.log(LevelInfo, "SUMMARY", func() []string {
		return []string{"updated", fmt.Sprintf("%d", updated), "no-op", fmt.Sprintf("%d", noOp)}
	})
}

// Error logs an error to stderr regardless of verbosity.
func (l *Logger) Error(msg string, kv ...string) {
	l.log(LevelError, "ERROR", func() []string {
		return append([]string{"message", msg}, kv...)
	})
}

// ResourceDiff prints an old/new snapshot of the resources block.
func (l *Logger) ResourceDiff(old, new []string) {
	if l.minLevel < LevelInfo {
		return
	}
	oldLines := resourceBlockLines(old)
	newLines := resourceBlockLines(new)
	if len(oldLines) == 0 && len(newLines) == 0 {
		return
	}
	for _, line := range oldLines {
		fmt.Fprintf(l.out, "%s-%s%s\n", colorRed, line, colorReset) // nolint:errcheck
	}
	for _, line := range newLines {
		fmt.Fprintf(l.out, "%s+%s%s\n", colorGreen, line, colorReset) // nolint:errcheck
	}
}

func (l *Logger) log(level LogLevel, tag string, builder func() []string) {
	if level > l.minLevel || builder == nil {
		return
	}
	kv := builder()
	l.write(tag, kv)
}

func (l *Logger) write(tag string, kv []string) {
	target := l.out
	if tag == "ERROR" {
		target = l.err
	}
	if target == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s[%-8s]%s", tagColors[tag], tag, colorReset) // nolint:errcheck
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			fmt.Fprintf(&b, " %s=%s", kv[i], kv[i+1]) // nolint:errcheck
			continue
		}
		fmt.Fprintf(&b, " %s", kv[i]) // nolint:errcheck
	}
	fmt.Fprintln(target, b.String()) // nolint:errcheck
}

func resourceBlockLines(entries []string) []string {
	if len(entries) == 0 {
		return []string{"resources: []"}
	}
	lines := []string{"resources:"}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("  - %s", entry))
	}
	return lines
}
