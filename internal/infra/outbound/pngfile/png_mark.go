// Package pngfile implements the outbound adapters that keep the frames of a
// flight as PNG files: a directory of frames and a single frame in a file.
//
// Every frame carries, inside, the identification of the set it belongs to and of
// the plan it was drawn from, so a directory needs no register of what it holds:
// what is a frame of this tool, of which set and of which plan, is read off the
// images themselves.
package pngfile

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

const (
	// headBytes is how much of the start of a frame file has its header and its
	// text chunk; tailBytes is how much of its end has the last chunk.
	headBytes = 4096
	tailBytes = 12

	// markKeyword and markPrefix say, in the text chunk, that the frame is of this
	// tool and of which set; planPrefix, in the text chunk that follows it, of
	// which plan the frame was drawn.
	markKeyword = "Sobrevoo"
	markPrefix  = markKeyword + "\x00frame-set="
	planPrefix  = markKeyword + "\x00plan="

	// afterHeader is where the first chunk after the header starts: the
	// signature, and the header chunk of 13 bytes of data.
	afterHeader = 8 + 4 + 4 + 13 + 4
)

var (
	pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	iendChunk    = []byte{0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xAE, 0x42, 0x60, 0x82}
)

// encodeFrame writes a frame as a PNG: 8 bits per channel, RGB, not interlaced,
// with two text chunks right after the header: the first says the frame is of
// this tool and belongs to the set of mark, the second which plan it was drawn
// from. The same frame and mark are always the same bytes.
func encodeFrame(frame domain.FrameImage, mark domain.FrameMark) ([]byte, error) {
	width, height := frame.Resolution.Width, frame.Resolution.Height

	// The encoder writes RGB, with no alpha, when every pixel is opaque.
	pixels := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < width*height; i++ {
		pixels.Pix[4*i], pixels.Pix[4*i+1], pixels.Pix[4*i+2], pixels.Pix[4*i+3] = frame.Pix[3*i], frame.Pix[3*i+1], frame.Pix[3*i+2], 255
	}

	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := encoder.Encode(&encoded, pixels); err != nil {
		return nil, err
	}

	data := encoded.Bytes()
	out := make([]byte, 0, len(data)+64)
	out = append(out, data[:afterHeader]...)
	out = append(out, textChunk(markPrefix+string(mark.SetID))...)
	out = append(out, textChunk(planPrefix+mark.PlanID)...)
	out = append(out, data[afterHeader:]...)
	return out, nil
}

// textChunk is a PNG chunk of type tEXt.
func textChunk(text string) []byte {
	chunk := make([]byte, 0, 12+len(text))
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(len(text)))
	chunk = append(chunk, "tEXt"...)
	chunk = append(chunk, text...)
	return binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
}

// readMark reads, from the start of a file, what a frame of this tool says about
// itself: the set it belongs to and the plan it was drawn from. ok is false for a
// file that is not a PNG, that has no set chunk before its image data, or whose
// set chunk does not check out; a frame with no plan chunk (or one that does not
// check out) is this tool's, with an empty PlanID.
func readMark(head []byte) (mark domain.FrameMark, ok bool) {
	if len(head) < len(pngSignature) || !bytes.Equal(head[:len(pngSignature)], pngSignature) {
		return domain.FrameMark{}, false
	}

	for position := len(pngSignature); position+8 <= len(head); {
		length := int(binary.BigEndian.Uint32(head[position:]))
		kind := string(head[position+4 : position+8])
		end := position + 8 + length
		if kind == "IDAT" || kind == "IEND" || end+4 > len(head) || length < 0 {
			break
		}

		if kind == "tEXt" && binary.BigEndian.Uint32(head[end:]) == crc32.ChecksumIEEE(head[position+4:end]) {
			text := head[position+8 : end]
			switch {
			case bytes.HasPrefix(text, []byte(markPrefix)):
				mark.SetID, ok = domain.FrameSetID(text[len(markPrefix):]), true
			case bytes.HasPrefix(text, []byte(planPrefix)):
				mark.PlanID = string(text[len(planPrefix):])
			}
		}
		position = end + 4
	}

	if !ok {
		return domain.FrameMark{}, false
	}
	return mark, true
}

// readInfo reads the size of a PNG off the start of its file and says whether it
// is whole: it ends, as a PNG does, with its last chunk. head is the start of the
// file and tail its last bytes.
func readInfo(head, tail []byte) (width, height int, complete bool) {
	if len(head) < afterHeader || !bytes.Equal(head[:len(pngSignature)], pngSignature) || string(head[12:16]) != "IHDR" {
		return 0, 0, false
	}

	width = int(binary.BigEndian.Uint32(head[16:]))
	height = int(binary.BigEndian.Uint32(head[20:]))
	return width, height, bytes.Equal(tail, iendChunk)
}
