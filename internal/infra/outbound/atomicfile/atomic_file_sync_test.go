package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func replaceSync(t *testing.T, replacement func(*os.File) error) {
	t.Helper()
	original := syncFile
	syncFile = replacement
	t.Cleanup(func() { syncFile = original })
}

// These tests exercise the flush to stable storage Publish does before it
// publishes a file: without it, a power failure right after the rename could
// leave an empty file under the final name.
func Test_Publish_Sync(t *testing.T) {
	t.Run("should sync the temporary file once before publishing it", func(t *testing.T) {
		// given
		calls := 0
		replaceSync(t, func(f *os.File) error {
			calls++
			return f.Sync()
		})
		path := filepath.Join(t.TempDir(), "out.bin")

		// when
		err := Publish(path, false, writeString("content"))

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("should refuse to publish when the sync fails, leaving no file behind", func(t *testing.T) {
		// given
		replaceSync(t, func(*os.File) error { return errors.New("disk on fire") })
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")

		// when
		err := Publish(path, false, writeString("content"))

		// then
		require.ErrorIs(t, err, ErrInvalid)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Empty(t, entries)
	})

	t.Run("should keep the previous file intact when the sync fails with overwrite", func(t *testing.T) {
		// given
		replaceSync(t, func(*os.File) error { return errors.New("disk on fire") })
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := Publish(path, true, writeString("new"))

		// then
		require.ErrorIs(t, err, ErrInvalid)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
	})
}

func Test_PublishPath_Sync(t *testing.T) {
	t.Run("should sync the file once, after the callback wrote it and before publishing it", func(t *testing.T) {
		// given
		var order []string
		replaceSync(t, func(f *os.File) error {
			order = append(order, "sync")
			return f.Sync()
		})
		path := filepath.Join(t.TempDir(), "out.bin")

		// when
		err := PublishPath(path, false, func(temporary string) error {
			order = append(order, "produce")
			return os.WriteFile(temporary, []byte("content"), 0o600)
		})

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"produce", "sync"}, order)
	})

	t.Run("should refuse to publish when the sync fails, leaving no file behind", func(t *testing.T) {
		// given
		replaceSync(t, func(*os.File) error { return errors.New("disk on fire") })
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")

		// when
		err := PublishPath(path, false, func(temporary string) error {
			return os.WriteFile(temporary, []byte("content"), 0o600)
		})

		// then
		require.ErrorIs(t, err, ErrInvalid)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Empty(t, entries)
	})
}
