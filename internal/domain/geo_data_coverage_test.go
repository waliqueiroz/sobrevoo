package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_Route_Coverage(t *testing.T) {
	t.Run("should report full coverage when a base map and an elevation source cover every point", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}
		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{baseMap}, []domain.GeoDataSource{elevation})

		// then
		assert.Equal(t, domain.CoverageStatusFull, report.Status)
		assert.Empty(t, report.UncoveredSegments)
		require.Len(t, report.BaseMapSourcesUsed, 1)
		assert.Equal(t, "europa-mapa", report.BaseMapSourcesUsed[0].Name)
		require.Len(t, report.ElevationSourcesUsed, 1)
		assert.Equal(t, "europa-relevo", report.ElevationSourcesUsed[0].Name)
	})

	t.Run("should report partial coverage when only a base map covers the whole track and no elevation is registered", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}
		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{baseMap}, nil)

		// then
		assert.Equal(t, domain.CoverageStatusPartial, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingElevation, report.UncoveredSegments[0].Missing)
		assert.NotEmpty(t, report.BaseMapSourcesUsed)
		assert.Empty(t, report.ElevationSourcesUsed)
	})

	t.Run("should report partial coverage when only an elevation source covers the whole track and no base map is registered", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		// when
		report := (domain.Route{Points: route}).Coverage(nil, []domain.GeoDataSource{elevation})

		// then
		assert.Equal(t, domain.CoverageStatusPartial, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingBaseMap, report.UncoveredSegments[0].Missing)
	})

	t.Run("should report a single uncovered segment with the coordinates of the point outside every registered source", func(t *testing.T) {
		// given
		covered := domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(60).WithLongitude(30).Build(), // outside "covered"
		}
		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).WithBoundingBox(covered).Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(covered).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{baseMap}, []domain.GeoDataSource{elevation})

		// then
		assert.Equal(t, domain.CoverageStatusPartial, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingBoth, report.UncoveredSegments[0].Missing)
		assert.Equal(t, 60.0, report.UncoveredSegments[0].StartLatitude)
		assert.Equal(t, 30.0, report.UncoveredSegments[0].StartLongitude)
		assert.Equal(t, 60.0, report.UncoveredSegments[0].EndLatitude)
		assert.Equal(t, 30.0, report.UncoveredSegments[0].EndLongitude)
	})

	t.Run("should report no coverage at all when no source is registered", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}

		// when
		report := (domain.Route{Points: route}).Coverage(nil, nil)

		// then
		assert.Equal(t, domain.CoverageStatusNone, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Empty(t, report.BaseMapSourcesUsed)
		assert.Empty(t, report.ElevationSourcesUsed)
	})

	t.Run("should pick the more specific of two overlapping base map sources", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(45.1).WithLongitude(15.1).Build(),
		}
		wide := builddomain.NewGeoDataSourceBuilder().WithName("regiao-ampla").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 90, MinLongitude: 0, MaxLongitude: 90}).Build()
		narrow := builddomain.NewGeoDataSourceBuilder().WithName("regiao-especifica").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{wide, narrow}, nil)

		// then
		require.Len(t, report.BaseMapSourcesUsed, 1)
		assert.Equal(t, "regiao-especifica", report.BaseMapSourcesUsed[0].Name)
	})

	t.Run("should break a tie between equally specific sources by the oldest RegisteredAt", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(45.1).WithLongitude(15.1).Build(),
		}
		older := builddomain.NewGeoDataSourceBuilder().WithName("mais-antigo").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(box).WithRegisteredAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).Build()
		newer := builddomain.NewGeoDataSourceBuilder().WithName("mais-novo").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(box).WithRegisteredAt(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{newer, older}, nil)

		// then
		require.Len(t, report.BaseMapSourcesUsed, 1)
		assert.Equal(t, "mais-antigo", report.BaseMapSourcesUsed[0].Name)
	})

	t.Run("should report coverage correctly for a track crossing the antimeridian", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
		}
		box := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}
		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("antimeridiano-mapa").WithType(domain.DataTypeBaseMap).WithBoundingBox(box).Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("antimeridiano-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(box).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{baseMap}, []domain.GeoDataSource{elevation})

		// then
		assert.Equal(t, domain.CoverageStatusFull, report.Status)
	})

	t.Run("should list every base map source that covered at least one point, sorted by name", func(t *testing.T) {
		// given: two non-overlapping base map sources, each covering one of the two points
		route := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
		}
		sourceB := builddomain.NewGeoDataSourceBuilder().WithName("b-mapa").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}).Build()
		sourceA := builddomain.NewGeoDataSourceBuilder().WithName("a-mapa").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 2, MinLongitude: 0, MaxLongitude: 2}).Build()

		// when
		report := (domain.Route{Points: route}).Coverage([]domain.GeoDataSource{sourceB, sourceA}, nil)

		// then
		require.Len(t, report.BaseMapSourcesUsed, 2)
		assert.Equal(t, "a-mapa", report.BaseMapSourcesUsed[0].Name)
		assert.Equal(t, "b-mapa", report.BaseMapSourcesUsed[1].Name)
	})
}
