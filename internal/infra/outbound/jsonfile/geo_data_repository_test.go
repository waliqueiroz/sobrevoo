package jsonfile_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/jsonfile"
)

func registryPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "registry.json")
}

func Test_Store_List(t *testing.T) {
	t.Run("should return no sources and no error when the registry file does not exist yet", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))

		// when
		sources, err := store.List()

		// then
		require.NoError(t, err)
		assert.Empty(t, sources)
	})
}

func Test_Store_Save(t *testing.T) {
	t.Run("should persist a new source so it can be listed afterward", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))
		source := build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").Build()

		// when
		err := store.Save(source)

		// then
		require.NoError(t, err)
		sources, err := store.List()
		require.NoError(t, err)
		require.Len(t, sources, 1)
		assert.Equal(t, source, sources[0])
	})

	t.Run("should replace the existing source when saving the same name again", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))
		require.NoError(t, store.Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").WithPath("/data/old.mbtiles").Build()))

		// when
		err := store.Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").WithPath("/data/new.mbtiles").Build())

		// then
		require.NoError(t, err)
		sources, err := store.List()
		require.NoError(t, err)
		require.Len(t, sources, 1)
		assert.Equal(t, "/data/new.mbtiles", sources[0].Path)
	})

	t.Run("should persist across separate Store instances pointed at the same path", func(t *testing.T) {
		// given
		path := registryPath(t)
		require.NoError(t, jsonfile.NewGeoDataRepository(path).Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").Build()))

		// when
		sources, err := jsonfile.NewGeoDataRepository(path).List()

		// then
		require.NoError(t, err)
		require.Len(t, sources, 1)
		assert.Equal(t, "europa-central-mapa", sources[0].Name)
	})
}

func Test_Store_FindByName(t *testing.T) {
	t.Run("should return the registered source with the given name", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))
		require.NoError(t, store.Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").Build()))

		// when
		source, found, err := store.FindByName("europa-central-mapa")

		// then
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "europa-central-mapa", source.Name)
	})

	t.Run("should report not found without an error when no source has that name", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))

		// when
		source, found, err := store.FindByName("nao-existe")

		// then
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, domain.GeoDataSource{}, source)
	})
}

func Test_Store_Delete(t *testing.T) {
	t.Run("should remove the registered source with the given name", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))
		require.NoError(t, store.Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").Build()))

		// when
		err := store.Delete("europa-central-mapa")

		// then
		require.NoError(t, err)
		sources, err := store.List()
		require.NoError(t, err)
		assert.Empty(t, sources)
	})

	t.Run("should leave other registered sources untouched", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))
		require.NoError(t, store.Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-mapa").Build()))
		require.NoError(t, store.Save(build_domain.NewGeoDataSourceBuilder().WithName("europa-central-relevo").WithType(domain.DataTypeElevation).Build()))

		// when
		err := store.Delete("europa-central-mapa")

		// then
		require.NoError(t, err)
		sources, err := store.List()
		require.NoError(t, err)
		require.Len(t, sources, 1)
		assert.Equal(t, "europa-central-relevo", sources[0].Name)
	})
}

func Test_Store_RoundTrip(t *testing.T) {
	t.Run("should preserve every field of a saved source, including RegisteredAt", func(t *testing.T) {
		// given
		store := jsonfile.NewGeoDataRepository(registryPath(t))
		registeredAt := time.Date(2026, time.March, 4, 10, 30, 0, 0, time.UTC)
		source := build_domain.NewGeoDataSourceBuilder().
			WithName("europa-central-relevo").
			WithPath("/data/europa-central.tif").
			WithType(domain.DataTypeElevation).
			WithFormat(domain.DataFormatGeoTIFF).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 1, MaxLatitude: 2, MinLongitude: 3, MaxLongitude: 4}).
			WithRegisteredAt(registeredAt).
			Build()

		// when
		require.NoError(t, store.Save(source))
		sources, err := store.List()

		// then
		require.NoError(t, err)
		require.Len(t, sources, 1)
		assert.Equal(t, source, sources[0])
		assert.True(t, registeredAt.Equal(sources[0].RegisteredAt))
	})
}
