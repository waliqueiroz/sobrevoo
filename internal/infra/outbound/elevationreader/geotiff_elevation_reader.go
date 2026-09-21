// Package elevationreader implements domain.ElevationReader: reading the
// content of a registered elevation file.
package elevationreader

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/tiff/lzw"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// TIFF tags and values this reader uses. It supports the subset of TIFF that
// elevation models come in (specs/004-geo-data-slice/research.md, item 8):
// one band, chunky, in strips or tiles, uncompressed, Deflate or LZW, with no
// predictor or the horizontal or floating-point one, integers of 8, 16 or 32
// bits or floats of 32 or 64.
const (
	tagImageWidth      = 256
	tagImageLength     = 257
	tagBitsPerSample   = 258
	tagCompression     = 259
	tagStripOffsets    = 273
	tagSamplesPerPixel = 277
	tagRowsPerStrip    = 278
	tagStripByteCounts = 279
	tagPlanarConfig    = 284
	tagPredictor       = 317
	tagTileWidth       = 322
	tagTileLength      = 323
	tagTileOffsets     = 324
	tagTileByteCounts  = 325
	tagSampleFormat    = 339
	tagPixelScale      = 33550
	tagTiepoint        = 33922
	tagGeoKeyDirectory = 34735
	tagGDALNoData      = 42113

	typeASCII  = 2
	typeShort  = 3
	typeLong   = 4
	typeDouble = 12

	compressionNone         = 1
	compressionLZW          = 5
	compressionDeflate      = 8
	compressionAdobeDeflate = 32946

	formatUnsigned = 1
	formatSigned   = 2
	formatFloat    = 3

	geoKeyModelType     = 1024
	geoKeyRasterType    = 1025
	geoKeyVerticalUnits = 4099

	modelTypeGeographic = 2
	rasterPixelIsPoint  = 2

	unitMeter  = 9001
	unitFoot   = 9002
	unitUSFoot = 9003

	feetToMeters   = 0.3048
	usFeetToMeters = 1200.0 / 3937.0
)

// GeoTIFF reads GeoTIFF files, the format the second stage registers as
// elevation.
type GeoTIFF struct{}

// NewGeoTIFF creates a GeoTIFF reader.
func NewGeoTIFF() GeoTIFF {
	return GeoTIFF{}
}

type entry struct {
	typ   uint16
	count uint32
	raw   [4]byte
}

// raster is what the tags of a file say about it.
type raster struct {
	file  *os.File
	order binary.ByteOrder

	width, height int
	bits          int
	format        int
	compression   int
	predictor     int

	tiled                   bool
	blockWidth, blockHeight int
	offsets, counts         []uint32

	noData *float64

	info domain.ElevationGridInfo
}

// Describe reads the geometry of the grid from the tags, never the samples.
func (r GeoTIFF) Describe(path string) (domain.ElevationGridInfo, error) {
	raster, err := open(path, false)
	if err != nil {
		return domain.ElevationGridInfo{}, err
	}
	defer raster.file.Close()

	return raster.info, nil
}

// ReadWindow reads the samples of a window of the grid, in meters, decoding
// only the strips or tiles that the window touches. A sample the file marks
// as having no data (its no-data value, or NaN) is a NaN in the result.
func (r GeoTIFF) ReadWindow(path string, window domain.GridWindow) (domain.ElevationWindow, error) {
	raster, err := open(path, true)
	if err != nil {
		return domain.ElevationWindow{}, err
	}
	defer raster.file.Close()

	if window.Rows == 0 || window.Cols == 0 {
		return domain.ElevationWindow{}, nil
	}
	if window.Rows < 0 || window.Cols < 0 || window.FirstRow < 0 || window.FirstCol < 0 ||
		window.FirstRow+window.Rows > raster.height || window.FirstCol+window.Cols > raster.width {
		return domain.ElevationWindow{}, unreadable(path, fmt.Errorf("window %+v is not inside the %d × %d grid", window, raster.height, raster.width))
	}

	values, err := raster.read(window)
	if err != nil {
		return domain.ElevationWindow{}, unreadable(path, err)
	}
	return domain.ElevationWindow{Values: values}, nil
}

// open reads the tags of the file. With blocks, it also reads the tables of
// where the strips or tiles are.
func open(path string, blocks bool) (*raster, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, unreadable(path, err)
	}

	r, err := parse(file, blocks)
	if err != nil {
		file.Close()
		var unit *unitError
		if errors.As(err, &unit) {
			return nil, fmt.Errorf("%w: %s: %s", domain.ErrElevationUnitUnsupported, path, unit.Error())
		}
		return nil, unreadable(path, err)
	}
	return r, nil
}

type unitError struct{ code uint16 }

func (e *unitError) Error() string {
	return fmt.Sprintf("vertical unit %d is neither meters nor feet", e.code)
}

func unsupported(what string, args ...any) error {
	return fmt.Errorf("unsupported encoding: %s", fmt.Sprintf(what, args...))
}

func unreadable(path string, cause error) error {
	return fmt.Errorf("%w: %s: %w", domain.ErrGeoDataContentUnreadable, path, cause)
}

func parse(file *os.File, blocks bool) (*raster, error) {
	header := make([]byte, 8)
	if _, err := file.ReadAt(header, 0); err != nil {
		return nil, errors.New("not a TIFF file")
	}

	r := &raster{file: file}
	switch string(header[:2]) {
	case "II":
		r.order = binary.LittleEndian
	case "MM":
		r.order = binary.BigEndian
	default:
		return nil, errors.New("not a TIFF file")
	}
	if r.order.Uint16(header[2:4]) != 42 {
		return nil, errors.New("not a classic TIFF file")
	}

	entries, err := readEntries(file, r.order, int64(r.order.Uint32(header[4:8])))
	if err != nil {
		return nil, err
	}

	number := func(tag uint16, fallback uint32) (uint32, error) {
		e, ok := entries[tag]
		if !ok {
			return fallback, nil
		}
		values, err := r.readIntegers(e)
		if err != nil || len(values) == 0 {
			return 0, fmt.Errorf("tag %d cannot be read", tag)
		}
		return values[0], nil
	}

	width, err := number(tagImageWidth, 0)
	if err != nil {
		return nil, err
	}
	height, err := number(tagImageLength, 0)
	if err != nil {
		return nil, err
	}
	if width == 0 || height == 0 {
		return nil, errors.New("the image has no size")
	}
	r.width, r.height = int(width), int(height)

	samples, err := number(tagSamplesPerPixel, 1)
	if err != nil {
		return nil, err
	}
	if samples != 1 {
		return nil, unsupported("%d bands, only one is supported", samples)
	}
	planar, err := number(tagPlanarConfig, 1)
	if err != nil {
		return nil, err
	}
	if planar != 1 {
		return nil, unsupported("planar configuration %d", planar)
	}

	bits, err := number(tagBitsPerSample, 8)
	if err != nil {
		return nil, err
	}
	format, err := number(tagSampleFormat, formatUnsigned)
	if err != nil {
		return nil, err
	}
	r.bits, r.format = int(bits), int(format)
	switch {
	case (r.format == formatUnsigned || r.format == formatSigned) && (r.bits == 8 || r.bits == 16 || r.bits == 32):
	case r.format == formatFloat && (r.bits == 32 || r.bits == 64):
	default:
		return nil, unsupported("%d-bit samples of format %d", r.bits, r.format)
	}

	compression, err := number(tagCompression, compressionNone)
	if err != nil {
		return nil, err
	}
	switch compression {
	case compressionNone, compressionLZW, compressionDeflate, compressionAdobeDeflate:
	default:
		return nil, unsupported("compression %d", compression)
	}
	r.compression = int(compression)

	predictor, err := number(tagPredictor, 1)
	if err != nil {
		return nil, err
	}
	switch {
	case predictor == 1:
	case predictor == 2 && r.format != formatFloat:
	case predictor == 3 && r.format == formatFloat:
	default:
		return nil, unsupported("predictor %d for samples of format %d", predictor, r.format)
	}
	r.predictor = int(predictor)

	if err := r.readLayout(entries, blocks); err != nil {
		return nil, err
	}
	if err := r.readGeoreference(entries); err != nil {
		return nil, err
	}

	if e, ok := entries[tagGDALNoData]; ok {
		if text, err := r.readASCII(e); err == nil {
			if value, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
				r.noData = &value
			}
		}
	}

	return r, nil
}

func (r *raster) readLayout(entries map[uint16]entry, blocks bool) error {
	first := func(tag uint16) (int, bool, error) {
		e, ok := entries[tag]
		if !ok {
			return 0, false, nil
		}
		values, err := r.readIntegers(e)
		if err != nil || len(values) == 0 {
			return 0, false, fmt.Errorf("tag %d cannot be read", tag)
		}
		return int(values[0]), true, nil
	}

	tileWidth, tiled, err := first(tagTileWidth)
	if err != nil {
		return err
	}

	offsetsTag, countsTag := uint16(tagStripOffsets), uint16(tagStripByteCounts)
	if tiled {
		tileLength, ok, err := first(tagTileLength)
		if err != nil {
			return err
		}
		if !ok || tileWidth <= 0 || tileLength <= 0 {
			return errors.New("the tile size is missing")
		}
		r.tiled, r.blockWidth, r.blockHeight = true, tileWidth, tileLength
		offsetsTag, countsTag = tagTileOffsets, tagTileByteCounts
	} else {
		rows, ok, err := first(tagRowsPerStrip)
		if err != nil {
			return err
		}
		if !ok || rows <= 0 || rows > r.height {
			rows = r.height
		}
		r.blockWidth, r.blockHeight = r.width, rows
	}

	if !blocks {
		return nil
	}

	offsets, ok := entries[offsetsTag]
	if !ok {
		return errors.New("the strip or tile offsets are missing")
	}
	counts, ok := entries[countsTag]
	if !ok {
		return errors.New("the strip or tile byte counts are missing")
	}
	if r.offsets, err = r.readIntegers(offsets); err != nil {
		return err
	}
	if r.counts, err = r.readIntegers(counts); err != nil {
		return err
	}

	expected := r.blockCount()
	if len(r.offsets) != expected || len(r.counts) != expected {
		return fmt.Errorf("the file lists %d strips or tiles, the image needs %d", len(r.offsets), expected)
	}
	return nil
}

func (r *raster) blocksAcross() int { return (r.width + r.blockWidth - 1) / r.blockWidth }
func (r *raster) blocksDown() int   { return (r.height + r.blockHeight - 1) / r.blockHeight }
func (r *raster) blockCount() int   { return r.blocksAcross() * r.blocksDown() }

func (r *raster) readGeoreference(entries map[uint16]entry) error {
	scaleEntry, hasScale := entries[tagPixelScale]
	tiepointEntry, hasTiepoint := entries[tagTiepoint]
	keysEntry, hasKeys := entries[tagGeoKeyDirectory]
	if !hasScale || !hasTiepoint || !hasKeys {
		return errors.New("the georeferencing tags are missing")
	}

	scale, err := r.readDoubles(scaleEntry, 2)
	if err != nil {
		return err
	}
	tiepoint, err := r.readDoubles(tiepointEntry, 6)
	if err != nil {
		return err
	}
	keys, err := r.readShorts(keysEntry)
	if err != nil || len(keys) < 4 {
		return errors.New("the GeoKey directory cannot be read")
	}

	modelType, rasterType, unitCode := -1, 1, -1
	for i := 0; i < int(keys[3]) && 4+4*i+3 < len(keys); i++ {
		key := keys[4+4*i : 4+4*i+4]
		if key[1] != 0 { // stored in another tag, not inline
			continue
		}
		switch key[0] {
		case geoKeyModelType:
			modelType = int(key[3])
		case geoKeyRasterType:
			rasterType = int(key[3])
		case geoKeyVerticalUnits:
			unitCode = int(key[3])
		}
	}
	if modelType != modelTypeGeographic {
		return errors.New("the raster is not in a geographic coordinate system")
	}

	unit := 1.0
	switch unitCode {
	case -1, unitMeter:
	case unitFoot:
		unit = feetToMeters
	case unitUSFoot:
		unit = usFeetToMeters
	default:
		return &unitError{code: uint16(unitCode)}
	}

	cellLon, cellLat := scale[0], scale[1]
	if cellLon <= 0 || cellLat <= 0 {
		return errors.New("the pixel scale is not positive")
	}

	// the tie point maps raster point (i, j) to (x, y); the north-west corner
	// of cell (0, 0) is where raster point (0, 0) is, unless the raster
	// points are cell centers
	west := tiepoint[3] - tiepoint[0]*cellLon
	north := tiepoint[4] + tiepoint[1]*cellLat
	if rasterType == rasterPixelIsPoint {
		west -= cellLon / 2
		north += cellLat / 2
	}

	r.info = domain.ElevationGridInfo{
		Rows: r.height, Cols: r.width,
		NorthLatitude: north, WestLongitude: wrapLongitude(west),
		CellLatitude: cellLat, CellLongitude: cellLon,
		UnitToMeters: unit,
	}
	return nil
}

func wrapLongitude(lon float64) float64 {
	return math.Mod(math.Mod(lon+180, 360)+360, 360) - 180
}

// read decodes the samples of window, in meters.
func (r *raster) read(window domain.GridWindow) ([]float32, error) {
	size := r.bits / 8
	values := make([]float32, 0, window.Rows*window.Cols)
	decoded := map[int][]byte{}

	blockAt := func(row, col int) (block []byte, blockRow, blockCol int, err error) {
		index := (row/r.blockHeight)*r.blocksAcross() + col/r.blockWidth
		if data, ok := decoded[index]; ok {
			return data, row % r.blockHeight, col % r.blockWidth, nil
		}
		data, err := r.decodeBlock(index)
		if err != nil {
			return nil, 0, 0, err
		}
		decoded[index] = data
		return data, row % r.blockHeight, col % r.blockWidth, nil
	}

	for row := window.FirstRow; row < window.FirstRow+window.Rows; row++ {
		for col := window.FirstCol; col < window.FirstCol+window.Cols; col++ {
			block, blockRow, blockCol, err := blockAt(row, col)
			if err != nil {
				return nil, err
			}
			position := (blockRow*r.blockWidth + blockCol) * size
			values = append(values, r.sample(block[position:position+size]))
		}
	}
	return values, nil
}

// sample converts one raw sample to meters, or NaN when it has no value.
func (r *raster) sample(raw []byte) float32 {
	var value float64
	switch {
	case r.format == formatFloat && r.bits == 32:
		single := math.Float32frombits(r.order.Uint32(raw))
		if single != single || (r.noData != nil && single == float32(*r.noData)) {
			return float32(math.NaN())
		}
		value = float64(single)
	case r.format == formatFloat:
		value = math.Float64frombits(r.order.Uint64(raw))
	case r.format == formatUnsigned && r.bits == 8:
		value = float64(raw[0])
	case r.format == formatSigned && r.bits == 8:
		value = float64(int8(raw[0]))
	case r.format == formatUnsigned && r.bits == 16:
		value = float64(r.order.Uint16(raw))
	case r.format == formatSigned && r.bits == 16:
		value = float64(int16(r.order.Uint16(raw)))
	case r.format == formatUnsigned:
		value = float64(r.order.Uint32(raw))
	default:
		value = float64(int32(r.order.Uint32(raw)))
	}

	if math.IsNaN(value) || (r.noData != nil && value == *r.noData) {
		return float32(math.NaN())
	}
	return float32(value * r.info.UnitToMeters)
}

// decodeBlock reads, decompresses and un-predicts one strip or tile, giving
// its samples as raw bytes, row by row.
func (r *raster) decodeBlock(index int) ([]byte, error) {
	size := r.bits / 8

	rows := r.blockHeight
	if !r.tiled {
		if remaining := r.height - index*r.blockHeight; remaining < rows {
			rows = remaining
		}
	}
	expected := rows * r.blockWidth * size

	raw := make([]byte, r.counts[index])
	if _, err := r.file.ReadAt(raw, int64(r.offsets[index])); err != nil {
		return nil, fmt.Errorf("strip or tile %d is cut off: %w", index, err)
	}

	var data []byte
	switch r.compression {
	case compressionNone:
		data = raw
	case compressionDeflate, compressionAdobeDeflate:
		reader, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("strip or tile %d: %w", index, err)
		}
		defer reader.Close()
		if data, err = io.ReadAll(reader); err != nil {
			return nil, fmt.Errorf("strip or tile %d: %w", index, err)
		}
	case compressionLZW:
		reader := lzw.NewReader(bytes.NewReader(raw), lzw.MSB, 8)
		defer reader.Close()
		var err error
		if data, err = io.ReadAll(reader); err != nil {
			return nil, fmt.Errorf("strip or tile %d: %w", index, err)
		}
	}

	if len(data) < expected {
		return nil, fmt.Errorf("strip or tile %d has %d bytes, %d were expected", index, len(data), expected)
	}
	data = data[:expected]

	switch r.predictor {
	case 2:
		r.undoHorizontalPredictor(data, rows)
	case 3:
		r.undoFloatingPointPredictor(data, rows)
	}
	return data, nil
}

// undoHorizontalPredictor turns each row of differences back into samples.
func (r *raster) undoHorizontalPredictor(data []byte, rows int) {
	size := r.bits / 8
	rowBytes := r.blockWidth * size

	for row := 0; row < rows; row++ {
		line := data[row*rowBytes : (row+1)*rowBytes]
		switch size {
		case 1:
			for i := 1; i < r.blockWidth; i++ {
				line[i] += line[i-1]
			}
		case 2:
			for i := 1; i < r.blockWidth; i++ {
				r.order.PutUint16(line[2*i:], r.order.Uint16(line[2*i:])+r.order.Uint16(line[2*(i-1):]))
			}
		case 4:
			for i := 1; i < r.blockWidth; i++ {
				r.order.PutUint32(line[4*i:], r.order.Uint32(line[4*i:])+r.order.Uint32(line[4*(i-1):]))
			}
		}
	}
}

// undoFloatingPointPredictor undoes TIFF's floating-point predictor: each row
// holds the bytes of its samples grouped by significance, most significant
// first, and differenced.
func (r *raster) undoFloatingPointPredictor(data []byte, rows int) {
	size := r.bits / 8
	rowBytes := r.blockWidth * size
	planes := make([]byte, rowBytes)

	for row := 0; row < rows; row++ {
		line := data[row*rowBytes : (row+1)*rowBytes]
		copy(planes, line)
		for i := 1; i < rowBytes; i++ {
			planes[i] += planes[i-1]
		}
		for sample := 0; sample < r.blockWidth; sample++ {
			for b := 0; b < size; b++ {
				destination := b
				if r.order == binary.LittleEndian {
					destination = size - 1 - b
				}
				line[size*sample+destination] = planes[b*r.blockWidth+sample]
			}
		}
	}
}

func readEntries(file *os.File, order binary.ByteOrder, offset int64) (map[uint16]entry, error) {
	countBytes := make([]byte, 2)
	if _, err := file.ReadAt(countBytes, offset); err != nil {
		return nil, errors.New("the directory of the TIFF file cannot be read")
	}
	count := int(order.Uint16(countBytes))

	raw := make([]byte, count*12)
	if _, err := file.ReadAt(raw, offset+2); err != nil {
		return nil, errors.New("the directory of the TIFF file is cut off")
	}

	entries := make(map[uint16]entry, count)
	for i := 0; i < count; i++ {
		item := raw[i*12 : i*12+12]
		e := entry{typ: order.Uint16(item[2:4]), count: order.Uint32(item[4:8])}
		copy(e.raw[:], item[8:12])
		entries[order.Uint16(item[0:2])] = e
	}
	return entries, nil
}

func typeSize(typ uint16) int {
	switch typ {
	case typeASCII:
		return 1
	case typeShort:
		return 2
	case typeLong:
		return 4
	case typeDouble:
		return 8
	default:
		return 0
	}
}

// value returns the bytes of an entry's values: inline when they fit in the
// four bytes of the entry, else read from where the entry points.
func (r *raster) value(e entry) ([]byte, error) {
	size := typeSize(e.typ) * int(e.count)
	if size == 0 {
		return nil, fmt.Errorf("a tag of type %d cannot be read", e.typ)
	}
	if size <= 4 {
		return e.raw[:size], nil
	}

	data := make([]byte, size)
	if _, err := r.file.ReadAt(data, int64(r.order.Uint32(e.raw[:]))); err != nil {
		return nil, errors.New("a tag value is cut off")
	}
	return data, nil
}

func (r *raster) readIntegers(e entry) ([]uint32, error) {
	if e.typ != typeShort && e.typ != typeLong {
		return nil, fmt.Errorf("a tag of type %d is not an integer", e.typ)
	}
	data, err := r.value(e)
	if err != nil {
		return nil, err
	}
	values := make([]uint32, e.count)
	for i := range values {
		if e.typ == typeShort {
			values[i] = uint32(r.order.Uint16(data[2*i:]))
		} else {
			values[i] = r.order.Uint32(data[4*i:])
		}
	}
	return values, nil
}

func (r *raster) readShorts(e entry) ([]uint16, error) {
	if e.typ != typeShort {
		return nil, errors.New("the tag is not made of shorts")
	}
	data, err := r.value(e)
	if err != nil {
		return nil, err
	}
	values := make([]uint16, e.count)
	for i := range values {
		values[i] = r.order.Uint16(data[2*i:])
	}
	return values, nil
}

func (r *raster) readDoubles(e entry, atLeast int) ([]float64, error) {
	if e.typ != typeDouble || int(e.count) < atLeast {
		return nil, errors.New("a georeferencing tag is not what it should be")
	}
	data, err := r.value(e)
	if err != nil {
		return nil, err
	}
	values := make([]float64, e.count)
	for i := range values {
		values[i] = math.Float64frombits(r.order.Uint64(data[8*i:]))
	}
	return values, nil
}

func (r *raster) readASCII(e entry) (string, error) {
	if e.typ != typeASCII {
		return "", errors.New("the tag is not text")
	}
	data, err := r.value(e)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\x00"), nil
}
