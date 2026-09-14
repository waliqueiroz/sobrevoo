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
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/filechecker"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/geodatainspector"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/geodatastore/jsonfile"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/simplifier/douglaspeucker"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/smoother/catmullrom"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/trackparser"
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

	parser := trackparser.NewGPXParser()
	simplifier := douglaspeucker.New()
	smoother := catmullrom.New()
	inspectTrackService := application.NewInspectTrackService(parser, simplifier, smoother, cfg.MinPoints, cfg.MaxPlausibleSpeedKmh)

	geoDataInspector := geodatainspector.New()
	geoDataRegistry := jsonfile.New(cfg.RegistryPath)
	geoDataFileChecker := filechecker.New()
	registerGeoDataService := application.NewRegisterGeoDataService(geoDataRegistry, geoDataInspector)
	checkCoverageService := application.NewCheckCoverageService(parser, geoDataRegistry, geoDataFileChecker, cfg.MinPoints, cfg.MaxPlausibleSpeedKmh)
	listGeoDataService := application.NewListGeoDataService(geoDataRegistry, geoDataFileChecker)
	removeGeoDataService := application.NewRemoveGeoDataService(geoDataRegistry)

	geoDataCommand := cli.NewGeoDataCommand()
	geoDataCommand.AddCommand(cli.NewGeoDataRegisterCommand(registerGeoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataCheckCommand(checkCoverageService))
	geoDataCommand.AddCommand(cli.NewGeoDataListCommand(listGeoDataService))
	geoDataCommand.AddCommand(cli.NewGeoDataRemoveCommand(removeGeoDataService))

	root := cli.NewRootCommand()
	root.AddCommand(cli.NewInspectCommand(inspectTrackService, cfg.DefaultLevel))
	root.AddCommand(geoDataCommand)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}

	return 0
}
