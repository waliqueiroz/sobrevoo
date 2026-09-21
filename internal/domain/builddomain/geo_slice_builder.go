package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// GeoSliceBuilder builds a GeoSlice out of a default TileSet and a default
// ElevationGrid.
type GeoSliceBuilder struct {
	area      domain.BoundingBox
	tileSets  []domain.TileSet
	elevation []domain.ElevationGrid
}

func NewGeoSliceBuilder() *GeoSliceBuilder {
	return &GeoSliceBuilder{
		area: domain.BoundingBox{
			MinLatitude: -23.003, MaxLatitude: -23,
			MinLongitude: -47, MaxLongitude: -46.997,
		},
		tileSets:  []domain.TileSet{NewTileSetBuilder().Build()},
		elevation: []domain.ElevationGrid{NewElevationGridBuilder().Build()},
	}
}

func (b *GeoSliceBuilder) WithArea(area domain.BoundingBox) *GeoSliceBuilder {
	b.area = area
	return b
}

func (b *GeoSliceBuilder) WithTileSets(tileSets ...domain.TileSet) *GeoSliceBuilder {
	b.tileSets = tileSets
	return b
}

func (b *GeoSliceBuilder) WithElevation(grids ...domain.ElevationGrid) *GeoSliceBuilder {
	b.elevation = grids
	return b
}

func (b *GeoSliceBuilder) Build() domain.GeoSlice {
	return domain.NewGeoSlice(b.area, b.tileSets, b.elevation)
}
