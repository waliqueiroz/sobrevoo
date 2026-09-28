// Package basemapreader implements domain.BaseMapReader: reading the content
// of a registered base map file.
package basemapreader

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// defaultFormat is the tile image format of an MBTiles file that does not
// say (the specification's most common one).
const defaultFormat = "png"

// MBTiles reads MBTiles files (a SQLite database with a "tiles" table and a
// "metadata" table), the format the second stage registers as a base map.
type MBTiles struct{}

// NewMBTiles creates an MBTiles reader.
func NewMBTiles() MBTiles {
	return MBTiles{}
}

// Levels reports the range of zoom levels of the file: the metadata's
// minzoom and maxzoom, or, when it does not have them, the lowest and highest
// zoom among the tiles.
func (r MBTiles) Levels(path string) (domain.LevelRange, error) {
	db, err := open(path)
	if err != nil {
		return domain.LevelRange{}, err
	}
	defer db.Close()

	min, hasMin, err := metadataInt(db, "minzoom")
	if err != nil {
		return domain.LevelRange{}, unreadable(path, err)
	}
	max, hasMax, err := metadataInt(db, "maxzoom")
	if err != nil {
		return domain.LevelRange{}, unreadable(path, err)
	}
	if hasMin && hasMax {
		return domain.LevelRange{Min: min, Max: max}, nil
	}

	var low, high sql.NullInt64
	if err := db.QueryRow(`SELECT MIN(zoom_level), MAX(zoom_level) FROM tiles`).Scan(&low, &high); err != nil {
		return domain.LevelRange{}, unreadable(path, err)
	}
	if !low.Valid || !high.Valid {
		return domain.LevelRange{}, unreadable(path, errors.New("the file has no tiles"))
	}

	if !hasMin {
		min = int(low.Int64)
	}
	if !hasMax {
		max = int(high.Int64)
	}
	return domain.LevelRange{Min: min, Max: max}, nil
}

// ReadTiles reads the requested tiles at level. MBTiles counts tile rows from
// the south (TMS): the conversion to and from the XYZ scheme of the domain is
// done here and nowhere else. A tile the file does not contain is reported in
// Missing.
func (r MBTiles) ReadTiles(path string, level int, ids []domain.TileID) (domain.TileRead, error) {
	db, err := open(path)
	if err != nil {
		return domain.TileRead{}, err
	}
	defer db.Close()

	format, err := metadataText(db, "format")
	if err != nil {
		return domain.TileRead{}, unreadable(path, err)
	}
	if format == "" {
		format = defaultFormat
	}

	query, err := db.Prepare(`SELECT tile_data FROM tiles WHERE zoom_level = ? AND tile_column = ? AND tile_row = ?`)
	if err != nil {
		return domain.TileRead{}, unreadable(path, err)
	}
	defer query.Close()

	read := domain.TileRead{Format: format}
	for _, id := range ids {
		tmsRow := 1<<level - 1 - id.Y

		var data []byte
		err := query.QueryRow(level, id.X, tmsRow).Scan(&data)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			read.Missing = append(read.Missing, id)
		case err != nil:
			return domain.TileRead{}, unreadable(path, err)
		default:
			read.Tiles = append(read.Tiles, domain.Tile{ID: id, Data: data})
		}
	}

	return read, nil
}

// open opens the file read-only and checks it is a database, so no journal or
// other file is ever created next to the user's map.
func open(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, unreadable(path, err)
	}

	// A relative path in a "file:" URI is read by SQLite as a URI authority
	// (file://resources/... reads "resources" as a host, which SQLite only
	// accepts as empty or "localhost"), so it must be made absolute first.
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, unreadable(path, err)
	}

	dsn := (&url.URL{Scheme: "file", Path: absolute, RawQuery: "mode=ro&immutable=1"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, unreadable(path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, unreadable(path, err)
	}
	return db, nil
}

func metadataText(db *sql.DB, name string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM metadata WHERE name = ?`, name).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func metadataInt(db *sql.DB, name string) (value int, found bool, err error) {
	text, err := metadataText(db, name)
	if err != nil || text == "" {
		return 0, false, err
	}
	if _, err := fmt.Sscanf(text, "%d", &value); err != nil {
		return 0, false, nil
	}
	return value, true, nil
}

func unreadable(path string, cause error) error {
	return fmt.Errorf("%w: %s: %w", domain.ErrGeoDataContentUnreadable, path, cause)
}
