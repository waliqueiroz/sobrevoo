package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// GeoSliceBuilder builds a GeoSlice out of a default TileSet and a default
// ElevationGrid.
type GeoSliceBuilder struct {
	area      domain.BoundingBox
	tileSets  []domain.TileSet
	elevation []domain.ElevationGrid
	planID    string
	contentID string
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

// WithPlanID sets the identification of the plan the slice was made from.
func (b *GeoSliceBuilder) WithPlanID(id string) *GeoSliceBuilder {
	b.planID = id
	return b
}

// WithContentID sets the identification of the slice file, which only the
// reader of a slice file knows.
func (b *GeoSliceBuilder) WithContentID(id string) *GeoSliceBuilder {
	b.contentID = id
	return b
}

func (b *GeoSliceBuilder) Build() domain.GeoSlice {
	slice := domain.NewGeoSlice(b.area, b.tileSets, b.elevation)
	slice.PlanID, slice.ContentID = b.planID, b.contentID
	return slice
}
