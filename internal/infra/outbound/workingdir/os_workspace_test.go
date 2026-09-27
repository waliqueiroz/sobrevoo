package workingdir_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/workingdir"
)

func Test_OS_NewTemporary(t *testing.T) {
	t.Run("should create a directory that exists and can be written to", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()

		// when
		path, remove, err := workspace.NewTemporary()
		defer remove()

		// then
		require.NoError(t, err)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
		assert.NoError(t, os.WriteFile(filepath.Join(path, "frame_000000.png"), []byte("x"), 0o644))
	})

	t.Run("should give a different directory on every call", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()

		// when
		first, removeFirst, err := workspace.NewTemporary()
		require.NoError(t, err)
		defer removeFirst()
		second, removeSecond, err := workspace.NewTemporary()
		require.NoError(t, err)
		defer removeSecond()

		// then
		assert.NotEqual(t, first, second)
	})

	t.Run("should remove the directory and everything inside it", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()
		path, remove, err := workspace.NewTemporary()
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(path, "frame_000000.png"), []byte("x"), 0o644))

		// when
		err = remove()

		// then
		require.NoError(t, err)
		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should not fail when remove is called on a directory already gone", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()
		_, remove, err := workspace.NewTemporary()
		require.NoError(t, err)
		require.NoError(t, remove())

		// when
		err = remove()

		// then
		assert.NoError(t, err)
	})
}

func Test_OS_EnsureDirectory(t *testing.T) {
	t.Run("should create a directory, and any missing parent, that does not exist yet", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()
		path := filepath.Join(t.TempDir(), "kept", "intermediates")

		// when
		err := workspace.EnsureDirectory(path)

		// then
		require.NoError(t, err)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})

	t.Run("should not fail when the directory already exists", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()
		path := t.TempDir()

		// when
		err := workspace.EnsureDirectory(path)

		// then
		assert.NoError(t, err)
	})

	t.Run("should fail when the path is already a file", func(t *testing.T) {
		// given
		workspace := workingdir.NewOS()
		path := filepath.Join(t.TempDir(), "not-a-directory")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

		// when
		err := workspace.EnsureDirectory(path)

		// then
		assert.Error(t, err)
	})
}
