package helper

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

// ValidMBTiles returns the raw content of a minimal, real MBTiles file (a
// SQLite database) whose "metadata" table has a "bounds" entry, per the
// MBTiles 1.3 specification's "minLon,minLat,maxLon,maxLat" format.
func ValidMBTiles() []byte {
	return buildMBTiles(func(db *sql.DB) {
		mustExec(db, `INSERT INTO metadata (name, value) VALUES ('bounds', '10.0,40.0,20.0,50.0')`)
		mustExec(db, `INSERT INTO metadata (name, value) VALUES ('format', 'png')`)
	})
}

// MBTilesWithoutBounds returns a real SQLite database with the MBTiles
// "metadata" table, but without a "bounds" entry.
func MBTilesWithoutBounds() []byte {
	return buildMBTiles(func(db *sql.DB) {
		mustExec(db, `INSERT INTO metadata (name, value) VALUES ('format', 'png')`)
	})
}

// MBTile is one tile of an MBTilesSpec, in the XYZ scheme (Y grows
// southwards); MBTilesWithTiles stores it with the TMS row MBTiles uses.
type MBTile struct {
	Z, X, Y int
	Data    []byte
}

// MBTilesSpec describes an MBTiles fixture with content.
type MBTilesSpec struct {
	// Bounds are minLon, minLat, maxLon, maxLat.
	Bounds [4]float64

	// MinZoom and MaxZoom go to the metadata table when set; when nil, only
	// the tiles table reveals the levels.
	MinZoom, MaxZoom *int

	// Format is the metadata "format" ("png" when empty).
	Format string

	Tiles []MBTile
}

// TileData returns deterministic bytes that differ for every tile, so a test
// can tell tiles apart.
func TileData(z, x, y int) []byte {
	return []byte("tile-" + strconv.Itoa(z) + "-" + strconv.Itoa(x) + "-" + strconv.Itoa(y))
}

// MBTilesWithTiles returns a real MBTiles file with the metadata and tiles
// of spec.
func MBTilesWithTiles(spec MBTilesSpec) []byte {
	return buildMBTiles(func(db *sql.DB) {
		bounds := fmt.Sprintf("%g,%g,%g,%g", spec.Bounds[0], spec.Bounds[1], spec.Bounds[2], spec.Bounds[3])
		mustExec(db, fmt.Sprintf(`INSERT INTO metadata (name, value) VALUES ('bounds', '%s')`, bounds))

		format := spec.Format
		if format == "" {
			format = "png"
		}
		mustExec(db, fmt.Sprintf(`INSERT INTO metadata (name, value) VALUES ('format', '%s')`, format))

		if spec.MinZoom != nil {
			mustExec(db, fmt.Sprintf(`INSERT INTO metadata (name, value) VALUES ('minzoom', '%d')`, *spec.MinZoom))
		}
		if spec.MaxZoom != nil {
			mustExec(db, fmt.Sprintf(`INSERT INTO metadata (name, value) VALUES ('maxzoom', '%d')`, *spec.MaxZoom))
		}

		mustExec(db, `CREATE TABLE tiles (zoom_level INTEGER, tile_column INTEGER, tile_row INTEGER, tile_data BLOB)`)
		// the index real MBTiles files have, so reading a tile is a lookup
		mustExec(db, `CREATE UNIQUE INDEX tile_index ON tiles (zoom_level, tile_column, tile_row)`)

		transaction, err := db.Begin()
		if err != nil {
			panic(fmt.Errorf("starting the MBTiles fixture transaction: %w", err))
		}
		for _, tile := range spec.Tiles {
			tmsRow := 1<<tile.Z - 1 - tile.Y
			if _, err := transaction.Exec(`INSERT INTO tiles (zoom_level, tile_column, tile_row, tile_data) VALUES (?, ?, ?, ?)`,
				tile.Z, tile.X, tmsRow, tile.Data); err != nil {
				panic(fmt.Errorf("inserting MBTiles fixture tile: %w", err))
			}
		}
		if err := transaction.Commit(); err != nil {
			panic(fmt.Errorf("committing the MBTiles fixture tiles: %w", err))
		}
	})
}

// CorruptMBTiles returns a SQLite database whose metadata has valid bounds
// (so the file is recognized and registered) but whose tiles table is
// unusable: it has the wrong columns, so reading tiles fails.
func CorruptMBTiles() []byte {
	return CorruptMBTilesOver(10, 40, 20, 50)
}

// CorruptMBTilesOver is CorruptMBTiles with the given bounds (west, south,
// east, north), so it covers the area of another map.
func CorruptMBTilesOver(west, south, east, north float64) []byte {
	return buildMBTiles(func(db *sql.DB) {
		mustExec(db, fmt.Sprintf(`INSERT INTO metadata (name, value) VALUES ('bounds', '%g,%g,%g,%g')`, west, south, east, north))
		mustExec(db, `CREATE TABLE tiles (garbage TEXT)`)
	})
}

// NotSQLiteContent returns bytes that are not a SQLite database at all —
// used to exercise MBTiles format detection failing before any SQL is run.
func NotSQLiteContent() []byte {
	return []byte("this is not a geographic data file")
}

// buildMBTiles creates a real, temporary SQLite database file with the
// MBTiles "metadata" table, lets populate fill it in, and returns the
// file's raw bytes.
func buildMBTiles(populate func(db *sql.DB)) []byte {
	dir, err := os.MkdirTemp("", "sobrevoo-mbtiles-fixture-*")
	if err != nil {
		panic(fmt.Errorf("creating temp dir for MBTiles fixture: %w", err))
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "fixture.mbtiles")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		panic(fmt.Errorf("opening MBTiles fixture database: %w", err))
	}

	mustExec(db, `CREATE TABLE metadata (name TEXT, value TEXT)`)
	populate(db)

	if err := db.Close(); err != nil {
		panic(fmt.Errorf("closing MBTiles fixture database: %w", err))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("reading MBTiles fixture file: %w", err))
	}

	return data
}

func mustExec(db *sql.DB, query string) {
	if _, err := db.Exec(query); err != nil {
		panic(fmt.Errorf("executing %q: %w", query, err))
	}
}
