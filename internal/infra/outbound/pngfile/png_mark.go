// Package pngfile implements the outbound adapters that keep the frames of a
// flight as PNG files: a directory of frames and a single frame in a file.
//
// Every frame carries, inside, the identification of the set it belongs to, so a
// directory needs no register of what it holds: what is a frame of this tool, and
// of which set, is read off the images themselves.
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
	// tool and of which set.
	markKeyword = "Sobrevoo"
	markPrefix  = markKeyword + "\x00frame-set="

	// afterHeader is where the first chunk after the header starts: the
	// signature, and the header chunk of 13 bytes of data.
	afterHeader = 8 + 4 + 4 + 13 + 4
)

var (
	pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	iendChunk    = []byte{0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xAE, 0x42, 0x60, 0x82}
)

// encodeFrame writes a frame as a PNG: 8 bits per channel, RGB, not interlaced,
// with a text chunk right after the header that says the frame is of this tool
// and belongs to the set id. The same frame and set are always the same bytes.
func encodeFrame(frame domain.FrameImage, id domain.FrameSetID) ([]byte, error) {
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
	out = append(out, textChunk(markPrefix+string(id))...)
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

// readMark reads, from the start of a file, the set a frame of this tool says it
// belongs to; ok is false for a file that is not a PNG, that has no such text
// chunk before its image data, or whose chunk does not check out.
func readMark(head []byte) (id domain.FrameSetID, ok bool) {
	if len(head) < len(pngSignature) || !bytes.Equal(head[:len(pngSignature)], pngSignature) {
		return "", false
	}

	for position := len(pngSignature); position+8 <= len(head); {
		length := int(binary.BigEndian.Uint32(head[position:]))
		kind := string(head[position+4 : position+8])
		end := position + 8 + length
		if kind == "IDAT" || kind == "IEND" || end+4 > len(head) || length < 0 {
			return "", false
		}

		if kind == "tEXt" {
			text := head[position+8 : end]
			if binary.BigEndian.Uint32(head[end:]) == crc32.ChecksumIEEE(head[position+4:end]) && bytes.HasPrefix(text, []byte(markPrefix)) {
				return domain.FrameSetID(text[len(markPrefix):]), true
			}
		}
		position = end + 4
	}
	return "", false
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
