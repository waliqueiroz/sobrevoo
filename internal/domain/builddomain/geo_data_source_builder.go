package builddomain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type GeoDataSourceBuilder struct {
	source domain.GeoDataSource
}

func NewGeoDataSourceBuilder() *GeoDataSourceBuilder {
	return &GeoDataSourceBuilder{
		source: domain.GeoDataSource{
			Name:   "europa-central-mapa",
			Path:   "/data/europa-central.mbtiles",
			Type:   domain.DataTypeBaseMap,
			Format: domain.DataFormatMBTiles,
			BoundingBox: domain.BoundingBox{
				MinLatitude:  40.0,
				MaxLatitude:  50.0,
				MinLongitude: 10.0,
				MaxLongitude: 20.0,
			},
			RegisteredAt: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
	}
}

func (b *GeoDataSourceBuilder) WithName(name string) *GeoDataSourceBuilder {
	b.source.Name = name
	return b
}

func (b *GeoDataSourceBuilder) WithPath(path string) *GeoDataSourceBuilder {
	b.source.Path = path
	return b
}

func (b *GeoDataSourceBuilder) WithType(dataType domain.DataType) *GeoDataSourceBuilder {
	b.source.Type = dataType
	return b
}

func (b *GeoDataSourceBuilder) WithFormat(format domain.DataFormat) *GeoDataSourceBuilder {
	b.source.Format = format
	return b
}

func (b *GeoDataSourceBuilder) WithBoundingBox(boundingBox domain.BoundingBox) *GeoDataSourceBuilder {
	b.source.BoundingBox = boundingBox
	return b
}

func (b *GeoDataSourceBuilder) WithRegisteredAt(registeredAt time.Time) *GeoDataSourceBuilder {
	b.source.RegisteredAt = registeredAt
	return b
}

func (b *GeoDataSourceBuilder) Build() domain.GeoDataSource {
	return b.source
}
