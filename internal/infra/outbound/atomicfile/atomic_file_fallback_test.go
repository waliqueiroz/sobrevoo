package atomicfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func failingLink(t *testing.T) {
	t.Helper()
	original := link
	link = func(string, string) error { return errors.New("links are not supported here") }
	t.Cleanup(func() { link = original })
}

func writeString(content string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := w.Write([]byte(content))
		return err
	}
}

// These tests exercise Publish's fallback, used when a hard link cannot be
// made (typically an unsupported file system): the link is replaced by one
// that always fails with an error other than "already exists".
func Test_Publish_Fallback(t *testing.T) {
	t.Run("should create the destination directly when linking is not possible", func(t *testing.T) {
		// given
		failingLink(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")

		// when
		err := Publish(path, false, writeString("content"))

		// then
		require.NoError(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "content", string(content))
		entries, _ := os.ReadDir(dir)
		assert.Len(t, entries, 1)
	})

	t.Run("should still refuse an existing destination when linking is not possible", func(t *testing.T) {
		// given
		failingLink(t)
		path := filepath.Join(t.TempDir(), "out.bin")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := Publish(path, false, writeString("content"))

		// then
		require.ErrorIs(t, err, ErrExists)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
	})

	t.Run("should report an invalid destination when it cannot be created either", func(t *testing.T) {
		// given
		failingLink(t)
		dir := t.TempDir()

		// when
		err := Publish(filepath.Join(dir, "missing", "out.bin"), false, writeString("content"))

		// then
		assert.ErrorIs(t, err, ErrInvalid)
	})
}
