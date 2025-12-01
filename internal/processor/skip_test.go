package processor

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dirEntryStub struct {
	name string
}

func (d dirEntryStub) Name() string               { return d.name }
func (d dirEntryStub) IsDir() bool                { return true }
func (d dirEntryStub) Type() os.FileMode          { return os.ModeDir }
func (d dirEntryStub) Info() (os.FileInfo, error) { return nil, nil }

func TestMatchSkipModes(t *testing.T) {
	t.Parallel()

	t.Run("subtree matches directory", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"flux/config/**"})
		ok, mode, _ := matchSkip("flux/config", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeSubtree, mode)
	})

	t.Run("subtree does not skip descendants", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"apps/debug/**"})
		skip, _, _ := matchSkip("apps/debug/nested", true, rules)
		assert.False(t, skip)
	})

	t.Run("children rule matches directory", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"flux/config/*"})
		ok, mode, _ := matchSkip("flux/config/child", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeChildren, mode)
	})

	t.Run("subtree with trailing slash matches", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"apps/debug/**/"})
		ok, mode, _ := matchSkip("apps/debug", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeSubtree, mode)
	})

	t.Run("glob matches pattern", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"flux/*.yaml"})
		ok, mode, _ := matchSkip("flux/sample.yaml", false, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeGlob, mode)
	})

	t.Run("exact matches", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"README"})
		ok, mode, _ := matchSkip("README", false, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeExact, mode)
	})
}

func TestHandleSkipDir(t *testing.T) {
	t.Parallel()

	entry := dirEntryStub{name: "flux/config"}

	t.Run("children mode keeps dir and skips walk", func(t *testing.T) {
		t.Parallel()

		dirEntries, childDirs := handleSkipDir(entry, skipModeChildren, nil, nil)
		assert.Equal(t, []string{"flux/config"}, dirEntries)
		require.Len(t, childDirs, 1)
		assert.True(t, childDirs[0].skipWalk)
	})

	t.Run("subtree mode keeps dir and skips update", func(t *testing.T) {
		t.Parallel()

		dirEntries, childDirs := handleSkipDir(entry, skipModeSubtree, nil, nil)
		assert.Equal(t, []string{"flux/config"}, dirEntries)
		require.Len(t, childDirs, 1)
		assert.True(t, childDirs[0].skipUpdate)
	})

	t.Run("exact mode drops directory", func(t *testing.T) {
		t.Parallel()

		dirEntries, childDirs := handleSkipDir(entry, skipModeExact, []string{"foo"}, nil)
		assert.Equal(t, []string{"foo"}, dirEntries)
		assert.Len(t, childDirs, 0)
	})
}

func TestMatchesPrefixAndChild(t *testing.T) {
	t.Parallel()

	t.Run("child matches direct descendant", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesChild("flux/config/app", "flux/config"))
	})

	t.Run("child does not match nested more than one level", func(t *testing.T) {
		t.Parallel()
		assert.False(t, matchesChild("flux/config/app/sub", "flux/config"))
	})

	t.Run("child matches root with simple name", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesChild("app", ""))
	})
}
