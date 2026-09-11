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

	t.Run("skip drops matching directory", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"debug"}, nil, nil)
		ok, mode, pattern := matchSkip("apps/debug", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeDrop, mode)
		assert.Equal(t, "debug", pattern)
	})

	t.Run("opaque matches directory", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules(nil, []string{"flux/config"}, nil)
		ok, mode, _ := matchSkip("flux/config", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeOpaque, mode)
	})

	t.Run("preserve matches directory", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules(nil, nil, []string{"apps/*"})
		ok, mode, _ := matchSkip("apps/debug", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModePreserve, mode)
	})

	t.Run("glob without slash matches basename", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"patch-*.yaml"}, nil, nil)
		ok, mode, _ := matchSkip("flux/patch-test.yaml", false, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeDrop, mode)
	})

	t.Run("opaque does not hide files", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules(nil, []string{"*.yaml"}, nil)
		ok, _, _ := matchSkip("flux/app.yaml", false, rules)
		assert.False(t, ok)
	})

	t.Run("skip has precedence", func(t *testing.T) {
		t.Parallel()
		rules := parseSkipRules([]string{"config"}, []string{"config"}, []string{"config"})
		ok, mode, _ := matchSkip("config", true, rules)
		require.True(t, ok)
		assert.Equal(t, skipModeDrop, mode)
	})
}

func TestHandleSkipDir(t *testing.T) {
	t.Parallel()

	entry := dirEntryStub{name: "config"}

	t.Run("opaque keeps dir and skips walk", func(t *testing.T) {
		t.Parallel()
		dirEntries, childDirs := handleSkipDir(entry, skipModeOpaque, nil, nil)
		assert.Equal(t, []string{"config"}, dirEntries)
		require.Len(t, childDirs, 1)
		assert.True(t, childDirs[0].skipWalk)
	})

	t.Run("preserve keeps dir and skips its update", func(t *testing.T) {
		t.Parallel()
		dirEntries, childDirs := handleSkipDir(entry, skipModePreserve, nil, nil)
		assert.Equal(t, []string{"config"}, dirEntries)
		require.Len(t, childDirs, 1)
		assert.True(t, childDirs[0].skipUpdate)
	})

	t.Run("skip drops directory", func(t *testing.T) {
		t.Parallel()
		dirEntries, childDirs := handleSkipDir(entry, skipModeDrop, []string{"foo"}, nil)
		assert.Equal(t, []string{"foo"}, dirEntries)
		assert.Empty(t, childDirs)
	})
}

func TestMatchesSkipPattern(t *testing.T) {
	t.Parallel()

	assert.True(t, matchesSkipPattern("apps/debug", "debug"))
	assert.True(t, matchesSkipPattern("apps/debug", "apps/*"))
	assert.False(t, matchesSkipPattern("apps/debug/nested", "apps/*"))
	assert.True(t, matchesSkipPattern("apps/patch-test.yaml", "patch-*.yaml"))
}
