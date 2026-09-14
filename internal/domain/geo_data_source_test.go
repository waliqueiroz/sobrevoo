package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_NewGeoDataSource(t *testing.T) {
	t.Run("should build a source from the name, path and what the inspector discovered, stamped with the current time", func(t *testing.T) {
		// given
		inspected := domain.InspectedGeoData{
			Format:      domain.DataFormatMBTiles,
			Type:        domain.DataTypeBaseMap,
			BoundingBox: domain.BoundingBox{MinLatitude: 40, MaxLatitude: 50, MinLongitude: 10, MaxLongitude: 20},
		}
		before := time.Now()

		// when
		source := domain.NewGeoDataSource("europa-central-mapa", "/data/mapa.mbtiles", inspected)

		// then
		after := time.Now()
		assert.Equal(t, "europa-central-mapa", source.Name)
		assert.Equal(t, "/data/mapa.mbtiles", source.Path)
		assert.Equal(t, domain.DataTypeBaseMap, source.Type)
		assert.Equal(t, domain.DataFormatMBTiles, source.Format)
		assert.Equal(t, inspected.BoundingBox, source.BoundingBox)
		assert.False(t, source.RegisteredAt.Before(before))
		assert.False(t, source.RegisteredAt.After(after))
	})
}
