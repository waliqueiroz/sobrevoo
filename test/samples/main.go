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
// The program prints the coordinates of the known cells.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

const (
	saoPauloNorth, saoPauloSouth = -23.2, -23.9
	saoPauloWest, saoPauloEast   = -47.0, -46.2
	cell                         = 0.001
)

func main() {
	out := flag.String("out", "amostras", "directory to write the samples to")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
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
