package processor

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gi8lino/karma/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProcessorProcess(t *testing.T) {
	t.Parallel()

	t.Run("honors canceled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, err := proc.Process(ctx, t.TempDir())
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("creates missing kustomization", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		stats, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, 1, stats.Updated)
		assert.Equal(t, 0, stats.NoOp)

		data, err := os.ReadFile(filepath.Join(temp, "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "resources:")
		assert.Contains(t, string(data), "app.yaml")
	})

	t.Run("reuses up-to-date kustomization", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)

		stats, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, 0, stats.Updated)
		assert.Equal(t, 1, stats.NoOp)
	})

	t.Run("skips named component", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		kustom := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(kustom, []byte("kind: Component\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		stats, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, ResourceStats{}, stats)

		data, err := os.ReadFile(kustom)
		require.NoError(t, err)
		assert.Contains(t, string(data), "Component")
		assert.NotContains(t, string(data), "resources:")
	})

	t.Run("does not add yaml referenced by other kustomization fields", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		kustom := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(kustom, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - app.yaml
  - patch.yaml
patches:
  - path: patch.yaml
configurations:
  - kustomizeconfig.yaml
helmCharts:
  - name: demo
    valuesFile: values.yaml
`), 0o644))
		for _, name := range []string{"app.yaml", "patch.yaml", "kustomizeconfig.yaml", "values.yaml"} {
			require.NoError(t, os.WriteFile(filepath.Join(temp, name), []byte("kind: ConfigMap\n"), 0o644))
		}

		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		_, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)

		data, err := os.ReadFile(kustom)
		require.NoError(t, err)
		var doc struct {
			Resources []string `yaml:"resources"`
		}
		require.NoError(t, yaml.Unmarshal(data, &doc))
		assert.Equal(t, []string{"app.yaml"}, doc.Resources)
	})

	t.Run("removes duplicate resource entries", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - app.yaml
  - app.yaml
`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		stats, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, 1, stats.Updated)
		assert.Equal(t, 1, stats.Removed)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var doc struct {
			Resources []string `yaml:"resources"`
		}
		require.NoError(t, yaml.Unmarshal(data, &doc))
		assert.Equal(t, []string{"app.yaml"}, doc.Resources)
	})

	t.Run("does not add component directory to resources", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		component := filepath.Join(temp, "component")
		require.NoError(t, os.Mkdir(component, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(component, "kustomization.yaml"), []byte(`apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		kustom := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(kustom, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
components:
  - component
resources:
  - app.yaml
`), 0o644))

		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		_, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)

		data, err := os.ReadFile(kustom)
		require.NoError(t, err)
		var doc struct {
			Resources []string `yaml:"resources"`
		}
		require.NoError(t, yaml.Unmarshal(data, &doc))
		assert.Equal(t, []string{"app.yaml"}, doc.Resources)
	})

	t.Run("does not treat arbitrary yaml-looking values as references", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		kustom := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(kustom, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
commonAnnotations:
  config: app.yaml
resources: []
`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))

		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		_, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)

		data, err := os.ReadFile(kustom)
		require.NoError(t, err)
		var doc struct {
			Resources []string `yaml:"resources"`
		}
		require.NoError(t, yaml.Unmarshal(data, &doc))
		assert.Equal(t, []string{"app.yaml"}, doc.Resources)
	})

	t.Run("does not use arbitrary yaml by kind", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		custom := filepath.Join(temp, "custom.yaml")
		require.NoError(t, os.WriteFile(custom, []byte("kind: Kustomization\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		stats, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, 1, stats.Updated)

		data, err := os.ReadFile(filepath.Join(temp, "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "app.yaml")
		assert.Contains(t, string(data), "custom.yaml")

		customData, err := os.ReadFile(custom)
		require.NoError(t, err)
		assert.Equal(t, "kind: Kustomization\n", string(customData))
	})
}

func TestResourceStatsAdd(t *testing.T) {
	t.Parallel()

	t.Run("sums_all_fields", func(t *testing.T) {
		t.Parallel()
		base := ResourceStats{Reordered: 1, Added: 2, Removed: 3, Updated: 4, NoOp: 5}
		add := ResourceStats{Reordered: 10, Added: 20, Removed: 30, Updated: 40, NoOp: 50}
		base.Add(add)
		assert.Equal(t, ResourceStats{Reordered: 11, Added: 22, Removed: 33, Updated: 44, NoOp: 55}, base)
	})

	t.Run("zero_other_leaves_original", func(t *testing.T) {
		t.Parallel()
		base := ResourceStats{Reordered: 1, Added: 1, Removed: 1, Updated: 1, NoOp: 1}
		base.Add(ResourceStats{})
		assert.Equal(t, ResourceStats{Reordered: 1, Added: 1, Removed: 1, Updated: 1, NoOp: 1}, base)
	})
}

func TestScanEntries(t *testing.T) {
	t.Parallel()

	t.Run("skips and reports", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(temp, "normal"), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(temp, "skipdir"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "normal.yaml"), []byte("x: 1\n"), 0o644))

		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{
			Opaque:     []string{"skipdir"},
			IncludeDot: false,
		}, logger)

		dirEntries, fileEntries, childDirs, err := proc.scanEntries(context.Background(), temp, temp, nil, "")
		require.NoError(t, err)
		assert.Contains(t, dirEntries, "normal")
		assert.Contains(t, dirEntries, "skipdir")
		assert.Contains(t, fileEntries, "normal.yaml")
		require.Len(t, childDirs, 2)
		for _, child := range childDirs {
			if child.name == "skipdir" {
				assert.True(t, child.skipWalk)
			}
		}
	})
}

func TestProcessorRelPath(t *testing.T) {
	t.Parallel()

	t.Run("returns basename for root", func(t *testing.T) {
		t.Parallel()
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		temp := t.TempDir()
		path := filepath.Join(temp, "foo")
		rel := proc.relPath(temp, path)
		assert.Equal(t, "foo", rel)
	})

	t.Run("converts to slash", func(t *testing.T) {
		t.Parallel()
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		base := filepath.Join(t.TempDir(), "base")
		full := filepath.Join(base, "nested", "file")
		rel := proc.relPath(base, full)
		assert.Equal(t, "nested/file", rel)
	})
}

func TestProcessorPickKustomizationPath(t *testing.T) {
	t.Parallel()

	for _, name := range kustomizationNames {
		t.Run("selects "+name, func(t *testing.T) {
			t.Parallel()
			temp := t.TempDir()
			path := filepath.Join(temp, name)
			require.NoError(t, os.WriteFile(path, []byte("kind: Kustomization\n"), 0o644))
			proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

			got, exists, err := proc.pickKustomizationPath(temp)
			require.NoError(t, err)
			assert.True(t, exists)
			assert.Equal(t, path, got)
		})
	}

	t.Run("rejects multiple canonical files", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "kustomization.yaml"), []byte("kind: Kustomization\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "Kustomization"), []byte("kind: Kustomization\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, _, err := proc.pickKustomizationPath(temp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple kustomization files")
	})

	t.Run("ignores yaml kind when filename is not canonical", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "custom.yaml"), []byte("kind: Kustomization\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		got, exists, err := proc.pickKustomizationPath(temp)
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Equal(t, filepath.Join(temp, "kustomization.yaml"), got)
	})
}

func TestProcessorUpdateKustomization(t *testing.T) {
	t.Parallel()

	t.Run("rewrites resources", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - existing\n"), 0o644))
		opts := Options{
			AddDirPrefix:  true,
			AddDirSuffix:  true,
			ResourceOrder: []string{"external", "dirs"},
		}
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(opts, logger)
		updated, order, final, stats, err := proc.updateKustomization(path, true, []string{"added"}, []string{"alpha.yaml"})
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 0, stats.Reordered)
		assert.Equal(t, []string{"existing"}, order)
		t.Logf("final resources: %v", final)
		assert.Contains(t, final, "./added/")
		assert.Contains(t, final, "alpha.yaml")
		assert.Greater(t, stats.Added, 0)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "./added/")
		assert.Contains(t, string(data), "apiVersion")
		assert.Contains(t, string(data), "kind")
	})

	t.Run("writes missing header even when resources are unchanged", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("resources:\n  - app.yaml\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		updated, _, _, _, err := proc.updateKustomization(path, true, nil, []string{"app.yaml"})
		require.NoError(t, err)
		assert.True(t, updated)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "apiVersion: kustomize.config.k8s.io/v1beta1")
		assert.Contains(t, string(data), "kind: Kustomization")
	})

	t.Run("creates an empty missing kustomization", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		updated, _, _, _, err := proc.updateKustomization(path, false, nil, nil)
		require.NoError(t, err)
		assert.True(t, updated)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "apiVersion:")
		assert.Contains(t, string(data), "kind: Kustomization")
		assert.Contains(t, string(data), "resources:")
	})

	t.Run("dry run reports update without writing", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		proc := New(Options{DryRun: true}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		updated, _, final, _, err := proc.updateKustomization(path, false, nil, []string{"app.yaml"})
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, []string{"app.yaml"}, final)
		_, err = os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("returns false when unchanged", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - exist\n"), 0o644))
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{}, logger)

		_, _, _, _, err := proc.updateKustomization(path, true, []string{"exist"}, nil)
		require.NoError(t, err)

		updated, order, final, stats, err := proc.updateKustomization(path, true, []string{"exist"}, nil)
		require.NoError(t, err)
		assert.False(t, updated)
		assert.Equal(t, 0, stats.Reordered)
		assert.Equal(t, []string{"exist"}, order)
		assert.Equal(t, []string{"exist"}, final)
		assert.Equal(t, 0, stats.Added)
		assert.Equal(t, 0, stats.Removed)
	})
}

func TestProcessorApplyKustomization(t *testing.T) {
	t.Parallel()

	t.Run("respects skip update", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{}, logger)
		stats, err := proc.applyKustomization("", "", true, nil, nil, true)
		require.NoError(t, err)
		assert.Equal(t, 0, stats.Updated)
		assert.Equal(t, 0, stats.NoOp)
	})

	t.Run("reports updated when changed", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		opts := Options{AddDirPrefix: true, ResourceOrder: []string{"external", "dirs"}}
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(opts, logger)

		stats, err := proc.applyKustomization(temp, path, false, []string{"dir"}, []string{"file.yaml"}, false)
		require.NoError(t, err)
		assert.Equal(t, 1, stats.Updated)
		assert.Equal(t, 0, stats.NoOp)
	})
}

func TestProcessorLoadKustomization(t *testing.T) {
	t.Parallel()

	t.Run("loads existing file", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - kept\n"), 0o644))
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{}, logger)

		root, seq, order, nodes, _, err := proc.loadKustomization(path, true)
		require.NoError(t, err)
		require.NotNil(t, root)
		require.NotNil(t, seq)
		require.NotNil(t, nodes)
		assert.Contains(t, order, "kept")
	})

	t.Run("rejects non-mapping root", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("- one\n- two\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, _, _, _, _, err := proc.loadKustomization(path, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "root must be a YAML mapping")
	})

	t.Run("rejects non-sequence resources", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("resources: app.yaml\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, _, _, _, _, err := proc.loadKustomization(path, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resources must be a YAML sequence")
	})

	t.Run("initializes missing document", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{}, logger)

		root, seq, order, nodes, _, err := proc.loadKustomization(path, false)
		require.NoError(t, err)
		require.NotNil(t, root)
		require.NotNil(t, seq)
		assert.Empty(t, order)
		assert.Empty(t, nodes)
	})
}

func TestEnsureResourcesSeq(t *testing.T) {
	t.Parallel()

	t.Run("creates sequence when missing", func(t *testing.T) {
		t.Parallel()
		root := &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode},
			},
		}
		seq, order, _, _, err := ensureResourcesSeq(root)
		require.NoError(t, err)
		require.NotNil(t, seq)
		assert.Empty(t, order)
	})

	t.Run("reuses existing sequence", func(t *testing.T) {
		t.Parallel()
		seqNode := &yaml.Node{Kind: yaml.SequenceNode}
		key := &yaml.Node{Kind: yaml.ScalarNode, Value: "resources"}
		root := &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{
					Kind:    yaml.MappingNode,
					Content: []*yaml.Node{key, seqNode},
				},
			},
		}
		seq, order, _, _, err := ensureResourcesSeq(root)
		require.NoError(t, err)
		assert.Equal(t, seqNode, seq)
		assert.Empty(t, order)
	})
}

func TestCollectExistingResources(t *testing.T) {
	t.Parallel()

	t.Run("indexes scalar nodes and preserves duplicates", func(t *testing.T) {
		t.Parallel()
		seq := &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "one"},
				{Kind: yaml.ScalarNode, Value: "two"},
				{Kind: yaml.ScalarNode, Value: "one"},
			},
		}
		nodes, order, err := collectExistingResources(seq)
		require.NoError(t, err)
		assert.Equal(t, []string{"one", "two", "one"}, order)
		assert.Len(t, nodes, 2)
	})

	t.Run("rejects structured resource entries", func(t *testing.T) {
		t.Parallel()
		seq := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
		_, _, err := collectExistingResources(seq)
		require.Error(t, err)
	})
}

func TestMergeResourcesPreservesUnmanagedReferences(t *testing.T) {
	t.Parallel()

	proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
	final := proc.mergeResources(
		[]string{"../base", "nested/shared", "https://example.com/base", "removed.yaml"},
		nil,
		[]string{"app.yaml"},
	)

	assert.Contains(t, final, "../base")
	assert.Contains(t, final, "nested/shared")
	assert.Contains(t, final, "https://example.com/base")
	assert.Contains(t, final, "app.yaml")
	assert.NotContains(t, final, "removed.yaml")
}

func TestMergeResourcesOrders(t *testing.T) {
	t.Parallel()

	t.Run("dir first ordering", func(t *testing.T) {
		t.Parallel()
		opts := Options{
			AddDirPrefix:  true,
			AddDirSuffix:  true,
			ResourceOrder: []string{"external", "dirs"},
		}
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(opts, logger)
		final := proc.mergeResources([]string{"https://example.com"}, []string{"b", "a"}, []string{"z", "y"})
		require.Equal(t, []string{"https://example.com", "./a/", "./b/", "y", "z"}, final)
	})

	t.Run("alphabetical fallback", func(t *testing.T) {
		t.Parallel()
		opts := Options{
			AddDirPrefix:  true,
			AddDirSuffix:  true,
			ResourceOrder: []string{"external", "files", "dirs"},
		}
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(opts, logger)
		final := proc.mergeResources([]string{"https://example.com", "https://stable.com"}, []string{"b", "a"}, []string{"x"})
		require.Equal(t, []string{"https://example.com", "https://stable.com", "x", "./a/", "./b/"}, final)
	})
}

func TestProcessorEnsureDirSuffix(t *testing.T) {
	t.Parallel()

	t.Run("appends slash when enabled", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		opts := Options{AddDirSuffix: true}
		proc := New(opts, logger)
		got := proc.ensureDirSuffix([]string{"app", "config"})
		assert.Equal(t, []string{"app/", "config/"}, got)
	})

	t.Run("leaves input when disabled", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		opts := Options{}
		proc := New(opts, logger)
		got := proc.ensureDirSuffix([]string{"app", "config"})
		assert.Equal(t, []string{"app", "config"}, got)
	})
}

func TestProcessorEnsureDirPrefix(t *testing.T) {
	t.Parallel()

	t.Run("adds prefix to every directory", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		opts := Options{AddDirPrefix: true}
		proc := New(opts, logger)
		got := proc.ensureDirPrefix([]string{"app", "test", "./already"})
		assert.Equal(t, []string{"./app", "./test", "./already"}, got)
	})

	t.Run("leaves input when disabled", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		opts := Options{}
		proc := New(opts, logger)
		got := proc.ensureDirPrefix([]string{"app", "test"})
		assert.Equal(t, []string{"app", "test"}, got)
	})
}

func TestOrderChangedDetection(t *testing.T) {
	t.Parallel()

	t.Run("detects reorder only", func(t *testing.T) {
		t.Parallel()
		assert.True(t, orderChanged([]string{"a", "b", "c"}, []string{"b", "a", "c"}))
	})

	t.Run("ignores identical order", func(t *testing.T) {
		t.Parallel()
		assert.False(t, orderChanged([]string{"a", "b"}, []string{"a", "b"}))
	})

	t.Run("detects reorder even with additions", func(t *testing.T) {
		t.Parallel()
		assert.True(t, orderChanged([]string{"a", "b"}, []string{"b", "a", "c"}))
	})

	t.Run("ignores added entries that don't reorder", func(t *testing.T) {
		t.Parallel()
		assert.False(t, orderChanged([]string{"a", "b"}, []string{"a", "b", "c"}))
	})

	t.Run("ignores removal without reorder", func(t *testing.T) {
		t.Parallel()
		assert.False(t, orderChanged([]string{"a", "b"}, []string{"b"}))
	})
}
