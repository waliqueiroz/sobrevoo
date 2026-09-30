package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

type sliceCommandMocks struct {
	planService  *mockapplication.MockCameraPlanService
	sliceService *mockapplication.MockGeoSliceService
}

func newSliceCommandMocks(t *testing.T) sliceCommandMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	return sliceCommandMocks{
		planService:  mockapplication.NewMockCameraPlanService(mockCtrl),
		sliceService: mockapplication.NewMockGeoSliceService(mockCtrl),
	}
}

// executeGeoDataSliceCommand runs "geodata slice" with the given arguments and
// returns stdout, stderr and the resulting error. Both services are test
// doubles: this is a unit test of the CLI adapter alone.
func executeGeoDataSliceCommand(t *testing.T, m sliceCommandMocks, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := cli.NewGeoDataSliceCommand(m.planService, m.sliceService)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

func Test_GeoDataSliceCommand_Args(t *testing.T) {
	t.Run("should return a usage error when no plan file is given", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m)

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error when two plan files are given", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "a.json", "b.json")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error for an unknown flag", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--nonsense")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func executeSlice(t *testing.T, slice domain.GeoSlice) string {
	t.Helper()
	m := newSliceCommandMocks(t)
	m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
	m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(slice, nil)

	stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

	require.NoError(t, err)
	return stdout
}

func Test_GeoDataSliceCommand_Execute(t *testing.T) {
	t.Run("should load the plan and slice it, printing the summary of the slice", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().Build()
		map1 := builddomain.NewGeoDataSourceBuilder().WithName("sp-osm").Build()
		relief := builddomain.NewGeoDataSourceBuilder().WithName("srtm-sp").
			WithType(domain.DataTypeElevation).WithFormat(domain.DataFormatGeoTIFF).Build()
		tileSet := builddomain.NewTileSetBuilder().WithSource(map1).
			WithDetail(domain.DetailLevel{
				Ideal: 19, Chosen: 16, Min: 0, Max: 16,
				Reason:      "above the source's maximum level",
				Explanation: "nearest camera distance 300.0 m, area closest to the equator at latitude 23.48, tiles of at most 0.46 m/px",
			}).Build()
		grid := builddomain.NewElevationGridBuilder().WithSource(relief).WithValues(712, 800, 900, 1000, 1204, 800, 800, 800, 800).WithNoValueAt(0, 0).Build()
		slice := builddomain.NewGeoSliceBuilder().
			WithArea(domain.BoundingBox{MinLatitude: -23.61, MaxLatitude: -23.48, MinLongitude: -46.72, MaxLongitude: -46.53}).
			WithTileSets(tileSet).WithElevation(grid).Build()
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load("plan.json").Return(plan, nil)
		m.sliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Area: lat -23.6100 to -23.4800, lon -46.7200 to -46.5300\n"+
			"Base map detail (sp-osm): level 16 (ideal 19, source offers 0-16; above the source's maximum level)\n"+
			"  nearest camera distance 300.0 m, area closest to the equator at latitude 23.48, tiles of at most 0.46 m/px\n"+
			"Map tiles: 3 present\n"+
			"Elevation samples: 9 (1 without value)\n"+
			"Elevation range: 800.0 m - 1204.0 m\n"+
			"Sources:\n"+
			"  sp-osm (base map, MBTiles)\n"+
			"  srtm-sp (elevation, GeoTIFF)\n"+
			"Size: 45 B\n", stdout)
	})

	t.Run("should say when the area crosses the antimeridian", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().
			WithArea(domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}).
			Build()

		// when
		stdout := executeSlice(t, slice)

		// then
		assert.Contains(t, stdout, "Area: lat -1.0000 to 1.0000, lon 170.0000 to -170.0000 (crosses the antimeridian)\n")
	})

	t.Run("should print a line of detail for each base map used, by name", func(t *testing.T) {
		// given
		first := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("map-a").Build()).
			WithDetail(domain.DetailLevel{Ideal: 14, Chosen: 14, Min: 5, Max: 18, Reason: "within the source's range", Explanation: "explained a"}).Build()
		second := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("map-b").Build()).
			WithDetail(domain.DetailLevel{Ideal: 14, Chosen: 12, Min: 0, Max: 12, Reason: "above the source's maximum level", Explanation: "explained b"}).Build()

		// when
		stdout := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(first, second).Build())

		// then
		assert.Contains(t, stdout, "Base map detail (map-a): level 14 (ideal 14, source offers 5-18; within the source's range)\n  explained a\n")
		assert.Contains(t, stdout, "Base map detail (map-b): level 12 (ideal 14, source offers 0-12; above the source's maximum level)\n  explained b\n")
	})

	t.Run("should list each missing tile, saying where", func(t *testing.T) {
		// given
		tileSet := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("sp-osm").Build()).
			WithMissing(domain.TileID{Level: 16, X: 24122, Y: 36870}, domain.TileID{Level: 16, X: 24123, Y: 36870}).Build()

		// when
		stdout := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(tileSet).Build())

		// then
		assert.Contains(t, stdout, "Map tiles: 3 present, 2 missing\n"+
			"  missing: sp-osm level 16 x=24122 y=36870\n"+
			"  missing: sp-osm level 16 x=24123 y=36870\n")
	})

	t.Run("should list at most twenty missing tiles and count the rest", func(t *testing.T) {
		// given
		var missing []domain.TileID
		for x := 0; x < 25; x++ {
			missing = append(missing, domain.TileID{Level: 10, X: x, Y: 1})
		}
		tileSet := builddomain.NewTileSetBuilder().WithMissing(missing...).Build()

		// when
		stdout := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(tileSet).Build())

		// then
		assert.Contains(t, stdout, "  missing: europa-central-mapa level 10 x=19 y=1\n")
		assert.NotContains(t, stdout, "x=20 y=1")
		assert.Contains(t, stdout, "  ... and 5 more (all listed in the exported file)\n")
	})

	t.Run("should say so when no tile is present at all", func(t *testing.T) {
		// given
		tileSet := builddomain.NewTileSetBuilder().WithTiles().WithMissing(domain.TileID{Level: 16, X: 1, Y: 1}).Build()

		// when
		stdout := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(tileSet).Build())

		// then
		assert.Contains(t, stdout, "Map tiles: 0 present, 1 missing (no imagery in this slice)\n")
	})

	t.Run("should not mention samples without value when there are none", func(t *testing.T) {
		// when
		stdout := executeSlice(t, builddomain.NewGeoSliceBuilder().Build())

		// then
		assert.Contains(t, stdout, "Elevation samples: 9\n")
		assert.NotContains(t, stdout, "without value")
	})

	t.Run("should say there is no elevation range when no sample has a value", func(t *testing.T) {
		// given
		builder := builddomain.NewElevationGridBuilder()
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				builder.WithNoValueAt(row, col)
			}
		}

		// when
		stdout := executeSlice(t, builddomain.NewGeoSliceBuilder().WithElevation(builder.Build()).Build())

		// then
		assert.Contains(t, stdout, "Elevation samples: 9 (9 without value)\n")
		assert.Contains(t, stdout, "Elevation range: none (no sample has a value)\n")
	})

	t.Run("should print the size in the unit that fits", func(t *testing.T) {
		// given
		big := builddomain.NewTileSetBuilder().WithTiles(domain.Tile{ID: domain.TileID{Level: 1}, Data: make([]byte, 3*1024*1024)}).Build()
		small := builddomain.NewTileSetBuilder().WithTiles(domain.Tile{ID: domain.TileID{Level: 1}, Data: make([]byte, 5*1024)}).Build()
		huge := builddomain.NewTileSetBuilder().WithTiles(domain.Tile{ID: domain.TileID{Level: 1}, Data: make([]byte, 1)}).Build()

		// when
		inMiB := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(big).WithElevation().Build())
		inKiB := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(small).WithElevation().Build())
		inBytes := executeSlice(t, builddomain.NewGeoSliceBuilder().WithTileSets(huge).WithElevation().Build())

		// then
		assert.Contains(t, inMiB, "Size: 3.0 MiB\n")
		assert.Contains(t, inKiB, "Size: 5.0 KiB\n")
		assert.Contains(t, inBytes, "Size: 1 B\n")
	})

	t.Run("should return the error of loading the plan, unchanged, printing nothing", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, domain.ErrPlanFileInvalid)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.Equal(t, 17, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return the error of a plan format version that is not supported", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, domain.ErrPlanFormatVersionUnsupported)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

		// then
		assert.Equal(t, 18, cli.ExitCode(err))
	})

	t.Run("should return an area-not-covered error with the report and print nothing", func(t *testing.T) {
		// given
		notCovered := &domain.AreaNotCoveredError{Report: domain.CoverageReport{
			Status: domain.CoverageStatusPartial,
			UncoveredSegments: []domain.UncoveredSegment{
				{StartLatitude: 40.5, StartLongitude: 10.75, EndLatitude: 40.5, EndLongitude: 10.75, Missing: domain.MissingElevation},
			},
		}}
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.GeoSlice{}, notCovered)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

		// then
		require.ErrorIs(t, err, domain.ErrAreaNotCovered)
		assert.Equal(t, 19, cli.ExitCode(err))
		assert.ErrorContains(t, err, "missing elevation")
		assert.Empty(t, stdout)
	})

	t.Run("should pass the requested base map and elevation names as a SourceSelection", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		baseMap, elevation := "mapa-b", "relevo-a"
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().
			Generate(gomock.Any(), domain.SourceSelection{BaseMapName: &baseMap, ElevationName: &elevation}).
			Return(builddomain.NewGeoSliceBuilder().Build(), nil)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--base-map", "mapa-b", "--elevation", "relevo-a")

		// then
		require.NoError(t, err)
	})

	t.Run("should pass an empty SourceSelection when neither flag is given", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().
			Generate(gomock.Any(), domain.SourceSelection{}).
			Return(builddomain.NewGeoSliceBuilder().Build(), nil)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

		// then
		require.NoError(t, err)
	})

	t.Run("should map a requested source of the wrong type to exit code 56", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.GeoSlice{}, domain.ErrDataSourceTypeMismatch)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--base-map", "relevo-a")

		// then
		require.Error(t, err)
		assert.Equal(t, 56, cli.ExitCode(err))
	})
}

func Test_GeoDataSliceCommand_Export(t *testing.T) {
	t.Run("should export the slice before printing anything, then say where it went", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().Build()
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(slice, nil)
		m.sliceService.EXPECT().Export(slice, "out.zip", false).Return(nil)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--export", "out.zip")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Area: ")
		assert.True(t, strings.HasSuffix(stdout, "Slice written to out.zip\n"))
	})

	t.Run("should ask the export to overwrite with --overwrite", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(builddomain.NewGeoSliceBuilder().Build(), nil)
		m.sliceService.EXPECT().Export(gomock.Any(), "out.zip", true).Return(nil)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--export", "out.zip", "--overwrite")

		// then
		assert.NoError(t, err)
	})

	t.Run("should not export without --export", func(t *testing.T) {
		// given: no Export expectation, so the mock fails the test if it is called
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(builddomain.NewGeoSliceBuilder().Build(), nil)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json")

		// then
		require.NoError(t, err)
		assert.NotContains(t, stdout, "Slice written")
	})

	t.Run("should print no summary when the destination already exists", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(builddomain.NewGeoSliceBuilder().Build(), nil)
		m.sliceService.EXPECT().Export(gomock.Any(), gomock.Any(), false).Return(domain.ErrSliceDestinationExists)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--export", "out.zip")

		// then
		assert.Equal(t, 23, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should print no summary when the destination is not valid", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(domain.CameraPlan{}, nil)
		m.sliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(builddomain.NewGeoSliceBuilder().Build(), nil)
		m.sliceService.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.ErrSliceDestinationInvalid)

		// when
		stdout, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--export", "missing/out.zip")

		// then
		assert.Equal(t, 24, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return a usage error for --overwrite without --export", func(t *testing.T) {
		// given
		m := newSliceCommandMocks(t)

		// when
		_, _, err := executeGeoDataSliceCommand(t, m, "plan.json", "--overwrite")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}
