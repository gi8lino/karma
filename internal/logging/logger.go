package logging

import (
	"fmt"
	"io"
	"os"
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
	LevelVerbose
	LevelDebug
	LevelTrace
)

// LevelFromVerbosity maps a CLI verbosity counter to an internal log level.
func LevelFromVerbosity(v int) LogLevel {
	if v < 0 {
		return LevelOff
	}
	switch {
	case v >= 3:
		return LevelTrace
	case v == 2:
		return LevelDebug
	case v == 1:
		return LevelVerbose
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
	return &Logger{
		out:      out,
		err:      err,
		minLevel: level,
	}
}

// Flush exists for symmetry with buffered I/O.
func (l *Logger) Flush() {}

// Processing logs the current directory when the console level allows it.
func (l *Logger) Processing(kind string, kv ...string) {
	l.log(l.out, LevelInfo, "PROCESS", func() []string {
		return append([]string{"kind", kind}, kv...)
	})
}

// Skipped logs resources that were skipped because of a rule.
func (l *Logger) Skipped(kv ...string) {
	l.log(l.out, LevelDebug, "SKIPPING", func() []string {
		return kv
	})
}

// Updated logs that a kustomization was rewritten.
func (l *Logger) Updated(path string, kv ...string) {
	l.log(l.out, LevelInfo, "UPDATED", func() []string {
		return append([]string{"kustomization", path}, kv...)
	})
}

// NoOp logs that a kustomization was already in sync.
func (l *Logger) NoOp(path string, kv ...string) {
	l.log(l.out, LevelDebug, "NO-OP", func() []string {
		return append([]string{"kustomization", path}, kv...)
	})
}

// Debug logs debug-level output when the log level permits.
func (l *Logger) Debug(msg string, kv ...string) {
	l.log(l.out, LevelDebug, "DEBUG", func() []string {
		return append([]string{"message", msg}, kv...)
	})
}

// DebugKV logs debug-level key/value pairs without a message prefix.
func (l *Logger) DebugKV(kv ...string) {
	l.log(l.out, LevelDebug, "DEBUG", func() []string {
		return kv
	})
}

// Trace logs trace-level output when the log level permits.
func (l *Logger) Trace(msg string, kv ...string) {
	l.log(l.out, LevelTrace, "TRACE", func() []string {
		return append([]string{"message", msg}, kv...)
	})
}

// Summary prints the overall update statistics.
func (l *Logger) Summary(updated, noOp, reordered, added, removed int) {
	l.log(l.out, LevelInfo, "SUMMARY", func() []string {
		kv := []string{
			"updated", fmt.Sprintf("%d", updated),
			"no-op", fmt.Sprintf("%d", noOp),
			"order", fmt.Sprintf("%d", reordered),
			"added", fmt.Sprintf("%d", added),
			"removed", fmt.Sprintf("%d", removed),
		}
		return kv
	})
}

// Error logs an error to stderr regardless of verbosity.
func (l *Logger) Error(msg string, kv ...string) {
	l.log(l.err, LevelError, "ERROR", func() []string {
		return append([]string{"message", msg}, kv...)
	})
}

// ResourceDiff prints an old/new snapshot of the resources block.
func (l *Logger) ResourceDiff(old, new []string) {
	if l.minLevel < LevelVerbose {
		return
	}
	const diffIndent = "           "
	removed, added := diffStrings(old, new)
	if len(removed) == 0 && len(added) == 0 {
		return
	}
	useColor := shouldColor(l.out)
	for _, line := range removed {
		if useColor {
			fmt.Fprintf(l.out, "%s%s-  - %q%s\n", colorRed, diffIndent, line, colorReset) // nolint:errcheck
			continue
		}
		fmt.Fprintf(l.out, "%s-  - %q\n", diffIndent, line) // nolint:errcheck
	}
	for _, line := range added {
		if useColor {
			fmt.Fprintf(l.out, "%s%s+  - %q%s\n", colorGreen, diffIndent, line, colorReset) // nolint:errcheck
			continue
		}
		fmt.Fprintf(l.out, "%s+  - %q\n", diffIndent, line) // nolint:errcheck
	}
}

// DiffStrings returns removed and added entries between two slices of resources.
func diffStrings(old, new []string) (removed, added []string) {
	counts := make(map[string]int, len(old))

	// Count the number of times each resource is in the
	for _, entry := range old {
		counts[entry]++
	}

	// Count the number of times each resource is in the new list.
	for _, entry := range new {
		if counts[entry] > 0 {
			counts[entry]--
			if counts[entry] == 0 {
				delete(counts, entry)
			}
			continue
		}
		added = append(added, entry)
	}

	// Collect the removed resources.
	for _, entry := range old {
		if c, ok := counts[entry]; ok && c > 0 {
			removed = append(removed, entry)
			counts[entry]--
			if counts[entry] == 0 {
				delete(counts, entry)
			}
		}
	}

	return removed, added
}

// Log executes the provided builder when the configured level allows it.
func (l *Logger) log(w io.Writer, level LogLevel, tag string, builder func() []string) {
	if level > l.minLevel || builder == nil {
		return
	}
	kv := builder()
	l.write(w, tag, kv)
}

// Write renders a formatted log line to the configured output stream.
func (l *Logger) write(w io.Writer, tag string, kv []string) {
	var b strings.Builder
	if shouldColor(w) {
		fmt.Fprintf(&b, "%s[%-8s]%s", tagColors[tag], tag, colorReset) // nolint:errcheck
	} else {
		fmt.Fprintf(&b, "[%-8s]", tag) // nolint:errcheck
	}
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			fmt.Fprintf(&b, " %s=%s", kv[i], kv[i+1]) // nolint:errcheck
			continue
		}
		fmt.Fprintf(&b, " %s", kv[i]) // nolint:errcheck
	}
	fmt.Fprintln(w, b.String()) // nolint:errcheck
}

// shouldColor reports whether ANSI colors are appropriate for w. Colors are
// disabled for redirected output, non-files, dumb terminals, and NO_COLOR.
func shouldColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
