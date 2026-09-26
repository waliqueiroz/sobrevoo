package geodatainspector

import (
	"database/sql"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// readMBTilesBounds opens path as a SQLite database and reads the MBTiles
// 1.3 "bounds" entry from its "metadata" table
// ("minLon,minLat,maxLon,maxLat" — research.md items 1-2), producing the
// equivalent domain.BoundingBox.
func readMBTilesBounds(path string) (domain.BoundingBox, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}
	defer db.Close()

	var value string
	if err := db.QueryRow(`SELECT value FROM metadata WHERE name = 'bounds'`).Scan(&value); err != nil {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}

	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}

	bounds := make([]float64, 4)
	for i, part := range parts {
		n, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
		}
		bounds[i] = n
	}
	minLon, minLat, maxLon, maxLat := bounds[0], bounds[1], bounds[2], bounds[3]

	declared := domain.BoundingBox{
		MinLatitude:  minLat,
		MaxLatitude:  maxLat,
		MinLongitude: minLon,
		MaxLongitude: maxLon,
		// The MBTiles spec does not define antimeridian-crossing bounds
		// explicitly; a minLon greater than maxLon is the only way such a
		// bounds string could express it, so it is treated the same way
		// domain.ComputeBoundingBox reports it for a route (FR-018).
		CrossesAntimeridian: minLon > maxLon,
	}

	// The declared bounds are not always right: some writers leave a corner
	// at 0,0. The tiles say where the map really has data, so the area is the
	// part of the declared bounds that has tiles.
	if extent, ok := tilesExtent(db); ok {
		return declared.ClippedTo(extent), nil
	}
	return declared, nil
}

// tilesExtent is the area covered by the tiles of the most detailed level of
// the file, if the file has a readable tiles table with tiles in it. MBTiles
// counts rows from the south, so they are turned around to the XYZ scheme the
// domain uses.
func tilesExtent(db *sql.DB) (domain.BoundingBox, bool) {
	var level sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(zoom_level) FROM tiles`).Scan(&level); err != nil || !level.Valid {
		return domain.BoundingBox{}, false
	}

	var minColumn, maxColumn, minRow, maxRow sql.NullInt64
	err := db.QueryRow(`SELECT MIN(tile_column), MAX(tile_column), MIN(tile_row), MAX(tile_row) FROM tiles WHERE zoom_level = ?`, level.Int64).
		Scan(&minColumn, &maxColumn, &minRow, &maxRow)
	if err != nil || !minColumn.Valid || !maxColumn.Valid || !minRow.Valid || !maxRow.Valid {
		return domain.BoundingBox{}, false
	}

	top := int(1)<<level.Int64 - 1
	return domain.TileRange{
		Level: int(level.Int64),
		MinX:  int(minColumn.Int64), MaxX: int(maxColumn.Int64),
		MinY: top - int(maxRow.Int64), MaxY: top - int(minRow.Int64),
	}.Bounds(), true
}
