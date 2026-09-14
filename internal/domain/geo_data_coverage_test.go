package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
)

func Test_ComputeCoverage(t *testing.T) {
	t.Run("should report full coverage when a base map and an elevation source cover every point", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}
		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{baseMap}, []domain.GeoDataSource{elevation})

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
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}
		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{baseMap}, nil)

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
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		// when
		report := domain.ComputeCoverage(route, nil, []domain.GeoDataSource{elevation})

		// then
		assert.Equal(t, domain.CoverageStatusPartial, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingBaseMap, report.UncoveredSegments[0].Missing)
	})

	t.Run("should report a single uncovered segment with the coordinates of the point outside every registered source", func(t *testing.T) {
		// given
		covered := domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}
		route := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(60).WithLongitude(30).Build(), // outside "covered"
		}
		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).WithBoundingBox(covered).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(covered).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{baseMap}, []domain.GeoDataSource{elevation})

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
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		}

		// when
		report := domain.ComputeCoverage(route, nil, nil)

		// then
		assert.Equal(t, domain.CoverageStatusNone, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Empty(t, report.BaseMapSourcesUsed)
		assert.Empty(t, report.ElevationSourcesUsed)
	})

	t.Run("should pick the more specific of two overlapping base map sources", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(45.1).WithLongitude(15.1).Build(),
		}
		wide := build_domain.NewGeoDataSourceBuilder().WithName("regiao-ampla").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 90, MinLongitude: 0, MaxLongitude: 90}).Build()
		narrow := build_domain.NewGeoDataSourceBuilder().WithName("regiao-especifica").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{wide, narrow}, nil)

		// then
		require.Len(t, report.BaseMapSourcesUsed, 1)
		assert.Equal(t, "regiao-especifica", report.BaseMapSourcesUsed[0].Name)
	})

	t.Run("should break a tie between equally specific sources by the oldest RegisteredAt", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}
		route := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(45.1).WithLongitude(15.1).Build(),
		}
		older := build_domain.NewGeoDataSourceBuilder().WithName("mais-antigo").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(box).WithRegisteredAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).Build()
		newer := build_domain.NewGeoDataSourceBuilder().WithName("mais-novo").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(box).WithRegisteredAt(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{newer, older}, nil)

		// then
		require.Len(t, report.BaseMapSourcesUsed, 1)
		assert.Equal(t, "mais-antigo", report.BaseMapSourcesUsed[0].Name)
	})

	t.Run("should report coverage correctly for a track crossing the antimeridian", func(t *testing.T) {
		// given
		route := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
		}
		box := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}
		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("antimeridiano-mapa").WithType(domain.DataTypeBaseMap).WithBoundingBox(box).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("antimeridiano-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(box).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{baseMap}, []domain.GeoDataSource{elevation})

		// then
		assert.Equal(t, domain.CoverageStatusFull, report.Status)
	})

	t.Run("should list every base map source that covered at least one point, sorted by name", func(t *testing.T) {
		// given: two non-overlapping base map sources, each covering one of the two points
		route := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
		}
		sourceB := build_domain.NewGeoDataSourceBuilder().WithName("b-mapa").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}).Build()
		sourceA := build_domain.NewGeoDataSourceBuilder().WithName("a-mapa").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 2, MinLongitude: 0, MaxLongitude: 2}).Build()

		// when
		report := domain.ComputeCoverage(route, []domain.GeoDataSource{sourceB, sourceA}, nil)

		// then
		require.Len(t, report.BaseMapSourcesUsed, 2)
		assert.Equal(t, "a-mapa", report.BaseMapSourcesUsed[0].Name)
		assert.Equal(t, "b-mapa", report.BaseMapSourcesUsed[1].Name)
	})
}
