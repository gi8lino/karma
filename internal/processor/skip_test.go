package processor

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchSkipModes(t *testing.T) {
	t.Parallel()

	t.Run("subtree", func(t *testing.T) {
		rules := parseSkipRules([]string{"flux/config/**"})
		ok, mode, _ := matchSkip("flux/config/nested", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeSubtree, mode)
	})

	t.Run("children", func(t *testing.T) {
		rules := parseSkipRules([]string{"flux/config/*"})
		ok, mode, _ := matchSkip("flux/config/child", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeChildren, mode)
	})

	t.Run("glob", func(t *testing.T) {
		rules := parseSkipRules([]string{"flux/*.yaml"})
		ok, mode, _ := matchSkip("flux/sample.yaml", false, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeGlob, mode)
	})

	t.Run("exact", func(t *testing.T) {
		rules := parseSkipRules([]string{"README"})
		ok, mode, _ := matchSkip("README", false, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeExact, mode)
	})
}

func TestHandleSkipDir(t *testing.T) {
	t.Parallel()

	entry := dirEntryStub{name: "flux/config"}
	dirEntries, childDirs := handleSkipDir(entry, skipModeChildren, nil, nil)
	assert.Equal(t, []string{"flux/config"}, dirEntries)
	require.Len(t, childDirs, 1)
	assert.True(t, childDirs[0].skipWalk)

	dirEntries, childDirs = handleSkipDir(entry, skipModeSubtree, nil, nil)
	assert.Equal(t, []string{"flux/config"}, dirEntries)
	require.Len(t, childDirs, 1)
	assert.True(t, childDirs[0].skipUpdate)

	dirEntries, childDirs = handleSkipDir(entry, skipModeExact, []string{"foo"}, nil)
	assert.Equal(t, []string{"foo"}, dirEntries)
	assert.Len(t, childDirs, 0)
}

func TestMatchesPrefixAndChild(t *testing.T) {
	t.Parallel()

	assert.True(t, matchesPrefix("flux/config/app", "flux/config"))
	assert.False(t, matchesPrefix("flux/config", "flux/config/app"))
	assert.True(t, matchesChild("flux/config/app", "flux/config"))
	assert.False(t, matchesChild("flux/config/app/sub", "flux/config"))
	assert.True(t, matchesChild("app", ""))
}

type dirEntryStub struct {
	name string
}

func (d dirEntryStub) Name() string               { return d.name }
func (d dirEntryStub) IsDir() bool                { return true }
func (d dirEntryStub) Type() os.FileMode          { return os.ModeDir }
func (d dirEntryStub) Info() (os.FileInfo, error) { return nil, nil }
