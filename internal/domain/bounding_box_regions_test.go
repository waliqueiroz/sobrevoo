package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func box(minLat, maxLat, minLon, maxLon float64) domain.BoundingBox {
	return domain.BoundingBox{
		MinLatitude: minLat, MaxLatitude: maxLat,
		MinLongitude: minLon, MaxLongitude: maxLon,
		CrossesAntimeridian: minLon > maxLon,
	}
}

func source(name string, dataType domain.DataType, area domain.BoundingBox) domain.GeoDataSource {
	return builddomain.NewGeoDataSourceBuilder().WithName(name).WithType(dataType).WithBoundingBox(area).Build()
}

func baseMap(name string, area domain.BoundingBox) domain.GeoDataSource {
	return source(name, domain.DataTypeBaseMap, area)
}

func elevation(name string, area domain.BoundingBox) domain.GeoDataSource {
	return source(name, domain.DataTypeElevation, area)
}

func Test_BoundingBox_Regions(t *testing.T) {
	area := box(40, 41, 10, 11)

	t.Run("should give a single region when one source of each type covers everything", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		reliefs := []domain.GeoDataSource{elevation("dem", box(30, 50, 0, 20))}

		// when
		regions, route := area.Regions(maps, reliefs)

		// then
		require.Len(t, regions, 1)
		assert.Equal(t, area, regions[0].Box)
		assert.Equal(t, "map", regions[0].BaseMap.Name)
		assert.Equal(t, "dem", regions[0].Elevation.Name)
		require.Len(t, route.Points, 1)
		assert.InDelta(t, 40.5, route.Points[0].Latitude, 1e-9)
		assert.InDelta(t, 10.5, route.Points[0].Longitude, 1e-9)
		assert.Equal(t, domain.CoverageStatusFull, route.Coverage(maps, reliefs).Status)
	})

	t.Run("should split the area where the winning source changes, west to east", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		reliefs := []domain.GeoDataSource{
			elevation("west", box(40, 41, 10, 10.5)),
			elevation("east", box(40, 41, 10.5, 11)),
		}

		// when
		regions, route := area.Regions(maps, reliefs)

		// then
		require.Len(t, regions, 2)
		assert.Equal(t, box(40, 41, 10, 10.5), regions[0].Box)
		assert.Equal(t, "west", regions[0].Elevation.Name)
		assert.Equal(t, box(40, 41, 10.5, 11), regions[1].Box)
		assert.Equal(t, "east", regions[1].Elevation.Name)
		assert.Equal(t, "map", regions[1].BaseMap.Name)
		assert.Len(t, route.Points, 2)
	})

	t.Run("should order the regions from south to north and, within a row, from west to east", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		reliefs := []domain.GeoDataSource{
			elevation("north-east", box(40.5, 41, 10.5, 11)),
			elevation("south-west", box(40, 40.5, 10, 10.5)),
			elevation("north-west", box(40.5, 41, 10, 10.5)),
			elevation("south-east", box(40, 40.5, 10.5, 11)),
		}

		// when
		regions, _ := area.Regions(maps, reliefs)

		// then
		require.Len(t, regions, 4)
		assert.Equal(t, "south-west", regions[0].Elevation.Name)
		assert.Equal(t, "south-east", regions[1].Elevation.Name)
		assert.Equal(t, "north-west", regions[2].Elevation.Name)
		assert.Equal(t, "north-east", regions[3].Elevation.Name)
	})

	t.Run("should pick the smaller source where two overlap", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		reliefs := []domain.GeoDataSource{
			elevation("big", box(40, 41, 10, 11)),
			elevation("small", box(40, 41, 10, 10.5)),
		}

		// when
		regions, _ := area.Regions(maps, reliefs)

		// then
		require.Len(t, regions, 2)
		assert.Equal(t, "small", regions[0].Elevation.Name)
		assert.Equal(t, "big", regions[1].Elevation.Name)
	})

	t.Run("should pick the oldest source when two overlap with the same area", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		older := builddomain.NewGeoDataSourceBuilder().WithName("older").WithType(domain.DataTypeElevation).
			WithBoundingBox(box(30, 50, 0, 20)).WithRegisteredAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).Build()
		newer := builddomain.NewGeoDataSourceBuilder().WithName("newer").WithType(domain.DataTypeElevation).
			WithBoundingBox(box(30, 50, 0, 20)).WithRegisteredAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)).Build()

		// when
		regions, _ := area.Regions(maps, []domain.GeoDataSource{newer, older})

		// then
		require.Len(t, regions, 1)
		assert.Equal(t, "older", regions[0].Elevation.Name)
	})

	t.Run("should ignore sources that do not intersect the area", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20)), baseMap("faraway", box(-10, -5, 100, 110))}
		reliefs := []domain.GeoDataSource{elevation("dem", box(30, 50, 0, 20)), elevation("faraway", box(-10, -5, 100, 110))}

		// when
		regions, _ := area.Regions(maps, reliefs)

		// then
		assert.Len(t, regions, 1)
	})

	t.Run("should make coverage report the uncovered stretch at the center of the uncovered region", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		reliefs := []domain.GeoDataSource{elevation("west-only", box(40, 41, 10, 10.5))}

		// when
		regions, route := area.Regions(maps, reliefs)
		report := route.Coverage(maps, reliefs)

		// then
		require.Len(t, regions, 2)
		assert.Equal(t, domain.CoverageStatusPartial, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingElevation, report.UncoveredSegments[0].Missing)
		assert.InDelta(t, 40.5, report.UncoveredSegments[0].StartLatitude, 1e-9)
		assert.InDelta(t, 10.75, report.UncoveredSegments[0].StartLongitude, 1e-9)
	})

	t.Run("should report both types missing where nothing covers", func(t *testing.T) {
		// given
		var none []domain.GeoDataSource

		// when
		regions, route := area.Regions(none, none)
		report := route.Coverage(none, none)

		// then
		require.Len(t, regions, 1)
		assert.Equal(t, domain.CoverageStatusNone, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingBoth, report.UncoveredSegments[0].Missing)
	})

	t.Run("should find the regions of an area that crosses the antimeridian, west to east", func(t *testing.T) {
		// given
		crossing := box(0, 1, 175, -175)
		maps := []domain.GeoDataSource{baseMap("map", box(-10, 10, 170, -170))}
		reliefs := []domain.GeoDataSource{
			elevation("west", box(0, 1, 175, 178)),
			elevation("east", box(0, 1, 178, -170)),
		}

		// when
		regions, route := crossing.Regions(maps, reliefs)

		// then
		require.Len(t, regions, 2)
		assert.Equal(t, box(0, 1, 175, 178), regions[0].Box)
		assert.Equal(t, "west", regions[0].Elevation.Name)
		assert.Equal(t, box(0, 1, 178, -175), regions[1].Box)
		assert.True(t, regions[1].Box.CrossesAntimeridian)
		assert.Equal(t, "east", regions[1].Elevation.Name)
		assert.Equal(t, domain.CoverageStatusFull, route.Coverage(maps, reliefs).Status)
		assert.InDelta(t, 181.5, route.Points[1].Longitude+360, 1e-9)
	})

	t.Run("should partition the area: the regions add up to it and their centers are inside it", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map-a", box(39, 40.6, 9, 10.3)), baseMap("map-b", box(40.4, 42, 10.2, 12))}
		reliefs := []domain.GeoDataSource{elevation("dem-a", box(39, 40.5, 9, 10.7)), elevation("dem-b", box(40.3, 42, 10.5, 12))}

		// when
		regions, route := area.Regions(maps, reliefs)

		// then
		var total float64
		for i, region := range regions {
			total += region.Box.AreaDegrees()
			assert.True(t, area.Contains(route.Points[i].Latitude, route.Points[i].Longitude))
		}
		assert.InDelta(t, area.AreaDegrees(), total, 1e-9)
	})

	t.Run("should be deterministic", func(t *testing.T) {
		// given
		maps := []domain.GeoDataSource{baseMap("map", box(30, 50, 0, 20))}
		reliefs := []domain.GeoDataSource{elevation("a", box(40, 41, 10, 10.5)), elevation("b", box(40, 41, 10.5, 11))}
		expectedRegions, expectedRoute := area.Regions(maps, reliefs)

		for i := 0; i < 100; i++ {
			// when
			regions, route := area.Regions(maps, reliefs)

			// then
			assert.Equal(t, expectedRegions, regions)
			assert.Equal(t, expectedRoute, route)
		}
	})
}
