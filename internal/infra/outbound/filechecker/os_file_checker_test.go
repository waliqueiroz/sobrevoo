package filechecker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/filechecker"
)

func Test_FileChecker_Exists(t *testing.T) {
	t.Run("should report true for a file that exists", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "present.mbtiles")
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o644))
		checker := filechecker.NewOS()

		// when
		exists := checker.Exists(path)

		// then
		assert.True(t, exists)
	})

	t.Run("should report false for a path that does not exist", func(t *testing.T) {
		// given
		checker := filechecker.NewOS()
		path := filepath.Join(t.TempDir(), "missing.mbtiles")

		// when
		exists := checker.Exists(path)

		// then
		assert.False(t, exists)
	})
}
