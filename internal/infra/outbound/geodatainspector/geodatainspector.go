// Package geodatainspector implements the domain.GeoDataInspector port for
// the geo data formats Sobrevoo recognizes (FR-002, FR-003): MBTiles for
// base maps and GeoTIFF for elevation. It identifies which of the two a
// file is by sniffing its content's signature, then delegates to
// mbtiles.go or geotiff.go — mirroring how trackparser identifies GPX
// content in the previous stage.
package geodatainspector

import (
	"io"
	"os"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// sqliteMagic is the fixed 16-byte header every SQLite database file (and
// therefore every MBTiles file) starts with.
const sqliteMagic = "SQLite format 3\x00"

// Inspector implements domain.GeoDataInspector.
type Inspector struct{}

// New creates an Inspector.
func New() Inspector {
	return Inspector{}
}

// Inspect examines the file at path, determines whether it is a base map
// (MBTiles) or an elevation (GeoTIFF) source, and computes the geographic
// area it covers (research.md item 8). Filesystem errors encountered while
// opening path are translated here into domain.ErrDataFileNotFound /
// domain.ErrDataFileUnreadable; a path pointing at a directory is treated
// as unreadable, matching the edge case documented in spec.md.
func (Inspector) Inspect(path string) (domain.InspectedGeoData, error) {
	header, err := readHeader(path)
	if err != nil {
		return domain.InspectedGeoData{}, err
	}

	switch {
	case isSQLiteHeader(header):
		boundingBox, err := readMBTilesBounds(path)
		if err != nil {
			return domain.InspectedGeoData{}, err
		}
		return domain.InspectedGeoData{
			Format:      domain.DataFormatMBTiles,
			Type:        domain.DataTypeBaseMap,
			BoundingBox: boundingBox,
		}, nil

	case isTIFFHeader(header):
		boundingBox, err := readGeoTIFFBoundingBox(path)
		if err != nil {
			return domain.InspectedGeoData{}, err
		}
		return domain.InspectedGeoData{
			Format:      domain.DataFormatGeoTIFF,
			Type:        domain.DataTypeElevation,
			BoundingBox: boundingBox,
		}, nil

	default:
		return domain.InspectedGeoData{}, domain.ErrUnsupportedDataFormat
	}
}

// readHeader opens path and reads up to the first 16 bytes, enough to
// identify either signature. It also rejects a path that does not exist,
// cannot be opened, or is a directory.
func readHeader(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrDataFileNotFound
		}
		return nil, domain.ErrDataFileUnreadable
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, domain.ErrDataFileUnreadable
	}
	if info.IsDir() {
		return nil, domain.ErrDataFileUnreadable
	}

	header := make([]byte, len(sqliteMagic))
	n, err := io.ReadFull(f, header)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, domain.ErrDataFileUnreadable
	}

	return header[:n], nil
}

// isSQLiteHeader reports whether header starts with the SQLite database
// file signature (MBTiles is a SQLite database — research.md item 1).
func isSQLiteHeader(header []byte) bool {
	return len(header) >= len(sqliteMagic) && string(header[:len(sqliteMagic)]) == sqliteMagic
}

// isTIFFHeader reports whether header starts with a valid TIFF byte-order
// marker and magic number, in either byte order.
func isTIFFHeader(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	switch string(header[0:2]) {
	case "II":
		return header[2] == 0x2A && header[3] == 0x00
	case "MM":
		return header[2] == 0x00 && header[3] == 0x2A
	default:
		return false
	}
}
