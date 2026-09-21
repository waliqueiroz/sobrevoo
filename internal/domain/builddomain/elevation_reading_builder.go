package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// ElevationReadingBuilder builds an ElevationReading of 760 m at (-23.5,
// -46.6), from a registered elevation source.
type ElevationReadingBuilder struct {
	reading domain.ElevationReading
}

func NewElevationReadingBuilder() *ElevationReadingBuilder {
	return &ElevationReadingBuilder{
		reading: domain.ElevationReading{
			Latitude: -23.5, Longitude: -46.6,
			Meters: 760, HasValue: true,
			Source: NewGeoDataSourceBuilder().
				WithName("relevo").
				WithType(domain.DataTypeElevation).
				WithFormat(domain.DataFormatGeoTIFF).
				Build(),
			Row: 412, Col: 88,
		},
	}
}

func (b *ElevationReadingBuilder) WithCoordinate(latitude, longitude float64) *ElevationReadingBuilder {
	b.reading.Latitude, b.reading.Longitude = latitude, longitude
	return b
}

func (b *ElevationReadingBuilder) WithMeters(meters float64) *ElevationReadingBuilder {
	b.reading.Meters, b.reading.HasValue = meters, true
	return b
}

func (b *ElevationReadingBuilder) WithoutValue() *ElevationReadingBuilder {
	b.reading.Meters, b.reading.HasValue = 0, false
	return b
}

func (b *ElevationReadingBuilder) WithSource(source domain.GeoDataSource) *ElevationReadingBuilder {
	b.reading.Source = source
	return b
}

func (b *ElevationReadingBuilder) WithCell(row, col int) *ElevationReadingBuilder {
	b.reading.Row, b.reading.Col = row, col
	return b
}

func (b *ElevationReadingBuilder) Build() domain.ElevationReading {
	return b.reading
}
