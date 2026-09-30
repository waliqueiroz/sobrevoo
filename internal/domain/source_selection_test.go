package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_SourceSelection_Resolve(t *testing.T) {
	t.Run("should leave both candidate lists unchanged when neither name is requested", func(t *testing.T) {
		// given
		baseMaps := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()}
		elevations := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("relevo").WithType(domain.DataTypeElevation).Build()}
		selection := domain.SourceSelection{}

		// when
		resolvedBaseMaps, resolvedElevations, err := selection.Resolve(baseMaps, elevations)

		// then
		require.NoError(t, err)
		assert.Equal(t, baseMaps, resolvedBaseMaps)
		assert.Equal(t, elevations, resolvedElevations)
	})

	t.Run("should narrow the base map candidates to exactly the one named, leaving elevation candidates untouched", func(t *testing.T) {
		// given
		mapaA := builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()
		mapaB := builddomain.NewGeoDataSourceBuilder().WithName("mapa-b").Build()
		elevations := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("relevo").WithType(domain.DataTypeElevation).Build()}
		name := "mapa-b"
		selection := domain.SourceSelection{BaseMapName: &name}

		// when
		resolvedBaseMaps, resolvedElevations, err := selection.Resolve([]domain.GeoDataSource{mapaA, mapaB}, elevations)

		// then
		require.NoError(t, err)
		assert.Equal(t, []domain.GeoDataSource{mapaB}, resolvedBaseMaps)
		assert.Equal(t, elevations, resolvedElevations)
	})

	t.Run("should narrow the elevation candidates to exactly the one named, leaving base map candidates untouched", func(t *testing.T) {
		// given
		baseMaps := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()}
		relevoA := builddomain.NewGeoDataSourceBuilder().WithName("relevo-a").WithType(domain.DataTypeElevation).Build()
		relevoB := builddomain.NewGeoDataSourceBuilder().WithName("relevo-b").WithType(domain.DataTypeElevation).Build()
		name := "relevo-b"
		selection := domain.SourceSelection{ElevationName: &name}

		// when
		resolvedBaseMaps, resolvedElevations, err := selection.Resolve(baseMaps, []domain.GeoDataSource{relevoA, relevoB})

		// then
		require.NoError(t, err)
		assert.Equal(t, baseMaps, resolvedBaseMaps)
		assert.Equal(t, []domain.GeoDataSource{relevoB}, resolvedElevations)
	})

	t.Run("should resolve both types independently when both names are requested", func(t *testing.T) {
		// given
		mapaA := builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()
		relevoA := builddomain.NewGeoDataSourceBuilder().WithName("relevo-a").WithType(domain.DataTypeElevation).Build()
		baseMapName, elevationName := "mapa-a", "relevo-a"
		selection := domain.SourceSelection{BaseMapName: &baseMapName, ElevationName: &elevationName}

		// when
		resolvedBaseMaps, resolvedElevations, err := selection.Resolve([]domain.GeoDataSource{mapaA}, []domain.GeoDataSource{relevoA})

		// then
		require.NoError(t, err)
		assert.Equal(t, []domain.GeoDataSource{mapaA}, resolvedBaseMaps)
		assert.Equal(t, []domain.GeoDataSource{relevoA}, resolvedElevations)
	})

	t.Run("should reject a base map name that is registered as elevation, citing the name", func(t *testing.T) {
		// given
		baseMaps := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()}
		relevo := builddomain.NewGeoDataSourceBuilder().WithName("relevo").WithType(domain.DataTypeElevation).Build()
		name := "relevo"
		selection := domain.SourceSelection{BaseMapName: &name}

		// when
		_, _, err := selection.Resolve(baseMaps, []domain.GeoDataSource{relevo})

		// then
		require.ErrorIs(t, err, domain.ErrDataSourceTypeMismatch)
		assert.ErrorContains(t, err, "relevo")
	})

	t.Run("should reject an elevation name that is registered as a base map, citing the name", func(t *testing.T) {
		// given
		mapaA := builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()
		elevations := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("relevo").WithType(domain.DataTypeElevation).Build()}
		name := "mapa-a"
		selection := domain.SourceSelection{ElevationName: &name}

		// when
		_, _, err := selection.Resolve([]domain.GeoDataSource{mapaA}, elevations)

		// then
		require.ErrorIs(t, err, domain.ErrDataSourceTypeMismatch)
		assert.ErrorContains(t, err, "mapa-a")
	})

	t.Run("should reject a name that is not registered as either type, citing the name", func(t *testing.T) {
		// given
		baseMaps := []domain.GeoDataSource{builddomain.NewGeoDataSourceBuilder().WithName("mapa-a").Build()}
		name := "nao-existe"
		selection := domain.SourceSelection{BaseMapName: &name}

		// when
		_, _, err := selection.Resolve(baseMaps, nil)

		// then
		require.ErrorIs(t, err, domain.ErrDataSourceNotRegistered)
		assert.ErrorContains(t, err, "nao-existe")
	})
}
