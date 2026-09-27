package videoencoder

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// read gives the numbers of frames readProgress reports for what ffmpeg wrote.
func read(output string) []int {
	var reported []int
	readProgress(strings.NewReader(output), func(encoded int) { reported = append(reported, encoded) })
	return reported
}

func Test_readProgress(t *testing.T) {
	t.Run("should report the frames of each block, in order", func(t *testing.T) {
		// given
		output := "frame=10\nfps=25.0\nprogress=continue\n" +
			"frame=60\nfps=25.0\nprogress=continue\n" +
			"frame=120\nfps=25.0\nprogress=end\n"

		// when / then
		assert.Equal(t, []int{10, 60, 120}, read(output))
	})

	t.Run("should ignore the other keys, the blank lines and what is not a key", func(t *testing.T) {
		// given
		output := "bitrate=  1024.0kbits/s\nout_time_us=1000000\n\nnot a key at all\nprogress=continue\nframe=7\n"

		// when / then
		assert.Equal(t, []int{7}, read(output))
	})

	t.Run("should ignore a number of frames that is not known yet", func(t *testing.T) {
		// given
		output := "frame=N/A\nframe=\nframe=abc\nframe=5\n"

		// when / then
		assert.Equal(t, []int{5}, read(output))
	})

	t.Run("should not report a number that does not grow", func(t *testing.T) {
		// given
		output := "frame=10\nframe=10\nframe=8\nframe=11\n"

		// when / then
		assert.Equal(t, []int{10, 11}, read(output))
	})

	t.Run("should read a last line that has no line break", func(t *testing.T) {
		// given
		output := "frame=3\nframe=9"

		// when / then
		assert.Equal(t, []int{3, 9}, read(output))
	})

	t.Run("should report nothing for an empty output", func(t *testing.T) {
		// given / when / then
		assert.Empty(t, read(""))
	})

	t.Run("should not fail for a line longer than it can hold, and go on to the next", func(t *testing.T) {
		// given
		output := "frame=1\n" + strings.Repeat("x", 200_000) + "\nframe=2\n"

		// when
		reported := read(output)

		// then: what follows a line too long is drained, not read
		assert.Equal(t, []int{1}, reported)
	})

	t.Run("should drain what is left, so the program that writes it does not block", func(t *testing.T) {
		// given
		reader := &countingReader{data: []byte("frame=1\n" + strings.Repeat("x", 300_000))}

		// when
		readProgress(reader, nil)

		// then
		assert.Equal(t, len(reader.data), reader.read)
	})

	t.Run("should accept no function to report to", func(t *testing.T) {
		// given / when / then
		assert.NotPanics(t, func() { readProgress(strings.NewReader("frame=1\n"), nil) })
	})
}

// countingReader is a reader that counts what was read from it.
type countingReader struct {
	data []byte
	read int
}

func (r *countingReader) Read(p []byte) (int, error) {
	if r.read >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.read:])
	r.read += n
	return n, nil
}
