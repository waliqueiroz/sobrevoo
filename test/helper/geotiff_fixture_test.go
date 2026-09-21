package helper

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/tiff/lzw"
)

func Test_lzwEncode(t *testing.T) {
	t.Run("should produce data the TIFF LZW decoder reads back, across the code width changes and table resets", func(t *testing.T) {
		// given
		random := rand.New(rand.NewSource(1))
		data := make([]byte, 60000)
		for i := range data {
			data[i] = byte(random.Intn(6)) // few symbols: long matches, big table
		}
		noisy := make([]byte, 20000)
		random.Read(noisy)
		data = append(data, noisy...)

		// when
		encoded := lzwEncode(data)
		decoded, err := io.ReadAll(lzw.NewReader(bytes.NewReader(encoded), lzw.MSB, 8))

		// then
		require.NoError(t, err)
		assert.Equal(t, data, decoded)
	})

	t.Run("should round-trip an empty input", func(t *testing.T) {
		// when
		decoded, err := io.ReadAll(lzw.NewReader(bytes.NewReader(lzwEncode(nil)), lzw.MSB, 8))

		// then
		require.NoError(t, err)
		assert.Empty(t, decoded)
	})

	t.Run("should round-trip a single byte", func(t *testing.T) {
		// when
		decoded, err := io.ReadAll(lzw.NewReader(bytes.NewReader(lzwEncode([]byte{7})), lzw.MSB, 8))

		// then
		require.NoError(t, err)
		assert.Equal(t, []byte{7}, decoded)
	})
}
