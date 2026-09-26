package helper

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"math"
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
	shortDimensions      bool
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

// GeoTIFFWithShortDimensions returns the same raster as ValidGeoTIFF, but
// with the image width and length stored as SHORT values, as Copernicus DEMs
// and other real-world writers do, instead of LONG.
func GeoTIFFWithShortDimensions() []byte {
	return buildGeoTIFF(geoTIFFParams{
		width: 100, height: 100,
		scaleX: 0.01, scaleY: 0.01,
		originLon: 10.0, originLat: 51.0,
		modelType:       geoModelTypeGeographic,
		shortDimensions: true,
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

	dimensionType := uint16(tiffTypeLong)
	if p.shortDimensions {
		dimensionType = tiffTypeShort
	}
	writeEntry(tiffTagImageWidth, dimensionType, 1, p.width)
	writeEntry(tiffTagImageLength, dimensionType, 1, p.height)
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

// TIFF compression codes, for GeoTIFFSpec.Compression.
const (
	TIFFNone         uint16 = 1
	TIFFLZW          uint16 = 5
	TIFFJPEG         uint16 = 7
	TIFFDeflate      uint16 = 8
	TIFFAdobeDeflate uint16 = 32946
)

// SampleType is the type of the samples of a GeoTIFFSpec.
type SampleType int

const (
	Uint8 SampleType = iota
	Int16
	Uint16
	Int32
	Float32
	Float64
)

// VerticalUnit codes (VerticalUnitsGeoKey), for GeoTIFFSpec.VerticalUnit.
const (
	UnitMeter    uint16 = 9001
	UnitFoot     uint16 = 9002
	UnitUSFoot   uint16 = 9003
	UnitOtherRaw uint16 = 9999
)

// GeoTIFFSpec describes a GeoTIFF fixture with samples. The raster is in a
// geographic CRS, anchored at its north-west corner (OriginLon, OriginLat).
type GeoTIFFSpec struct {
	Width, Height        int
	OriginLon, OriginLat float64
	ScaleX, ScaleY       float64

	SampleType SampleType

	// Values are the samples, row by row from north to south; a missing
	// row or value is zero.
	Values [][]float64

	// Compression is one of the TIFF* codes (TIFFNone when zero).
	Compression uint16

	// Predictor is 1 (none, when zero), 2 (horizontal differencing, for
	// integers) or 3 (floating point).
	Predictor int

	// The layout is strips of RowsPerStrip rows (all rows in one strip when
	// zero) unless TileWidth is set, which lays the image out in tiles of
	// TileWidth × TileLength pixels.
	RowsPerStrip          int
	TileWidth, TileLength int

	BigEndian bool

	// SamplesPerPixel is 1 when zero. Values greater than 1 only set the tag
	// (the data is not laid out for several bands).
	SamplesPerPixel int

	// NoData is the GDAL_NODATA tag text, when set.
	NoData *string

	// VerticalUnit is the VerticalUnitsGeoKey, when set.
	VerticalUnit *uint16

	PixelIsPoint bool
	ProjectedCRS bool
}

// Truncated returns the first keep bytes of data: the file as if its end had
// been cut off.
func Truncated(data []byte, keep int) []byte {
	if keep > len(data) {
		keep = len(data)
	}
	return append([]byte{}, data[:keep]...)
}

const (
	tiffTagBitsPerSample    = 258
	tiffTagCompression      = 259
	tiffTagPhotometric      = 262
	tiffTagStripOffsets     = 273
	tiffTagSamplesPerPixel  = 277
	tiffTagRowsPerStrip     = 278
	tiffTagStripByteCounts  = 279
	tiffTagPlanarConfig     = 284
	tiffTagPredictor        = 317
	tiffTagTileWidth        = 322
	tiffTagTileLength       = 323
	tiffTagTileOffsets      = 324
	tiffTagTileByteCounts   = 325
	tiffTagSampleFormat     = 339
	tiffTagGDALNoData       = 42113
	tiffTypeASCII           = 2
	geoKeyGTRasterType      = 1025
	geoKeyVerticalUnits     = 4099
	geoRasterPixelIsArea    = 1
	geoRasterPixelIsPoint   = 2
	lzwClearCode            = 256
	lzwEndOfInformationCode = 257
)

type tiffEntry struct {
	tag   uint16
	typ   uint16
	count uint32
	data  []byte
}

// GeoTIFFWithSamples returns the raw content of a GeoTIFF file with the
// samples, compression and layout of spec. It is hand-encoded, like the
// other fixtures, with no TIFF library.
func GeoTIFFWithSamples(spec GeoTIFFSpec) []byte {
	var order binary.ByteOrder = binary.LittleEndian
	orderTag := "II"
	if spec.BigEndian {
		order = binary.BigEndian
		orderTag = "MM"
	}

	compression := spec.Compression
	if compression == 0 {
		compression = TIFFNone
	}
	predictor := spec.Predictor
	if predictor == 0 {
		predictor = 1
	}

	blocks := buildTIFFBlocks(spec, order, predictor)
	for i, block := range blocks {
		blocks[i] = compressTIFFBlock(block, compression)
	}

	bits, format := sampleTypeInfo(spec.SampleType)
	samplesPerPixel := spec.SamplesPerPixel
	if samplesPerPixel == 0 {
		samplesPerPixel = 1
	}

	tiled := spec.TileWidth > 0
	rowsPerStrip := spec.RowsPerStrip
	if rowsPerStrip == 0 {
		rowsPerStrip = spec.Height
	}

	modelType := geoModelTypeGeographic
	if spec.ProjectedCRS {
		modelType = geoModelTypeProjected
	}
	rasterType := geoRasterPixelIsArea
	if spec.PixelIsPoint {
		rasterType = geoRasterPixelIsPoint
	}

	geoKeys := [][4]uint16{
		{geoKeyGTModelType, 0, 1, uint16(modelType)},
		{geoKeyGTRasterType, 0, 1, uint16(rasterType)},
	}
	if spec.VerticalUnit != nil {
		geoKeys = append(geoKeys, [4]uint16{geoKeyVerticalUnits, 0, 1, *spec.VerticalUnit})
	}

	u16 := func(vs ...uint16) []byte {
		out := make([]byte, 2*len(vs))
		for i, v := range vs {
			order.PutUint16(out[2*i:], v)
		}
		return out
	}
	u32 := func(vs ...uint32) []byte {
		out := make([]byte, 4*len(vs))
		for i, v := range vs {
			order.PutUint32(out[4*i:], v)
		}
		return out
	}
	f64 := func(vs ...float64) []byte {
		out := make([]byte, 8*len(vs))
		for i, v := range vs {
			order.PutUint64(out[8*i:], math.Float64bits(v))
		}
		return out
	}

	geoKeyWords := []uint16{1, 1, 0, uint16(len(geoKeys))}
	for _, key := range geoKeys {
		geoKeyWords = append(geoKeyWords, key[:]...)
	}

	originLon, originLat := spec.OriginLon, spec.OriginLat

	// build lays out the IFD and the external data given where the block
	// data starts; it is called twice, because the offsets of the blocks
	// live in an external array whose size does not depend on their values.
	build := func(blocksStart uint32) (ifd, extra []byte) {
		offsets := make([]uint32, len(blocks))
		counts := make([]uint32, len(blocks))
		position := blocksStart
		for i, block := range blocks {
			offsets[i] = position
			counts[i] = uint32(len(block))
			position += uint32(len(block))
		}

		entries := []tiffEntry{
			{tiffTagImageWidth, tiffTypeLong, 1, u32(uint32(spec.Width))},
			{tiffTagImageLength, tiffTypeLong, 1, u32(uint32(spec.Height))},
			{tiffTagBitsPerSample, tiffTypeShort, 1, u16(uint16(bits))},
			{tiffTagCompression, tiffTypeShort, 1, u16(compression)},
			{tiffTagPhotometric, tiffTypeShort, 1, u16(1)},
		}
		if !tiled {
			entries = append(entries, tiffEntry{tiffTagStripOffsets, tiffTypeLong, uint32(len(blocks)), u32(offsets...)})
		}
		entries = append(entries, tiffEntry{tiffTagSamplesPerPixel, tiffTypeShort, 1, u16(uint16(samplesPerPixel))})
		if !tiled {
			entries = append(entries,
				tiffEntry{tiffTagRowsPerStrip, tiffTypeLong, 1, u32(uint32(rowsPerStrip))},
				tiffEntry{tiffTagStripByteCounts, tiffTypeLong, uint32(len(blocks)), u32(counts...)},
			)
		}
		entries = append(entries, tiffEntry{tiffTagPlanarConfig, tiffTypeShort, 1, u16(1)})
		if predictor != 1 {
			entries = append(entries, tiffEntry{tiffTagPredictor, tiffTypeShort, 1, u16(uint16(predictor))})
		}
		if tiled {
			entries = append(entries,
				tiffEntry{tiffTagTileWidth, tiffTypeLong, 1, u32(uint32(spec.TileWidth))},
				tiffEntry{tiffTagTileLength, tiffTypeLong, 1, u32(uint32(spec.TileLength))},
				tiffEntry{tiffTagTileOffsets, tiffTypeLong, uint32(len(blocks)), u32(offsets...)},
				tiffEntry{tiffTagTileByteCounts, tiffTypeLong, uint32(len(blocks)), u32(counts...)},
			)
		}
		entries = append(entries,
			tiffEntry{tiffTagSampleFormat, tiffTypeShort, 1, u16(uint16(format))},
			tiffEntry{tiffTagModelPixelScale, tiffTypeDouble, 3, f64(spec.ScaleX, spec.ScaleY, 0)},
			tiffEntry{tiffTagModelTiepoint, tiffTypeDouble, 6, f64(0, 0, 0, originLon, originLat, 0)},
			tiffEntry{tiffTagGeoKeyDirectory, tiffTypeShort, uint32(len(geoKeyWords)), u16(geoKeyWords...)},
		)
		if spec.NoData != nil {
			text := append([]byte(*spec.NoData), 0)
			entries = append(entries, tiffEntry{tiffTagGDALNoData, tiffTypeASCII, uint32(len(text)), text})
		}

		ifdSize := 2 + len(entries)*12 + 4
		externalStart := uint32(8 + ifdSize)

		var ifdBuf bytes.Buffer
		ifdBuf.Write(u16(uint16(len(entries))))
		for _, entry := range entries {
			ifdBuf.Write(u16(entry.tag, entry.typ))
			ifdBuf.Write(u32(entry.count))
			if len(entry.data) <= 4 {
				value := make([]byte, 4)
				copy(value, entry.data)
				ifdBuf.Write(value)
				continue
			}
			ifdBuf.Write(u32(externalStart + uint32(len(extra))))
			extra = append(extra, entry.data...)
			if len(extra)%2 == 1 {
				extra = append(extra, 0)
			}
		}
		ifdBuf.Write(u32(0))
		return ifdBuf.Bytes(), extra
	}

	ifd, extra := build(0)
	blocksStart := uint32(8 + len(ifd) + len(extra))
	ifd, extra = build(blocksStart)

	var file bytes.Buffer
	file.WriteString(orderTag)
	file.Write(u16(42))
	file.Write(u32(8))
	file.Write(ifd)
	file.Write(extra)
	for _, block := range blocks {
		file.Write(block)
	}
	return file.Bytes()
}

func sampleTypeInfo(t SampleType) (bits, format int) {
	switch t {
	case Uint8:
		return 8, 1
	case Int16:
		return 16, 2
	case Uint16:
		return 16, 1
	case Int32:
		return 32, 2
	case Float32:
		return 32, 3
	default:
		return 64, 3
	}
}

// sampleBits returns the raw bits of a sample value in its type.
func sampleBits(t SampleType, v float64) uint64 {
	switch t {
	case Float32:
		return uint64(math.Float32bits(float32(v)))
	case Float64:
		return math.Float64bits(v)
	default:
		return uint64(int64(v))
	}
}

// buildTIFFBlocks encodes the samples into uncompressed strips or tiles,
// applying the predictor to each row of each block.
func buildTIFFBlocks(spec GeoTIFFSpec, order binary.ByteOrder, predictor int) [][]byte {
	bits, _ := sampleTypeInfo(spec.SampleType)
	size := bits / 8

	value := func(row, col int) float64 {
		if row < len(spec.Values) && col < len(spec.Values[row]) {
			return spec.Values[row][col]
		}
		return 0
	}

	encodeRow := func(row []uint64) []byte {
		switch predictor {
		case 2:
			mask := uint64(1)<<bits - 1
			if bits == 64 {
				mask = ^uint64(0)
			}
			for i := len(row) - 1; i >= 1; i-- {
				row[i] = (row[i] - row[i-1]) & mask
			}
		}

		raw := make([]byte, len(row)*size)
		for i, bitsValue := range row {
			switch size {
			case 1:
				raw[i] = byte(bitsValue)
			case 2:
				order.PutUint16(raw[2*i:], uint16(bitsValue))
			case 4:
				order.PutUint32(raw[4*i:], uint32(bitsValue))
			case 8:
				order.PutUint64(raw[8*i:], bitsValue)
			}
		}

		if predictor != 3 {
			return raw
		}

		// Floating-point predictor: bytes of all samples regrouped by
		// significance (most significant first), then differenced.
		planes := make([]byte, len(raw))
		for i := range row {
			for b := 0; b < size; b++ {
				source := b
				if order == binary.LittleEndian {
					source = size - 1 - b
				}
				planes[b*len(row)+i] = raw[size*i+source]
			}
		}
		for i := len(planes) - 1; i >= 1; i-- {
			planes[i] -= planes[i-1]
		}
		return planes
	}

	block := func(firstRow, firstCol, rows, cols int) []byte {
		var out []byte
		for r := 0; r < rows; r++ {
			row := make([]uint64, cols)
			for c := 0; c < cols; c++ {
				if firstRow+r < spec.Height && firstCol+c < spec.Width {
					row[c] = sampleBits(spec.SampleType, value(firstRow+r, firstCol+c))
				}
			}
			out = append(out, encodeRow(row)...)
		}
		return out
	}

	var blocks [][]byte
	if spec.TileWidth > 0 {
		for top := 0; top < spec.Height; top += spec.TileLength {
			for left := 0; left < spec.Width; left += spec.TileWidth {
				blocks = append(blocks, block(top, left, spec.TileLength, spec.TileWidth))
			}
		}
		return blocks
	}

	rowsPerStrip := spec.RowsPerStrip
	if rowsPerStrip == 0 {
		rowsPerStrip = spec.Height
	}
	for top := 0; top < spec.Height; top += rowsPerStrip {
		rows := rowsPerStrip
		if top+rows > spec.Height {
			rows = spec.Height - top
		}
		blocks = append(blocks, block(top, 0, rows, spec.Width))
	}
	return blocks
}

func compressTIFFBlock(block []byte, compression uint16) []byte {
	switch compression {
	case TIFFDeflate, TIFFAdobeDeflate:
		var out bytes.Buffer
		writer := zlib.NewWriter(&out)
		writer.Write(block)
		writer.Close()
		return out.Bytes()
	case TIFFLZW:
		return lzwEncode(block)
	default:
		return block
	}
}

// lzwEncode is a minimal TIFF LZW encoder (most significant bit first, with
// the "early change" of TIFF), following libtiff's algorithm. The golang.org/x
// package only decodes.
func lzwEncode(data []byte) []byte {
	var out bytes.Buffer
	var bitBuffer uint64
	var bitCount uint
	width := uint(9)

	put := func(code int) {
		bitBuffer = bitBuffer<<width | uint64(code)
		bitCount += width
		for bitCount >= 8 {
			bitCount -= 8
			out.WriteByte(byte(bitBuffer >> bitCount))
		}
	}

	table := map[string]int{}
	next := 258
	reset := func() {
		table = map[string]int{}
		next = 258
		width = 9
	}

	put(lzwClearCode)
	if len(data) > 0 {
		current := string(data[0:1])
		codeOf := func(s string) int {
			if len(s) == 1 {
				return int(s[0])
			}
			return table[s]
		}
		for _, b := range data[1:] {
			candidate := current + string([]byte{b})
			if _, ok := table[candidate]; ok {
				current = candidate
				continue
			}
			put(codeOf(current))
			table[candidate] = next
			next++
			if next == 4094 {
				put(lzwClearCode)
				reset()
			} else if next > 511 && width == 9 || next > 1023 && width == 10 || next > 2047 && width == 11 {
				width++
			}
			current = string([]byte{b})
		}
		put(codeOf(current))
		next++
		if next == 4094 {
			put(lzwClearCode)
			reset()
		} else if next > 511 && width == 9 || next > 1023 && width == 10 || next > 2047 && width == 11 {
			width++
		}
	}
	put(lzwEndOfInformationCode)
	if bitCount > 0 {
		out.WriteByte(byte(bitBuffer << (8 - bitCount)))
	}
	return out.Bytes()
}
