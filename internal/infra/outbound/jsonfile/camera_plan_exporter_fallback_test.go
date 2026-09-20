package jsonfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// These tests exercise publishExclusive's fallback, used when a hard link
// cannot be made: linking from a temporary file that does not exist fails
// with an error other than "already exists", which is what an unsupported
// file system looks like from the caller's side.
func Test_publishExclusive_Fallback(t *testing.T) {
	t.Run("should create the destination directly when linking is not possible", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "plan.json")

		// when
		err := publishExclusive(filepath.Join(dir, "no-such-temporary"), path, []byte("content"))

		// then
		require.NoError(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "content", string(content))
	})

	t.Run("should still refuse an existing destination when linking is not possible", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "plan.json")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := publishExclusive(filepath.Join(dir, "no-such-temporary"), path, []byte("content"))

		// then
		require.ErrorIs(t, err, domain.ErrPlanDestinationExists)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
	})

	t.Run("should report an invalid destination when it cannot be created either", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		err := publishExclusive(filepath.Join(dir, "no-such-temporary"), filepath.Join(dir, "missing", "plan.json"), []byte("content"))

		// then
		assert.ErrorIs(t, err, domain.ErrPlanDestinationInvalid)
	})
}
