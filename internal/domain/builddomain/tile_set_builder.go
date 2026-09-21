package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// TileSetBuilder builds a TileSet. The default is three PNG tiles of level 16
// from a registered MBTiles base map that offers levels 0 to 16.
type TileSetBuilder struct {
	tileSet domain.TileSet
}

func NewTileSetBuilder() *TileSetBuilder {
	return &TileSetBuilder{
		tileSet: domain.TileSet{
			Source: NewGeoDataSourceBuilder().Build(),
			Detail: domain.DetailLevel{Ideal: 16, Chosen: 16, Min: 0, Max: 16, Reason: "within the source's range"},
			Format: "png",
			Tiles: []domain.Tile{
				{ID: domain.TileID{Level: 16, X: 100, Y: 200}, Data: []byte{1, 2, 3}},
				{ID: domain.TileID{Level: 16, X: 101, Y: 200}, Data: []byte{4, 5, 6, 7}},
				{ID: domain.TileID{Level: 16, X: 100, Y: 201}, Data: []byte{8, 9}},
			},
		},
	}
}

func (b *TileSetBuilder) WithSource(source domain.GeoDataSource) *TileSetBuilder {
	b.tileSet.Source = source
	return b
}

func (b *TileSetBuilder) WithDetail(detail domain.DetailLevel) *TileSetBuilder {
	b.tileSet.Detail = detail
	return b
}

func (b *TileSetBuilder) WithFormat(format string) *TileSetBuilder {
	b.tileSet.Format = format
	return b
}

func (b *TileSetBuilder) WithTiles(tiles ...domain.Tile) *TileSetBuilder {
	b.tileSet.Tiles = tiles
	return b
}

func (b *TileSetBuilder) WithMissing(missing ...domain.TileID) *TileSetBuilder {
	b.tileSet.Missing = missing
	return b
}

func (b *TileSetBuilder) Build() domain.TileSet {
	return b.tileSet
}
