package atomicfile_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/atomicfile"
)

func writeContent(content string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := w.Write([]byte(content))
		return err
	}
}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

func Test_Publish(t *testing.T) {
	t.Run("should publish a new file with the content written by the callback", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "out.bin")

		// when
		err := atomicfile.Publish(path, false, writeContent("content"))

		// then
		require.NoError(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "content", string(content))
	})

	t.Run("should refuse an existing destination without overwrite, leaving it intact", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "out.bin")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := atomicfile.Publish(path, false, writeContent("new"))

		// then
		require.ErrorIs(t, err, atomicfile.ErrExists)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
		assert.Equal(t, []string{"out.bin"}, listDir(t, filepath.Dir(path)))
	})

	t.Run("should replace an existing destination entirely with overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "out.bin")
		require.NoError(t, os.WriteFile(path, []byte("old and much longer content"), 0o600))

		// when
		err := atomicfile.Publish(path, true, writeContent("new"))

		// then
		require.NoError(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "new", string(content))
	})

	t.Run("should report a missing directory as an invalid destination", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "missing", "out.bin")

		// when
		err := atomicfile.Publish(path, false, writeContent("content"))

		// then
		assert.ErrorIs(t, err, atomicfile.ErrInvalid)
	})

	t.Run("should leave nothing behind when the callback fails", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")
		callbackErr := errors.New("boom")

		// when
		err := atomicfile.Publish(path, false, func(io.Writer) error { return callbackErr })

		// then
		require.ErrorIs(t, err, callbackErr)
		assert.Empty(t, listDir(t, dir))
	})

	t.Run("should keep the previous destination intact when the callback fails with overwrite", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := atomicfile.Publish(path, true, func(w io.Writer) error {
			_, _ = w.Write([]byte("half"))
			return errors.New("boom")
		})

		// then
		require.Error(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
		assert.Equal(t, []string{"out.bin"}, listDir(t, dir))
	})

	t.Run("should leave only the published file after success", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		err := atomicfile.Publish(filepath.Join(dir, "out.bin"), false, writeContent("content"))

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"out.bin"}, listDir(t, dir))
	})

	t.Run("should create a readable file", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "out.bin")

		// when
		err := atomicfile.Publish(path, false, writeContent("content"))

		// then
		require.NoError(t, err)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})
}
