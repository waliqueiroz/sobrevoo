// Command samples writes the geo data samples that specs/004-geo-data-slice/
// quickstart.md uses: small, synthetic MBTiles and GeoTIFF files with content
// whose values are known, so the results of "geodata slice" and "geodata
// elevation" can be checked by hand. Nothing is downloaded and no real map is
// involved; the files are generated with the fixtures of test/helper.
//
//	go run ./test/samples --out specs/004-geo-data-slice/amostras
//
// The files, and the values a reader can expect from them:
//
//	mapa-sp.mbtiles          base map, levels 10 to 16, over São Paulo (the area of
//	                         the pedalada.gpx sample); three tiles of level 16
//	                         are missing: the one at the start of the track and
//	                         the ones east and south of it
//	relevo-sp.tif            int16, Deflate, meters; cell (row r, column c) holds
//	                         700 + (3r + 2c) mod 400, except a block of rows 240-249
//	                         and columns 300-309 that has no data
//	relevo-pes.tif           float32, feet: every cell holds 1000 ft (304.8 m)
//	relevo-projetado.tif     a vertical unit that is neither meters nor feet
//	mapa-corrompido.mbtiles  valid bounds, unusable tiles table
//	plano-enorme.json        a camera plan over São Paulo whose slice is too big:
//	                         it gets 300 m from the ground and 30 km from its marker
//	mapa-antimeridiano.mbtiles, relevo-antimeridiano.tif   around (-16.5, 180)
//	mapa-polar.mbtiles, relevo-polar.tif                   around (82, 15)
//
// For the fifth stage (specs/005-frame-rendering/quickstart.md) it also writes
// base maps made of images, which can be drawn:
//
//	mapa-imagem-sp.mbtiles             PNG tiles, levels 10 to 14, over the same area as
//	                                   mapa-sp.mbtiles, with three tiles of level 14 missing
//	                                   at the start of pedalada.gpx; every tile is a
//	                                   checkerboard whose tone depends on its position
//	mapa-vetorial-sp.mbtiles           the same area, with vector tiles (pbf), which are
//	                                   refused
//	relevo-sem-dado.tif                a GeoTIFF over that area in which no cell has a value
//	relevo-buraco.tif                  relevo-sp.tif with one more block with no value, of 15
//	                                   rows by 20 columns, at the start of pedalada.gpx (rows 345-359,
//	                                   columns 360-379)
//	mapa-imagem-antimeridiano.mbtiles,
//	mapa-imagem-polar.mbtiles          PNG tiles over the two areas above
//
// and, with --raster-over <track.gpx>, mapa-imagem-passeio.mbtiles: PNG tiles,
// levels 8 to 15, over the area of the track plus 0.1° on each side — smaller than
// the vector map of a real download, so it wins the choice of the smaller area.
//
// The program prints the coordinates of the known cells.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/trackparser"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

const (
	saoPauloNorth, saoPauloSouth = -23.2, -23.9
	saoPauloWest, saoPauloEast   = -47.0, -46.2
	cell                         = 0.001
)

func main() {
	out := flag.String("out", "amostras", "directory to write the samples to")
	rasterOver := flag.String("raster-over", "", "a GPX track: write mapa-imagem-passeio.mbtiles over its area, and nothing else")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}

	if *rasterOver != "" {
		writeRasterOver(*out, *rasterOver)
		return
	}

	// São Paulo
	startTile := tileOf(-23.5505, -46.6333, 16)
	missing := map[domain.TileID]bool{
		startTile: true,
		{Level: 16, X: startTile.X + 1, Y: startTile.Y}: true,
		{Level: 16, X: startTile.X, Y: startTile.Y + 1}: true,
	}
	write(*out, "mapa-sp.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds:  [4]float64{saoPauloWest, saoPauloSouth, saoPauloEast, saoPauloNorth},
		MinZoom: new(10), MaxZoom: new(16),
		Tiles: tiles(domain.BoundingBox{MinLatitude: saoPauloSouth, MaxLatitude: saoPauloNorth, MinLongitude: saoPauloWest, MaxLongitude: saoPauloEast}, 10, 16, missing),
	}))
	write(*out, "mapa-corrompido.mbtiles", helper.CorruptMBTilesOver(saoPauloWest, saoPauloSouth, saoPauloEast, saoPauloNorth))

	rows, cols := int(math.Round((saoPauloNorth-saoPauloSouth)/cell)), int(math.Round((saoPauloEast-saoPauloWest)/cell))
	values := make([][]float64, rows)
	for r := range values {
		values[r] = make([]float64, cols)
		for c := range values[r] {
			values[r][c] = 700 + float64((3*r+2*c)%400)
			if r >= 240 && r < 250 && c >= 300 && c < 310 {
				values[r][c] = -32768
			}
		}
	}
	write(*out, "relevo-sp.tif", helper.GeoTIFFWithSamples(helper.GeoTIFFSpec{
		Width: cols, Height: rows, OriginLon: saoPauloWest, OriginLat: saoPauloNorth, ScaleX: cell, ScaleY: cell,
		SampleType: helper.Int16, Values: values, Compression: helper.TIFFDeflate, Predictor: 2, RowsPerStrip: 50,
		NoData: new("-32768"),
	}))

	feet := make([][]float64, 100)
	for r := range feet {
		feet[r] = make([]float64, 100)
		for c := range feet[r] {
			feet[r][c] = 1000
		}
	}
	feetSpec := helper.GeoTIFFSpec{
		Width: 100, Height: 100, OriginLon: -46.7, OriginLat: -23.5, ScaleX: cell, ScaleY: cell,
		SampleType: helper.Float32, Values: feet, VerticalUnit: new(helper.UnitFoot),
	}
	write(*out, "relevo-pes.tif", helper.GeoTIFFWithSamples(feetSpec))
	feetSpec.VerticalUnit = new(helper.UnitOtherRaw)
	write(*out, "relevo-projetado.tif", helper.GeoTIFFWithSamples(feetSpec))

	// a plan whose area is large and whose camera gets close: too much to keep
	huge := helper.DefaultPlanFileSpec()
	huge.Frames = huge.Frames[:2]
	huge.DurationSeconds = 2.0 / 30
	huge.Frames[0].MarkerLat, huge.Frames[0].MarkerLon, huge.Frames[0].CameraToMarker = -23.55, -46.63, 300
	huge.Frames[1].MarkerLat, huge.Frames[1].MarkerLon, huge.Frames[1].CameraToMarker = -23.55, -46.6, 30000
	write(*out, "plano-enorme.json", helper.ValidPlanFile(huge))

	// around the antimeridian: lat -17 to -16, lon 179 to -179
	write(*out, "mapa-antimeridiano.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds: [4]float64{179, -17, -179, -16},
		Tiles:  tiles(domain.BoundingBox{MinLatitude: -17, MaxLatitude: -16, MinLongitude: 179, MaxLongitude: -179, CrossesAntimeridian: true}, 8, 13, nil),
	}))
	write(*out, "relevo-antimeridiano.tif", helper.GeoTIFFWithSamples(helper.GeoTIFFSpec{
		Width: 1000, Height: 500, OriginLon: 179, OriginLat: -16, ScaleX: 0.002, ScaleY: 0.002,
		SampleType: helper.Int16, Values: ramp(500, 1000), Compression: helper.TIFFDeflate, RowsPerStrip: 50,
	}))

	// high latitude: lat 81 to 83, lon 10 to 20
	write(*out, "mapa-polar.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds: [4]float64{10, 81, 20, 83},
		Tiles:  tiles(domain.BoundingBox{MinLatitude: 81, MaxLatitude: 83, MinLongitude: 10, MaxLongitude: 20}, 6, 11, nil),
	}))
	write(*out, "relevo-polar.tif", helper.GeoTIFFWithSamples(helper.GeoTIFFSpec{
		Width: 1000, Height: 500, OriginLon: 10, OriginLat: 83, ScaleX: 0.01, ScaleY: 0.004,
		SampleType: helper.Int16, Values: ramp(500, 1000), Compression: helper.TIFFDeflate, RowsPerStrip: 50,
	}))

	// base maps made of images, for the fifth stage
	saoPaulo := domain.BoundingBox{MinLatitude: saoPauloSouth, MaxLatitude: saoPauloNorth, MinLongitude: saoPauloWest, MaxLongitude: saoPauloEast}
	start14 := tileOf(-23.5505, -46.6333, 14)
	missing14 := map[domain.TileID]bool{
		start14: true,
		{Level: 14, X: start14.X + 1, Y: start14.Y}: true,
		{Level: 14, X: start14.X, Y: start14.Y + 1}: true,
	}
	write(*out, "mapa-imagem-sp.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds:  [4]float64{saoPauloWest, saoPauloSouth, saoPauloEast, saoPauloNorth},
		MinZoom: new(10), MaxZoom: new(14),
		Tiles: imageTiles(saoPaulo, 10, 14, missing14),
	}))
	write(*out, "mapa-vetorial-sp.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds:  [4]float64{saoPauloWest, saoPauloSouth, saoPauloEast, saoPauloNorth},
		MinZoom: new(10), MaxZoom: new(14), Format: "pbf",
		Tiles: tiles(saoPaulo, 10, 14, nil),
	}))
	// relevo-sp.tif with another block with no value, under the start of pedalada.gpx
	withBlock := make([][]float64, rows)
	for r := range withBlock {
		withBlock[r] = append([]float64(nil), values[r]...)
		for c := range withBlock[r] {
			if r >= 345 && r < 360 && c >= 360 && c < 380 {
				withBlock[r][c] = -32768
			}
		}
	}
	write(*out, "relevo-buraco.tif", helper.GeoTIFFWithSamples(helper.GeoTIFFSpec{
		Width: cols, Height: rows, OriginLon: saoPauloWest, OriginLat: saoPauloNorth, ScaleX: cell, ScaleY: cell,
		SampleType: helper.Int16, Values: withBlock, Compression: helper.TIFFDeflate, Predictor: 2, RowsPerStrip: 50,
		NoData: new("-32768"),
	}))
	empty := make([][]float64, rows)
	for r := range empty {
		empty[r] = make([]float64, cols)
		for c := range empty[r] {
			empty[r][c] = -32768
		}
	}
	write(*out, "relevo-sem-dado.tif", helper.GeoTIFFWithSamples(helper.GeoTIFFSpec{
		Width: cols, Height: rows, OriginLon: saoPauloWest, OriginLat: saoPauloNorth, ScaleX: cell, ScaleY: cell,
		SampleType: helper.Int16, Values: empty, Compression: helper.TIFFDeflate, Predictor: 2, RowsPerStrip: 50,
		NoData: new("-32768"),
	}))
	write(*out, "mapa-imagem-antimeridiano.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds: [4]float64{179, -17, -179, -16},
		Tiles:  imageTiles(domain.BoundingBox{MinLatitude: -17, MaxLatitude: -16, MinLongitude: 179, MaxLongitude: -179, CrossesAntimeridian: true}, 8, 12, nil),
	}))
	write(*out, "mapa-imagem-polar.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds: [4]float64{10, 81, 20, 83},
		Tiles:  imageTiles(domain.BoundingBox{MinLatitude: 81, MaxLatitude: 83, MinLongitude: 10, MaxLongitude: 20}, 6, 10, nil),
	}))

	fmt.Println("Known cells of relevo-sp.tif (cell row, column: latitude, longitude of its center):")
	known := func(label string, r, c int) {
		lat := saoPauloNorth - (float64(r)+0.5)*cell
		lon := saoPauloWest + (float64(c)+0.5)*cell
		fmt.Printf("  %-22s (%d, %d): --lat %.4f --lon %.4f", label, r, c, lat, lon)
		if values[r][c] == -32768 {
			fmt.Println("  -> no value")
		} else {
			fmt.Printf("  -> %.0f m\n", values[r][c])
		}
	}
	known("value", 100, 200)
	known("another value", 400, 500)
	known("no data", 245, 305)
	fmt.Println("  outside any relief:    --lat 0 --lon 0")
	fmt.Println("Cell of relevo-pes.tif: --lat -23.5505 --lon -46.6333 -> 304.8 m")
	fmt.Println("Antimeridian:           --lat -16.5 --lon 180  and  --lat -16.5 --lon -180 give the same answer")
}

// ramp is a rows × cols grid whose values grow with the column.
func ramp(rows, cols int) [][]float64 {
	values := make([][]float64, rows)
	for r := range values {
		values[r] = make([]float64, cols)
		for c := range values[r] {
			values[r][c] = float64(100 + (r+c)%300)
		}
	}
	return values
}

// tiles lists every tile that covers area, in the levels from low to high,
// except the ones in missing.
func tiles(area domain.BoundingBox, low, high int, missing map[domain.TileID]bool) []helper.MBTile {
	var list []helper.MBTile
	for level := low; level <= high; level++ {
		for _, span := range area.TileRange(level) {
			for x := span.MinX; x <= span.MaxX; x++ {
				for y := span.MinY; y <= span.MaxY; y++ {
					if missing[domain.TileID{Level: level, X: x, Y: y}] {
						continue
					}
					list = append(list, helper.MBTile{Z: level, X: x, Y: y, Data: helper.TileData(level, x, y)})
				}
			}
		}
	}
	return list
}

// imageTiles is like tiles, with an image in each tile that shows where the tile is.
func imageTiles(area domain.BoundingBox, low, high int, missing map[domain.TileID]bool) []helper.MBTile {
	list := tiles(area, low, high, missing)
	for i := range list {
		list[i].Data = helper.PositionPNGTile(list[i].Z, list[i].X, list[i].Y)
	}
	return list
}

// writeRasterOver writes a base map of images over the area of a GPX track, plus
// 0.1° on each side.
func writeRasterOver(dir, gpxPath string) {
	file, err := os.Open(gpxPath)
	if err != nil {
		fail(err)
	}
	defer file.Close()

	track, err := trackparser.NewGPX().Parse(file)
	if err != nil {
		fail(err)
	}
	box := domain.Route{Points: track.Points}.BoundingBox()

	const margin = 0.1
	area := domain.BoundingBox{
		MinLatitude: box.MinLatitude - margin, MaxLatitude: box.MaxLatitude + margin,
		MinLongitude: box.MinLongitude - margin, MaxLongitude: box.MaxLongitude + margin,
	}
	write(dir, "mapa-imagem-passeio.mbtiles", helper.MBTilesWithTiles(helper.MBTilesSpec{
		Bounds:  [4]float64{area.MinLongitude, area.MinLatitude, area.MaxLongitude, area.MaxLatitude},
		MinZoom: new(8), MaxZoom: new(15),
		Tiles: imageTiles(area, 8, 15, nil),
	}))
	fmt.Printf("area: lat %.4f to %.4f, lon %.4f to %.4f\n", area.MinLatitude, area.MaxLatitude, area.MinLongitude, area.MaxLongitude)
}

func tileOf(lat, lon float64, level int) domain.TileID {
	span := domain.BoundingBox{MinLatitude: lat, MaxLatitude: lat, MinLongitude: lon, MaxLongitude: lon}.TileRange(level)[0]
	return domain.TileID{Level: level, X: span.MinX, Y: span.MinY}
}

func write(dir, name string, content []byte) {
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s (%d bytes)\n", filepath.Join(dir, name), len(content))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
