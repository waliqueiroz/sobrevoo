package geodatainspector

import (
	"encoding/binary"
	"io"
	"math"
	"os"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// The minimal set of TIFF/GeoTIFF tags this reader understands (research.md
// items 3-4). Only ModelPixelScaleTag + ModelTiepointTag georeferencing is
// supported — the simplest and most common way a single-raster GeoTIFF
// expresses its geographic extent; ModelTransformationTag is not handled.
const (
	tiffTagImageWidth      = 256
	tiffTagImageLength     = 257
	tiffTagModelPixelScale = 33550
	tiffTagModelTiepoint   = 33922
	tiffTagGeoKeyDirectory = 34735

	tiffTypeShort  = 3
	tiffTypeLong   = 4
	tiffTypeDouble = 12

	// geoKeyGTModelType is the GeoKey id (in the GeoKeyDirectoryTag) that
	// identifies the coordinate reference system's model type.
	geoKeyGTModelType = 1024
	// geoModelTypeGeographic is the only GTModelTypeGeoKey value this
	// reader accepts — a geographic (lat/lon) CRS such as WGS84
	// (research.md item 3).
	geoModelTypeGeographic = 2
)

// tiffIFDEntry is one 12-byte Image File Directory entry.
type tiffIFDEntry struct {
	Type          uint16
	Count         uint32
	ValueOrOffset uint32
}

// readGeoTIFFBoundingBox opens path as a TIFF file and computes the
// geographic area its raster covers, from ImageWidth/ImageLength and the
// ModelPixelScaleTag/ModelTiepointTag georeferencing tags. It returns
// domain.ErrUnsupportedDataFormat when the CRS is not geographic, or when
// any required tag is missing or malformed.
func readGeoTIFFBoundingBox(path string) (domain.BoundingBox, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.BoundingBox{}, domain.ErrDataFileNotFound
		}
		return domain.BoundingBox{}, domain.ErrDataFileUnreadable
	}
	defer f.Close()

	header := make([]byte, 8)
	if _, err := io.ReadFull(f, header); err != nil {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}

	var order binary.ByteOrder
	switch string(header[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}

	entries, err := readIFD(f, order, order.Uint32(header[4:8]))
	if err != nil {
		return domain.BoundingBox{}, err
	}

	width, ok := entries[tiffTagImageWidth]
	if !ok || width.Type != tiffTypeLong {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}
	height, ok := entries[tiffTagImageLength]
	if !ok || height.Type != tiffTypeLong {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}
	pixelScale, ok := entries[tiffTagModelPixelScale]
	if !ok {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}
	tiepoint, ok := entries[tiffTagModelTiepoint]
	if !ok {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}
	geoKeys, ok := entries[tiffTagGeoKeyDirectory]
	if !ok {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}

	modelType, err := readGeoKeyModelType(f, order, geoKeys)
	if err != nil {
		return domain.BoundingBox{}, err
	}
	if modelType != geoModelTypeGeographic {
		return domain.BoundingBox{}, domain.ErrUnsupportedDataFormat
	}

	scaleX, scaleY, err := readPixelScale(f, order, pixelScale)
	if err != nil {
		return domain.BoundingBox{}, err
	}

	originLon, originLat, err := readTiepointOrigin(f, order, tiepoint)
	if err != nil {
		return domain.BoundingBox{}, err
	}

	minLon := originLon
	maxLon := originLon + scaleX*float64(width.ValueOrOffset)
	maxLat := originLat
	minLat := originLat - scaleY*float64(height.ValueOrOffset)

	// A raster placed near the antimeridian can compute a raw longitude
	// outside [-180, 180] (e.g. originLon=175 + 10 degrees wide = 185).
	// Wrap it back into range and flag the crossing, the same convention
	// domain.ComputeBoundingBox uses for a route (Princípio IV).
	var crossesAntimeridian bool
	switch {
	case maxLon > 180:
		maxLon -= 360
		crossesAntimeridian = true
	case minLon < -180:
		minLon += 360
		crossesAntimeridian = true
	}

	return domain.BoundingBox{
		MinLatitude:         minLat,
		MaxLatitude:         maxLat,
		MinLongitude:        minLon,
		MaxLongitude:        maxLon,
		CrossesAntimeridian: crossesAntimeridian,
	}, nil
}

// readIFD reads the Image File Directory at offset and returns its entries
// keyed by tag id.
func readIFD(f *os.File, order binary.ByteOrder, offset uint32) (map[uint16]tiffIFDEntry, error) {
	if _, err := f.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, domain.ErrUnsupportedDataFormat
	}

	var count uint16
	if err := binary.Read(f, order, &count); err != nil {
		return nil, domain.ErrUnsupportedDataFormat
	}

	entries := make(map[uint16]tiffIFDEntry, count)
	for i := 0; i < int(count); i++ {
		var raw [12]byte
		if _, err := io.ReadFull(f, raw[:]); err != nil {
			return nil, domain.ErrUnsupportedDataFormat
		}
		tag := order.Uint16(raw[0:2])
		entries[tag] = tiffIFDEntry{
			Type:          order.Uint16(raw[2:4]),
			Count:         order.Uint32(raw[4:8]),
			ValueOrOffset: order.Uint32(raw[8:12]),
		}
	}

	return entries, nil
}

// readPixelScale reads a ModelPixelScaleTag's (ScaleX, ScaleY, ScaleZ)
// triplet, returning ScaleX and ScaleY (degrees per pixel).
func readPixelScale(f *os.File, order binary.ByteOrder, entry tiffIFDEntry) (scaleX, scaleY float64, err error) {
	if entry.Type != tiffTypeDouble || entry.Count < 2 {
		return 0, 0, domain.ErrUnsupportedDataFormat
	}

	values, err := readDoubles(f, order, entry.ValueOrOffset, 2)
	if err != nil {
		return 0, 0, err
	}
	return values[0], values[1], nil
}

// readTiepointOrigin reads a ModelTiepointTag's (I, J, K, X, Y, Z) sextuple
// and returns (X, Y) — the geographic coordinate (longitude, latitude) the
// tiepoint's pixel anchors to. Only the single-tiepoint case (one raster
// corner mapped to one geographic point) is supported.
func readTiepointOrigin(f *os.File, order binary.ByteOrder, entry tiffIFDEntry) (lon, lat float64, err error) {
	if entry.Type != tiffTypeDouble || entry.Count < 6 {
		return 0, 0, domain.ErrUnsupportedDataFormat
	}

	values, err := readDoubles(f, order, entry.ValueOrOffset, 6)
	if err != nil {
		return 0, 0, err
	}
	return values[3], values[4], nil
}

// readGeoKeyModelType reads a GeoKeyDirectoryTag and returns the
// GTModelTypeGeoKey (id 1024) value stored inline in the directory.
func readGeoKeyModelType(f *os.File, order binary.ByteOrder, entry tiffIFDEntry) (uint16, error) {
	if entry.Type != tiffTypeShort || entry.Count < 4 {
		return 0, domain.ErrUnsupportedDataFormat
	}

	if _, err := f.Seek(int64(entry.ValueOrOffset), io.SeekStart); err != nil {
		return 0, domain.ErrUnsupportedDataFormat
	}

	raw := make([]byte, entry.Count*2)
	if _, err := io.ReadFull(f, raw); err != nil {
		return 0, domain.ErrUnsupportedDataFormat
	}

	numKeys := order.Uint16(raw[6:8])
	for i := 0; i < int(numKeys); i++ {
		base := 8 + i*8
		if base+8 > len(raw) {
			break
		}
		keyID := order.Uint16(raw[base : base+2])
		tagLocation := order.Uint16(raw[base+2 : base+4])
		value := order.Uint16(raw[base+6 : base+8])
		if keyID == geoKeyGTModelType && tagLocation == 0 {
			return value, nil
		}
	}

	return 0, domain.ErrUnsupportedDataFormat
}

// readDoubles seeks to offset and reads count little/big-endian float64
// values (per order).
func readDoubles(f *os.File, order binary.ByteOrder, offset uint32, count int) ([]float64, error) {
	if _, err := f.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, domain.ErrUnsupportedDataFormat
	}

	raw := make([]byte, count*8)
	if _, err := io.ReadFull(f, raw); err != nil {
		return nil, domain.ErrUnsupportedDataFormat
	}

	values := make([]float64, count)
	for i := range values {
		values[i] = math.Float64frombits(order.Uint64(raw[i*8 : i*8+8]))
	}
	return values, nil
}
