package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferencedLocalEntries(t *testing.T) {
	t.Parallel()

	t.Run("collects known path fields", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
components:
  - component
configurations:
  - config.yaml
crds:
  - crd.yaml
patches:
  - path: patch.yaml
patchesJson6902:
  - path: json-patch.yaml
patchesStrategicMerge:
  - strategic.yaml
replacements:
  - path: replacements.yaml
openapi:
  path: schema.yaml
helmCharts:
  - name: demo
    valuesFile: values.yaml
    additionalValuesFiles:
      - extra-values.yaml
helmChartInflationGenerator:
  - name: old
    values: old-values.yaml
configMapGenerator:
  - name: config
    env: config.env
    envs:
      - extra.env
    files:
      - config=config-file.yaml
secretGenerator:
  - name: secret
    files:
      - secret.yaml
`), 0o644))

		got, err := referencedLocalEntries(path)
		require.NoError(t, err)
		for _, name := range []string{
			"component", "config.yaml", "crd.yaml", "patch.yaml", "json-patch.yaml",
			"strategic.yaml", "replacements.yaml", "schema.yaml", "values.yaml",
			"extra-values.yaml", "old-values.yaml", "config.env", "extra.env",
			"config-file.yaml", "secret.yaml",
		} {
			assert.Contains(t, got, name)
		}
	})

	t.Run("ignores arbitrary yaml-looking strings", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  annotations:
    example.com/config: app.yaml
commonAnnotations:
  file: another.yaml
`), 0o644))

		got, err := referencedLocalEntries(path)
		require.NoError(t, err)
		assert.NotContains(t, got, "app.yaml")
		assert.NotContains(t, got, "another.yaml")
	})

	t.Run("keeps only direct local paths", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
components:
  - ./local
  - nested/component
  - ../shared
  - https://example.com/component
`), 0o644))

		got, err := referencedLocalEntries(path)
		require.NoError(t, err)
		assert.Equal(t, map[string]struct{}{"local": {}}, got)
	})
}
