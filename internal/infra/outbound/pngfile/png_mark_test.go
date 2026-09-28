package pngfile

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

const (
	anID     = domain.FrameSetID("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	aPlanID  = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	textHead = 33 // where the first chunk after the header starts
)

// aMark is what a frame of this tool says about itself.
var aMark = domain.FrameMark{SetID: anID, PlanID: aPlanID}

// patternFrame is a frame of 6 × 4 pixels, each of a color of its own.
func patternFrame() domain.FrameImage {
	image := domain.NewFrameImage(domain.Resolution{Width: 6, Height: 4}, domain.RGB{})
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			image.Set(x, y, domain.RGB{R: uint8(10 * x), G: uint8(40 * y), B: uint8(7*x + 3*y)})
		}
	}
	return image
}

// chunks lists the types of the chunks of a PNG, in order.
func chunks(t *testing.T, data []byte) []string {
	t.Helper()
	var types []string
	for position := 8; position < len(data); {
		length := int(binary.BigEndian.Uint32(data[position:]))
		types = append(types, string(data[position+4:position+8]))
		position += 12 + length
	}
	return types
}

func Test_encodeFrame(t *testing.T) {
	t.Run("should be a PNG that decodes to the same pixels", func(t *testing.T) {
		// given
		frame := patternFrame()

		// when
		data, err := encodeFrame(frame, aMark)

		// then
		require.NoError(t, err)
		decoded, err := png.Decode(bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, image.Rect(0, 0, 6, 4), decoded.Bounds())
		for y := 0; y < 4; y++ {
			for x := 0; x < 6; x++ {
				r, g, b, a := decoded.At(x, y).RGBA()
				want := frame.At(x, y)
				assert.Equal(t, [4]uint32{uint32(want.R) * 257, uint32(want.G) * 257, uint32(want.B) * 257, 65535}, [4]uint32{r, g, b, a}, "pixel %d,%d", x, y)
			}
		}
	})

	t.Run("should have a header of 8 bits per channel, RGB with no alpha, not interlaced", func(t *testing.T) {
		// given / when
		data, err := encodeFrame(patternFrame(), aMark)

		// then
		require.NoError(t, err)
		assert.Equal(t, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}, data[:8])
		assert.Equal(t, "IHDR", string(data[12:16]))
		assert.Equal(t, uint32(6), binary.BigEndian.Uint32(data[16:]))
		assert.Equal(t, uint32(4), binary.BigEndian.Uint32(data[20:]))
		assert.Equal(t, byte(8), data[24], "bit depth")
		assert.Equal(t, byte(2), data[25], "color type: truecolor")
		assert.Equal(t, byte(0), data[28], "interlace")
	})

	t.Run("should have two text chunks right after the header, the set and then the plan, and no other auxiliary chunk", func(t *testing.T) {
		// given / when
		data, err := encodeFrame(patternFrame(), aMark)

		// then
		require.NoError(t, err)
		types := chunks(t, data)
		assert.Equal(t, "IHDR", types[0])
		assert.Equal(t, "tEXt", types[1])
		assert.Equal(t, "tEXt", types[2])
		assert.Equal(t, "IEND", types[len(types)-1])
		for _, kind := range types[3:] {
			assert.Contains(t, []string{"IDAT", "IEND"}, kind)
		}

		first := data[textHead:]
		firstLength := int(binary.BigEndian.Uint32(first))
		assert.Equal(t, "Sobrevoo\x00frame-set="+string(anID), string(first[8:8+firstLength]))
		assert.Equal(t, crc32.ChecksumIEEE(first[4:8+firstLength]), binary.BigEndian.Uint32(first[8+firstLength:]), "the checksum of the set chunk")

		second := first[12+firstLength:]
		secondLength := int(binary.BigEndian.Uint32(second))
		assert.Equal(t, "Sobrevoo\x00plan="+aPlanID, string(second[8:8+secondLength]))
		assert.Equal(t, crc32.ChecksumIEEE(second[4:8+secondLength]), binary.BigEndian.Uint32(second[8+secondLength:]), "the checksum of the plan chunk")
	})

	t.Run("should give the same bytes for the same frame and set", func(t *testing.T) {
		// given / when
		first, _ := encodeFrame(patternFrame(), aMark)
		second, _ := encodeFrame(patternFrame(), aMark)

		// then
		assert.Equal(t, first, second)
	})

	t.Run("should give other bytes for another plan", func(t *testing.T) {
		// given / when
		first, _ := encodeFrame(patternFrame(), aMark)
		second, _ := encodeFrame(patternFrame(), domain.FrameMark{SetID: anID, PlanID: "ff" + aPlanID[2:]})

		// then
		assert.NotEqual(t, first, second)
	})

	t.Run("should give other bytes for another set", func(t *testing.T) {
		// given / when
		first, _ := encodeFrame(patternFrame(), aMark)
		second, _ := encodeFrame(patternFrame(), domain.FrameMark{SetID: domain.FrameSetID("ff" + string(anID)[2:]), PlanID: aPlanID})

		// then
		assert.NotEqual(t, first, second)
	})
}

func Test_readMark(t *testing.T) {
	t.Run("should give the set and the plan a frame of this tool says it belongs to and came from", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)

		// when
		mark, ok := readMark(data)

		// then
		assert.True(t, ok)
		assert.Equal(t, aMark, mark)
	})

	t.Run("should find both in the head of the file alone", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)

		// when
		mark, ok := readMark(data[:min(len(data), headBytes)])

		// then
		assert.True(t, ok)
		assert.Equal(t, aMark, mark)
	})

	t.Run("should say a frame drawn before the plan was written is this tool's, with no plan", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)

		// when
		mark, ok := readMark(helper.WithoutPlanMark(data))

		// then
		assert.True(t, ok)
		assert.Equal(t, domain.FrameMark{SetID: anID}, mark)
	})

	t.Run("should not find one in a PNG of another tool", func(t *testing.T) {
		// given
		var out bytes.Buffer
		require.NoError(t, png.Encode(&out, image.NewNRGBA(image.Rect(0, 0, 3, 3))))

		// when
		_, ok := readMark(out.Bytes())

		// then
		assert.False(t, ok)
	})

	t.Run("should not find one in what is not a PNG, or in a header cut short", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)

		// when / then
		for _, bad := range [][]byte{nil, []byte("not a png at all"), data[:20], data[:40], make([]byte, 100)} {
			_, ok := readMark(bad)
			assert.False(t, ok)
		}
	})

	t.Run("should not trust a set chunk whose checksum is wrong", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)
		data[45] ^= 0xFF

		// when
		_, ok := readMark(data)

		// then
		assert.False(t, ok)
	})

	t.Run("should ignore a plan chunk whose checksum is wrong, keeping the set", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)
		setLength := int(binary.BigEndian.Uint32(data[textHead:]))
		planStart := textHead + 12 + setLength
		data[planStart+8+len("Sobrevoo\x00plan=")] ^= 0xFF

		// when
		mark, ok := readMark(data)

		// then
		assert.True(t, ok)
		assert.Equal(t, domain.FrameMark{SetID: anID}, mark)
	})

	t.Run("should ignore a plan chunk that comes after the image data", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)
		withoutPlan := helper.WithoutPlanMark(data)
		end := len(withoutPlan) - len(iendChunk)
		late := append(append(append([]byte(nil), withoutPlan[:end]...), textChunk("Sobrevoo\x00plan="+aPlanID)...), iendChunk...)

		// when
		mark, ok := readMark(late)

		// then
		assert.True(t, ok)
		assert.Equal(t, domain.FrameMark{SetID: anID}, mark)
	})

	t.Run("should not find one when only the plan chunk is there", func(t *testing.T) {
		// given
		data, err := encodeFrame(patternFrame(), aMark)
		require.NoError(t, err)
		setLength := int(binary.BigEndian.Uint32(data[textHead:]))
		onlyPlan := append(append([]byte(nil), data[:textHead]...), data[textHead+12+setLength:]...)

		// when
		_, ok := readMark(onlyPlan)

		// then
		assert.False(t, ok)
	})
}

func Test_readInfo(t *testing.T) {
	data, err := encodeFrame(patternFrame(), aMark)
	require.NoError(t, err)
	tail := data[len(data)-min(len(data), tailBytes):]

	t.Run("should give the size of the image and say it is whole when it ends as a PNG does", func(t *testing.T) {
		// given / when
		width, height, complete := readInfo(data[:min(len(data), headBytes)], tail)

		// then
		assert.Equal(t, 6, width)
		assert.Equal(t, 4, height)
		assert.True(t, complete)
	})

	t.Run("should say a file cut short is not whole", func(t *testing.T) {
		// given
		cut := data[:len(data)-5]

		// when
		width, height, complete := readInfo(cut[:min(len(cut), headBytes)], cut[len(cut)-min(len(cut), tailBytes):])

		// then
		assert.Equal(t, 6, width)
		assert.Equal(t, 4, height)
		assert.False(t, complete)
	})

	t.Run("should say what is not a PNG is not whole", func(t *testing.T) {
		// given / when
		_, _, complete := readInfo([]byte("nothing"), []byte("nothing"))

		// then
		assert.False(t, complete)
	})
}
