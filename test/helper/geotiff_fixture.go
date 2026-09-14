package helper

import (
	"bytes"
	"encoding/binary"
)

// GeoTIFF tag ids used by the fixtures below, mirroring the minimal set
// internal/infra/outbound/geodatainspector/geotiff.go reads (research.md
// items 3-4). Only the little-endian byte order and the
// ModelPixelScaleTag + ModelTiepointTag pair are produced here — the
// simplest, most common way a single-raster GeoTIFF expresses
// georeferencing.
const (
	tiffTagImageWidth      = 256
	tiffTagImageLength     = 257
	tiffTagModelPixelScale = 33550
	tiffTagModelTiepoint   = 33922
	tiffTagGeoKeyDirectory = 34735
	tiffTypeLong           = 4
	tiffTypeDouble         = 12
	tiffTypeShort          = 3
	geoKeyGTModelType      = 1024
	geoModelTypeGeographic = 2
	geoModelTypeProjected  = 1
)

// geoTIFFParams describes the raster and georeferencing values a fixture
// GeoTIFF is built from.
type geoTIFFParams struct {
	width, height        uint32
	scaleX, scaleY       float64
	originLon, originLat float64
	modelType            uint16
}

// ValidGeoTIFF returns the raw content of a minimal, real GeoTIFF file in a
// geographic CRS (WGS84), covering longitude [10.0, 11.0] and latitude
// [50.0, 51.0] — a 100x100 raster at 0.01 degrees/pixel, anchored at its
// top-left corner (10.0, 51.0).
func ValidGeoTIFF() []byte {
	return buildGeoTIFF(geoTIFFParams{
		width: 100, height: 100,
		scaleX: 0.01, scaleY: 0.01,
		originLon: 10.0, originLat: 51.0,
		modelType: geoModelTypeGeographic,
	})
}

// GeoTIFFCrossingAntimeridian returns a GeoTIFF whose raster starts at
// longitude 175 and is 10 degrees wide, so its computed extent crosses the
// 180th meridian (covering longitude [175, 180] and [-180, -175]).
func GeoTIFFCrossingAntimeridian() []byte {
	return buildGeoTIFF(geoTIFFParams{
		width: 100, height: 20,
		scaleX: 0.1, scaleY: 0.1,
		originLon: 175.0, originLat: 1.0,
		modelType: geoModelTypeGeographic,
	})
}

// GeoTIFFWithProjectedCRS returns the same raster as ValidGeoTIFF, but
// tagged with a projected (non-geographic) model type — recognized as TIFF
// content, but not a supported geo data format (research.md item 3).
func GeoTIFFWithProjectedCRS() []byte {
	return buildGeoTIFF(geoTIFFParams{
		width: 100, height: 100,
		scaleX: 0.01, scaleY: 0.01,
		originLon: 10.0, originLat: 51.0,
		modelType: geoModelTypeProjected,
	})
}

// NotTIFFContent returns bytes that are not a TIFF file at all — used to
// exercise GeoTIFF format detection failing before any tag is read.
func NotTIFFContent() []byte {
	return []byte("this is not a geographic data file")
}

// buildGeoTIFF hand-encodes a minimal little-endian TIFF file with exactly
// the five tags internal/infra/outbound/geodatainspector/geotiff.go reads:
// ImageWidth, ImageLength, ModelPixelScaleTag, ModelTiepointTag and
// GeoKeyDirectoryTag. It intentionally avoids any TIFF-writing library
// (research.md item 4) since only this narrow, fixed structure is needed.
func buildGeoTIFF(p geoTIFFParams) []byte {
	const (
		headerSize     = 8
		entryCount     = 5
		entrySize      = 12
		ifdSize        = 2 + entryCount*entrySize + 4
		ifdOffset      = headerSize
		pixelScaleOff  = ifdOffset + ifdSize
		pixelScaleSize = 3 * 8 // ScaleX, ScaleY, ScaleZ (float64)
		tiepointOff    = pixelScaleOff + pixelScaleSize
		tiepointSize   = 6 * 8 // I, J, K, X, Y, Z (float64)
		geoKeyOff      = tiepointOff + tiepointSize
		geoKeySize     = 8 * 2 // 8 SHORTs: header(4) + one key entry(4)
	)

	var buf bytes.Buffer

	// Header: "II" (little-endian), magic 42, offset to first IFD.
	buf.WriteString("II")
	binary.Write(&buf, binary.LittleEndian, uint16(42))
	binary.Write(&buf, binary.LittleEndian, uint32(ifdOffset))

	// IFD: entry count, then the 5 entries in ascending tag order (as real
	// TIFF writers do, though this reader does not require it).
	binary.Write(&buf, binary.LittleEndian, uint16(entryCount))

	writeEntry := func(tag, typ uint16, count uint32, valueOrOffset uint32) {
		binary.Write(&buf, binary.LittleEndian, tag)
		binary.Write(&buf, binary.LittleEndian, typ)
		binary.Write(&buf, binary.LittleEndian, count)
		binary.Write(&buf, binary.LittleEndian, valueOrOffset)
	}

	writeEntry(tiffTagImageWidth, tiffTypeLong, 1, p.width)
	writeEntry(tiffTagImageLength, tiffTypeLong, 1, p.height)
	writeEntry(tiffTagModelPixelScale, tiffTypeDouble, 3, uint32(pixelScaleOff))
	writeEntry(tiffTagModelTiepoint, tiffTypeDouble, 6, uint32(tiepointOff))
	writeEntry(tiffTagGeoKeyDirectory, tiffTypeShort, 8, uint32(geoKeyOff))

	binary.Write(&buf, binary.LittleEndian, uint32(0)) // no next IFD

	// External data area.
	binary.Write(&buf, binary.LittleEndian, p.scaleX)
	binary.Write(&buf, binary.LittleEndian, p.scaleY)
	binary.Write(&buf, binary.LittleEndian, 0.0) // ScaleZ

	binary.Write(&buf, binary.LittleEndian, 0.0) // I
	binary.Write(&buf, binary.LittleEndian, 0.0) // J
	binary.Write(&buf, binary.LittleEndian, 0.0) // K
	binary.Write(&buf, binary.LittleEndian, p.originLon)
	binary.Write(&buf, binary.LittleEndian, p.originLat)
	binary.Write(&buf, binary.LittleEndian, 0.0) // Z

	// GeoKeyDirectory: KeyDirectoryVersion, KeyRevision, MinorRevision,
	// NumberOfKeys, then one key entry (GTModelTypeGeoKey, in-line value).
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // KeyDirectoryVersion
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // KeyRevision
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // MinorRevision
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // NumberOfKeys
	binary.Write(&buf, binary.LittleEndian, uint16(geoKeyGTModelType))
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // TIFFTagLocation (0 = value inline)
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // Count
	binary.Write(&buf, binary.LittleEndian, p.modelType)

	return buf.Bytes()
}
