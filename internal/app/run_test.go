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

	t.Run("successfully processes directories", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		file := filepath.Join(temp, "app.yaml")
		require.NoError(t, os.WriteFile(file, []byte("kind: ConfigMap\n"), 0o644))

		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{temp}, &out)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "[SUMMARY")

		data, err := os.ReadFile(filepath.Join(temp, "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "resources:")
		assert.Contains(t, string(data), "app.yaml")
	})

	t.Run("returns parse error when missing args", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", nil, &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "positional")
	})

	t.Run("prints help when requested", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{"--help"}, &out)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "Usage:")
	})

	t.Run("prints version when requested", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		err := Run(context.Background(), "v9.9.9", []string{"--version"}, &out)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "v9.9.9")
	})

	t.Run("skips files matching patterns", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "patch-extra.yaml"), []byte("kind: ConfigMap\n"), 0o644))

		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{"-s", "patch-*", temp}, &out)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(temp, "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "app.yaml")
		assert.NotContains(t, string(data), "patch-extra.yaml")
	})
	t.Run("dry run does not write", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))

		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{"--dry-run", temp}, &out)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "updated=1")
		_, err = os.Stat(filepath.Join(temp, "kustomization.yaml"))
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("check fails when changes are required without writing", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))

		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{"--check", temp}, &out)
		require.ErrorIs(t, err, ErrCheckFailed)
		assert.Contains(t, out.String(), "updated=1")
		_, statErr := os.Stat(filepath.Join(temp, "kustomization.yaml"))
		require.ErrorIs(t, statErr, os.ErrNotExist)
	})

	t.Run("check succeeds when already synchronized", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "kustomization.yaml"), []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - app.yaml
`), 0o644))

		var out bytes.Buffer
		err := Run(context.Background(), "v1.0.0", []string{"--check", temp}, &out)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "updated=0")
	})

}
