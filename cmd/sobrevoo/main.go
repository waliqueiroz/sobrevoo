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
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/pngfile"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/simplifier"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/smoother"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/tiledecoder"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/trackparser"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/videoencoder"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/videofile"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/workingdir"
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
	cameraPlanService := application.NewCameraPlanService(trackService, cameraPlanExporter, cameraPlanReader, domainCameraTuning(cfg.CameraTuning))

	geoDataInspector := geodatainspector.New()
	elevationReader := elevationreader.NewGeoTIFF()
	geoDataRepository := jsonfile.NewGeoDataRepository(cfg.RegistryPath)
	geoDataFileChecker := filechecker.NewOS()
	geoDataService := application.NewGeoDataService(geoDataRepository, geoDataInspector, geoDataFileChecker, trackService, elevationReader)

	baseMapReader := basemapreader.NewMBTiles()
	geoSliceExporter := zipfile.NewGeoSliceExporter()
	geoSliceReader := zipfile.NewGeoSliceReader()
	geoSliceService := application.NewGeoSliceService(geoDataRepository, geoDataFileChecker, baseMapReader, elevationReader, geoSliceExporter, geoSliceReader, domainSliceTuning(cfg.SliceTuning), domainCameraTuning(cfg.CameraTuning))

	defaultResolution, err := domainRenderResolution(cfg.RenderDefaults)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	defaultAppearance, err := domainAppearance(cfg.RenderDefaults)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	defaultOverlay, err := domainOverlayConfig(cfg.RenderDefaults)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	tileDecoder := tiledecoder.NewRaster()
	frameExporter := pngfile.NewFrameExporter()
	frameRepository := pngfile.NewFrameRepository()
	frameService := application.NewFrameService(tileDecoder, frameRepository, frameExporter, domainRenderTuning(cfg.RenderTuning), domainSliceTuning(cfg.SliceTuning))

	videoEncoder := videoencoder.NewFFmpeg(cfg.FFmpegBinary)
	videoExporter := videofile.NewVideoExporter()
	videoService := application.NewVideoService(frameRepository, videoEncoder, videoExporter)

	workspace := workingdir.NewOS()
	flightService := application.NewFlightService(cameraPlanService, geoSliceService, frameService, videoService, workspace)

	geoDataCommand := cli.NewGeoDataCommand()
	geoDataCommand.AddCommand(cli.NewGeoDataRegisterCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataCheckCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataListCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataRemoveCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataClearCommand(geoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataSliceCommand(cameraPlanService, geoSliceService))
	geoDataCommand.AddCommand(cli.NewGeoDataElevationCommand(geoDataService))

	renderCommand := cli.NewRenderCommand()
	renderCommand.AddCommand(cli.NewRenderFrameCommand(cameraPlanService, geoSliceService, frameService, defaultResolution, defaultAppearance, defaultOverlay))
	renderCommand.AddCommand(cli.NewRenderAllCommand(cameraPlanService, geoSliceService, frameService, defaultResolution, defaultAppearance, defaultOverlay))

	root := cli.NewRootCommand(resolveVersion())
	root.AddCommand(cli.NewInspectCommand(trackService, domainLevel(cfg.DefaultLevel)))
	root.AddCommand(cli.NewPlanCommand(cameraPlanService, domainPlanParameters(cfg.PlanDefaults, cfg.DefaultLevel)))
	root.AddCommand(geoDataCommand)
	root.AddCommand(renderCommand)
	root.AddCommand(cli.NewVideoCommand(cameraPlanService, videoService, domainVideoQuality(cfg.VideoDefaults.Quality)))
	root.AddCommand(cli.NewFlightCommand(flightService, domainPlanParameters(cfg.PlanDefaults, cfg.DefaultLevel), defaultResolution, defaultAppearance, defaultOverlay, domainVideoQuality(cfg.VideoDefaults.Quality)))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}

	return 0
}
