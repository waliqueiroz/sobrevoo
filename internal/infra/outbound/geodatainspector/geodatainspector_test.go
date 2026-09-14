package geodatainspector_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/geodatainspector"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func Test_Inspector_Inspect(t *testing.T) {
	t.Run("should reject a nonexistent path", func(t *testing.T) {
		// given
		inspector := geodatainspector.New()
		path := filepath.Join(t.TempDir(), "does-not-exist.mbtiles")

		// when
		_, err := inspector.Inspect(path)

		// then
		assert.ErrorIs(t, err, domain.ErrDataFileNotFound)
	})

	t.Run("should reject a path pointing at a directory", func(t *testing.T) {
		// given
		inspector := geodatainspector.New()

		// when
		_, err := inspector.Inspect(t.TempDir())

		// then
		assert.ErrorIs(t, err, domain.ErrDataFileUnreadable)
	})

	t.Run("should reject content that is neither MBTiles nor GeoTIFF", func(t *testing.T) {
		// given
		inspector := geodatainspector.New()
		path := writeFixture(t, "invalid.dat", helper.NotSQLiteContent())

		// when
		_, err := inspector.Inspect(path)

		// then
		assert.ErrorIs(t, err, domain.ErrUnsupportedDataFormat)
	})
}
