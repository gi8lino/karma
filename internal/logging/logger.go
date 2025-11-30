package logging

import (
	"fmt"
	"io"
)

// Logger manages structured terminal output with verbosity controls.
type Logger struct {
	out       io.Writer
	err       io.Writer
	verbosity int
}

// New creates a Logger that writes info to out and errors to err.
func New(out, err io.Writer, verbosity int) *Logger {
	return &Logger{
		out:       out,
		err:       err,
		verbosity: verbosity,
	}
}

// Flush exists for symmetry with buffered writers.
func (l *Logger) Flush() {}

// Processing logs the current directory being walked.
func (l *Logger) Processing(kind string, kv ...string) {
	l.log(l.out, "PROCESS", append([]string{"kind", kind}, kv...)...)
}

// Skipped logs resources that were skipped due to configured rules.
func (l *Logger) Skipped(kv ...string) {
	l.log(l.out, "SKIPPING", kv...)
}

// Updated logs that a kustomization was updated.
func (l *Logger) Updated(path string, kv ...string) {
	l.log(l.out, "UPDATED", append([]string{"kustomization", path}, kv...)...)
}

// NoOp logs that a kustomization was already in sync.
func (l *Logger) NoOp(path string, kv ...string) {
	l.log(l.out, "NO-OP", append([]string{"kustomization", path}, kv...)...)
}

// Trace logs detail-level traces when verbosity is high.
func (l *Logger) Trace(msg string, kv ...string) {
	l.log(l.out, "TRACE", append([]string{"message", msg}, kv...)...)
}

// Summary prints the final result counts.
func (l *Logger) Summary(updated, noOp int) {
	l.log(l.out, "SUMMARY", fmt.Sprintf("updated=%d", updated), fmt.Sprintf("no-op=%d", noOp))
}

// Error logs an error to stderr.
func (l *Logger) Error(msg string, kv ...string) {
	l.log(l.err, "ERROR", append([]string{"message", msg}, kv...)...)
}

func (l *Logger) log(w io.Writer, tag string, kv ...string) {
	if l.verbosity < 1 {
		return
	}

	fmt.Fprintf(w, "[%-8s]", tag) // nolint:errcheck
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			fmt.Fprintf(w, " %s=%s", kv[i], kv[i+1]) // nolint:errcheck
		} else {
			fmt.Fprintf(w, " %s", kv[i]) //nolint:errcheck
		}
	}
	fmt.Fprintln(w) // nolint:errcheck
}
