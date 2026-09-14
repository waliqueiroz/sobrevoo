package helper

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
