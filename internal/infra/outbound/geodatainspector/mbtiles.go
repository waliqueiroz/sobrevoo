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

	return domain.BoundingBox{
		MinLatitude:  minLat,
		MaxLatitude:  maxLat,
		MinLongitude: minLon,
		MaxLongitude: maxLon,
		// The MBTiles spec does not define antimeridian-crossing bounds
		// explicitly; a minLon greater than maxLon is the only way such a
		// bounds string could express it, so it is treated the same way
		// domain.ComputeBoundingBox reports it for a route (FR-018).
		CrossesAntimeridian: minLon > maxLon,
	}, nil
}
