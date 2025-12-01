package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("successfullyProcessesDirectories", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		file := filepath.Join(temp, "app.yaml")
		require.NoError(t, os.WriteFile(file, []byte("kind: ConfigMap\n"), 0o644))

		var out, errOut bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{temp}, &out, &errOut)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "[SUMMARY")
		assert.Empty(t, errOut.String())

		data, err := os.ReadFile(filepath.Join(temp, "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "resources:")
		assert.Contains(t, string(data), "app.yaml")
	})

	t.Run("returnsParseErrorWhenMissingArgs", func(t *testing.T) {
		t.Parallel()
		var out, errOut bytes.Buffer
		err := Run(context.Background(), "v1.0.0", nil, &out, &errOut)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "positional")
	})

	t.Run("printsHelpWhenRequested", func(t *testing.T) {
		t.Parallel()
		var out, errOut bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{"--help"}, &out, &errOut)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "Usage:")
		assert.Empty(t, errOut.String())
	})

	t.Run("printsVersionWhenRequested", func(t *testing.T) {
		t.Parallel()
		var out, errOut bytes.Buffer
		err := Run(context.Background(), "v9.9.9", []string{"--version"}, &out, &errOut)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "v9.9.9")
		assert.Empty(t, errOut.String())
	})
}
