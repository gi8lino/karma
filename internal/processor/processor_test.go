package processor

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gi8lino/kustomizer/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProcessorProcess(t *testing.T) {
	t.Parallel()

	t.Run("createsMissingKustomization", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		updated, noOp, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, 1, updated)
		assert.Equal(t, 0, noOp)

		data, err := os.ReadFile(filepath.Join(temp, "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "resources:")
		assert.Contains(t, string(data), "app.yaml")
	})

	t.Run("reusesUpToDateKustomization", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, "app.yaml"), []byte("kind: ConfigMap\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, _, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)

		updated, noOp, err := proc.Process(context.Background(), temp)
		require.NoError(t, err)
		assert.Equal(t, 0, updated)
		assert.Equal(t, 1, noOp)
	})
}

func TestScanEntriesHonorsSkips(t *testing.T) {
	t.Parallel()

	t.Run("skipsAndReports", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(temp, "normal"), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(temp, "skipdir"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(temp, "normal.yaml"), []byte("x: 1\n"), 0o644))

		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{
			Skip:       []string{"skipdir/*"},
			IncludeDot: false,
		}, logger)

		dirEntries, fileEntries, childDirs, err := proc.scanEntries(temp, temp, nil)
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

func TestProcessorLoadMatcher(t *testing.T) {
	t.Parallel()

	t.Run("returnsNilWhenDisabled", func(t *testing.T) {
		t.Parallel()
		proc := New(Options{UseGitIgnore: false}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		matcher, err := proc.loadMatcher(t.TempDir(), nil)
		require.NoError(t, err)
		assert.Nil(t, matcher)
	})

	t.Run("loadsAndRespectsGitignore", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(temp, ".gitignore"), []byte("secret.txt\n"), 0o644))
		proc := New(Options{UseGitIgnore: true}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		matcher, err := proc.loadMatcher(temp, nil)
		require.NoError(t, err)
		require.NotNil(t, matcher)
		assert.True(t, matcher.Ignored(filepath.Join(temp, "secret.txt"), false))
	})
}

func TestProcessorRelPath(t *testing.T) {
	t.Parallel()

	t.Run("returnsBaseNameForRoot", func(t *testing.T) {
		t.Parallel()
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		temp := t.TempDir()
		path := filepath.Join(temp, "foo")
		rel := proc.relPath(temp, path)
		assert.Equal(t, "foo", rel)
	})

	t.Run("convertsToSlash", func(t *testing.T) {
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

	t.Run("selectsYamlWhenPresent", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("kind: test\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		got, exists, err := proc.pickKustomizationPath(temp)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, path, got)
	})

	t.Run("selectsYmlWhenOnlyYmlExists", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yml")
		require.NoError(t, os.WriteFile(path, []byte("kind: test\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		got, exists, err := proc.pickKustomizationPath(temp)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, path, got)
	})

	t.Run("defaultsWhenMissing", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		got, exists, err := proc.pickKustomizationPath(temp)
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Equal(t, filepath.Join(temp, "kustomization.yaml"), got)
	})
}

func TestProcessorUpdateKustomization(t *testing.T) {
	t.Parallel()

	t.Run("rewritesResources", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - existing\n"), 0o644))
		proc := New(Options{DirSlash: true, DirFirst: true}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		updated, err := proc.updateKustomization(path, true, []string{"added"}, []string{"alpha.yaml"})
		require.NoError(t, err)
		assert.True(t, updated)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "added/")
		assert.Contains(t, string(data), "alpha.yaml")
	})

	t.Run("returnsFalseWhenUnchanged", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - exist\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		_, err := proc.updateKustomization(path, true, []string{"exist"}, nil)
		require.NoError(t, err)

		updated, err := proc.updateKustomization(path, true, []string{"exist"}, nil)
		require.NoError(t, err)
		assert.False(t, updated)
	})
}

func TestProcessorApplyKustomization(t *testing.T) {
	t.Parallel()

	t.Run("respectsSkipUpdate", func(t *testing.T) {
		t.Parallel()
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))
		updated, noOp, err := proc.applyKustomization("", "", true, nil, nil, true)
		require.NoError(t, err)
		assert.Equal(t, 0, updated)
		assert.Equal(t, 0, noOp)
	})

	t.Run("reportsUpdatedWhenChanged", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		proc := New(Options{DirSlash: true}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		updated, noOp, err := proc.applyKustomization(temp, path, false, []string{"dir"}, []string{"file.yaml"}, false)
		require.NoError(t, err)
		assert.Equal(t, 1, updated)
		assert.Equal(t, 0, noOp)
	})

	t.Run("silentSuppressesNoOpLog", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - exist\n"), 0o644))

		var out bytes.Buffer
		logger := logging.New(&out, io.Discard, logging.LevelInfo)
		proc := New(Options{Silent: true}, logger)

		updated, noOp, err := proc.applyKustomization(temp, path, true, []string{"exist"}, nil, false)
		require.NoError(t, err)
		assert.Equal(t, 0, updated)
		assert.Equal(t, 1, noOp)
		assert.Empty(t, out.String())
	})
}

func TestProcessorLoadKustomization(t *testing.T) {
	t.Parallel()

	t.Run("loadsExistingFile", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		require.NoError(t, os.WriteFile(path, []byte("---\nresources:\n  - kept\n"), 0o644))
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		root, seq, order, nodes, err := proc.loadKustomization(path, true)
		require.NoError(t, err)
		require.NotNil(t, root)
		require.NotNil(t, seq)
		require.NotNil(t, nodes)
		assert.Contains(t, order, "kept")
	})

	t.Run("initializesMissingDocument", func(t *testing.T) {
		t.Parallel()
		temp := t.TempDir()
		path := filepath.Join(temp, "kustomization.yaml")
		proc := New(Options{}, logging.New(io.Discard, io.Discard, logging.LevelInfo))

		root, seq, order, nodes, err := proc.loadKustomization(path, false)
		require.NoError(t, err)
		require.NotNil(t, root)
		require.NotNil(t, seq)
		assert.Empty(t, order)
		assert.Empty(t, nodes)
	})
}

func TestEnsureResourcesSeq(t *testing.T) {
	t.Parallel()

	t.Run("createsSequenceWhenMissing", func(t *testing.T) {
		t.Parallel()
		root := &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode},
			},
		}
		seq, order, _, err := ensureResourcesSeq(root)
		require.NoError(t, err)
		require.NotNil(t, seq)
		assert.Empty(t, order)
	})

	t.Run("reusesExistingSequence", func(t *testing.T) {
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
		seq, order, _, err := ensureResourcesSeq(root)
		require.NoError(t, err)
		assert.Equal(t, seqNode, seq)
		assert.Empty(t, order)
	})
}

func TestCollectExistingResources(t *testing.T) {
	t.Parallel()

	t.Run("indexesOnlyScalarNodes", func(t *testing.T) {
		t.Parallel()
		seq := &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "one"},
				{Kind: yaml.ScalarNode, Value: "two"},
				{Kind: yaml.MappingNode},
				{Kind: yaml.ScalarNode, Value: "one"},
			},
		}
		nodes, order := collectExistingResources(seq)
		require.Len(t, order, 2)
		assert.Equal(t, []*yaml.Node{{Kind: yaml.ScalarNode, Value: "one"}}, []*yaml.Node{nodes["one"]})
		assert.Equal(t, "one", order[0])
		assert.Equal(t, "two", order[1])
		assert.Len(t, nodes, 2)
	})
}

func TestMergeResourcesOrders(t *testing.T) {
	t.Parallel()

	t.Run("dirFirst", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{DirSlash: true, DirFirst: true}, logger)
		final := proc.mergeResources([]string{"https://example.com"}, []string{"b", "a"}, []string{"z", "y"})
		require.Equal(t, []string{"https://example.com", "a/", "b/", "y", "z"}, final)
	})

	t.Run("alphabeticalFallback", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{DirSlash: true, DirFirst: false}, logger)
		final := proc.mergeResources([]string{"https://example.com", "https://stable.com"}, []string{"b", "a"}, []string{"x"})
		require.Equal(t, []string{"https://example.com", "https://stable.com", "a/", "b/", "x"}, final)
	})
}

func TestProcessorDecorateSubdirs(t *testing.T) {
	t.Parallel()

	t.Run("appendsSlashWhenEnabled", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{DirSlash: true}, logger)
		got := proc.decorateSubdirs([]string{"app", "config/"})
		assert.Equal(t, []string{"app/", "config/"}, got)
	})

	t.Run("leavesInputWhenDisabled", func(t *testing.T) {
		t.Parallel()
		logger := logging.New(io.Discard, io.Discard, logging.LevelInfo)
		proc := New(Options{DirSlash: false}, logger)
		got := proc.decorateSubdirs([]string{"app", "config"})
		assert.Equal(t, []string{"app", "config"}, got)
	})
}
