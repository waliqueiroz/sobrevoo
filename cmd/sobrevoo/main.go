// Command sobrevoo is Sobrevoo's CLI entrypoint: the composition root that
// wires the concrete adapters (outbound and inbound) around the core use
// cases. This is the only place allowed to know about every concrete piece
// at once.
package main

import (
	"fmt"
	"os"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/basemapreader"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/elevationreader"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/filechecker"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/geodatainspector"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/jsonfile"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/simplifier"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/smoother"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/trackparser"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/zipfile"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}

	parser := trackparser.NewGPX()
	douglasPeucker := simplifier.NewDouglasPeucker()
	catmullRom := smoother.NewCatmullRom()
	trackService := application.NewTrackService(parser, douglasPeucker, catmullRom, cfg.MinPoints, cfg.MaxPlausibleSpeedKmh)

	cameraPlanExporter := jsonfile.NewCameraPlanExporter()
	cameraPlanReader := jsonfile.NewCameraPlanReader()
	cameraPlanService := application.NewCameraPlanService(trackService, cameraPlanExporter, cameraPlanReader, domainLevel(cfg.DefaultLevel), domainCameraTuning(cfg.CameraTuning))

	geoDataInspector := geodatainspector.New()
	elevationReader := elevationreader.NewGeoTIFF()
	geoDataRepository := jsonfile.NewGeoDataRepository(cfg.RegistryPath)
	geoDataFileChecker := filechecker.NewOS()
	geoDataService := application.NewGeoDataService(geoDataRepository, geoDataInspector, geoDataFileChecker, trackService, elevationReader)

	baseMapReader := basemapreader.NewMBTiles()
	geoSliceExporter := zipfile.NewGeoSliceExporter()
	geoSliceService := application.NewGeoSliceService(geoDataRepository, geoDataFileChecker, baseMapReader, elevationReader, geoSliceExporter, domainSliceTuning(cfg.SliceTuning), domainCameraTuning(cfg.CameraTuning))

	geoDataCommand := cli.NewGeoDataCommand()
	geoDataCommand.AddCommand(cli.NewGeoDataRegisterCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataCheckCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataListCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataRemoveCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataSliceCommand(cameraPlanService, geoSliceService))
	geoDataCommand.AddCommand(cli.NewGeoDataElevationCommand(geoDataService))

	root := cli.NewRootCommand()
	root.AddCommand(cli.NewInspectCommand(trackService, domainLevel(cfg.DefaultLevel)))
	root.AddCommand(cli.NewPlanCommand(cameraPlanService, domainPlanParameters(cfg.PlanDefaults)))
	root.AddCommand(geoDataCommand)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}

	return 0
}
