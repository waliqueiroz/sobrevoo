package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_SliceRegions_BaseMaps(t *testing.T) {
	t.Run("should list each base map once, in the order of its first region", func(t *testing.T) {
		// given
		regions := domain.SliceRegions{
			{Box: box(0, 1, 0, 1), BaseMap: baseMap("b", box(0, 9, 0, 9))},
			{Box: box(0, 1, 1, 2), BaseMap: baseMap("a", box(0, 9, 0, 9))},
			{Box: box(1, 2, 0, 1), BaseMap: baseMap("b", box(0, 9, 0, 9))},
		}

		// when
		maps := regions.BaseMaps()

		// then
		require.Len(t, maps, 2)
		assert.Equal(t, "b", maps[0].Name)
		assert.Equal(t, "a", maps[1].Name)
	})
}

func Test_SliceRegions_Sources(t *testing.T) {
	t.Run("should list each source of both types once, sorted by name", func(t *testing.T) {
		// given
		regions := domain.SliceRegions{
			{Box: box(0, 1, 0, 1), BaseMap: baseMap("mapa-b", box(0, 9, 0, 9)), Elevation: elevation("relevo", box(0, 9, 0, 9))},
			{Box: box(0, 1, 1, 2), BaseMap: baseMap("mapa-a", box(0, 9, 0, 9)), Elevation: elevation("relevo", box(0, 9, 0, 9))},
			{Box: box(1, 2, 0, 1), BaseMap: baseMap("mapa-b", box(0, 9, 0, 9)), Elevation: elevation("relevo", box(0, 9, 0, 9))},
		}

		// when
		sources := regions.Sources()

		// then
		require.Len(t, sources, 3)
		assert.Equal(t, "mapa-a", sources[0].Name)
		assert.Equal(t, "mapa-b", sources[1].Name)
		assert.Equal(t, "relevo", sources[2].Name)
	})

	t.Run("should add nothing for a type no source covers in a region", func(t *testing.T) {
		// given: the second region has no base map
		regions := domain.SliceRegions{
			{Box: box(0, 1, 0, 1), BaseMap: baseMap("mapa", box(0, 1, 0, 1)), Elevation: elevation("relevo", box(0, 9, 0, 9))},
			{Box: box(1, 2, 0, 1), Elevation: elevation("relevo", box(0, 9, 0, 9))},
		}

		// when
		sources := regions.Sources()

		// then
		require.Len(t, sources, 2)
		assert.Equal(t, "mapa", sources[0].Name)
		assert.Equal(t, "relevo", sources[1].Name)
	})
}

func Test_SliceRegions_TilesFor(t *testing.T) {
	// at level 4 a tile is 22.5° wide: longitudes 1° to 40° are columns 8 and 9, latitudes -30° to -1° rows 8 and 9
	t.Run("should ask the only base map for every tile of the region, sorted by column and row", func(t *testing.T) {
		// given
		regions := domain.SliceRegions{{Box: box(-30, -1, 1, 40), BaseMap: baseMap("map", box(-90, 90, -180, 180))}}

		// when
		tiles := regions.TilesFor(map[string]int{"map": 4})

		// then
		assert.Equal(t, []domain.TileID{
			{Level: 4, X: 8, Y: 8}, {Level: 4, X: 8, Y: 9},
			{Level: 4, X: 9, Y: 8}, {Level: 4, X: 9, Y: 9},
		}, tiles["map"])
	})

	t.Run("should ask a base map for the tiles of all its regions, once each", func(t *testing.T) {
		// given
		regions := domain.SliceRegions{
			{Box: box(1, 5, 1, 30), BaseMap: baseMap("map", box(-90, 90, -180, 180))},
			{Box: box(1, 5, 10, 40), BaseMap: baseMap("map", box(-90, 90, -180, 180))},
		}

		// when
		tiles := regions.TilesFor(map[string]int{"map": 4})

		// then
		seen := map[domain.TileID]int{}
		for _, id := range tiles["map"] {
			seen[id]++
		}
		for id, times := range seen {
			assert.Equal(t, 1, times, "tile %+v", id)
		}
		assert.Len(t, seen, 2)
	})

	t.Run("should give a tile that covers two regions to the base map of the first one", func(t *testing.T) {
		// given: both maps at level 4, and the tile x=8 (0° to 22.5°) covers both regions
		regions := domain.SliceRegions{
			{Box: box(1, 5, 1, 10), BaseMap: baseMap("west", box(-90, 90, -180, 180))},
			{Box: box(1, 5, 10, 20), BaseMap: baseMap("east", box(-90, 90, -180, 180))},
		}

		// when
		tiles := regions.TilesFor(map[string]int{"west": 4, "east": 4})

		// then
		assert.Len(t, tiles["west"], 1)
		assert.Empty(t, tiles["east"])
	})

	t.Run("should ask each base map for its own level, even for the same place", func(t *testing.T) {
		// given
		regions := domain.SliceRegions{
			{Box: box(1, 5, 1, 10), BaseMap: baseMap("west", box(-90, 90, -180, 180))},
			{Box: box(1, 5, 10, 20), BaseMap: baseMap("east", box(-90, 90, -180, 180))},
		}

		// when
		tiles := regions.TilesFor(map[string]int{"west": 4, "east": 5})

		// then
		require.NotEmpty(t, tiles["west"])
		require.NotEmpty(t, tiles["east"])
		assert.Equal(t, 4, tiles["west"][0].Level)
		assert.Equal(t, 5, tiles["east"][0].Level)
	})
}
