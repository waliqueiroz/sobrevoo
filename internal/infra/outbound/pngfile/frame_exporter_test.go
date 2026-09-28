package pngfile_test

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/pngfile"
)

const (
	setID  = domain.FrameSetID("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	planID = "99887766554433221100ffeeddccbbaa99887766554433221100ffeeddccbbaa"
)

// frameMark is what a frame of the set, drawn from the plan, says about itself.
var frameMark = domain.FrameMark{SetID: setID, PlanID: planID}

// drawnFrame is a small frame whose pixels tell where they are.
func drawnFrame() domain.FrameImage {
	frame := domain.NewFrameImage(domain.Resolution{Width: 8, Height: 6}, domain.RGB{})
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			frame.Set(x, y, domain.RGB{R: uint8(30 * x), G: uint8(40 * y), B: 99})
		}
	}
	return frame
}

func Test_FrameExporter_Export(t *testing.T) {
	t.Run("should write a PNG of the size of the frame", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "frame.png")

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, false)

		// then
		require.NoError(t, err)
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		decoded, decodeErr := png.Decode(bytes.NewReader(data))
		require.NoError(t, decodeErr)
		assert.Equal(t, 8, decoded.Bounds().Dx())
		assert.Equal(t, 6, decoded.Bounds().Dy())
		r, g, b, _ := decoded.At(3, 2).RGBA()
		assert.Equal(t, [3]uint32{90 * 257, 80 * 257, 99 * 257}, [3]uint32{r, g, b})
	})

	t.Run("should say, inside, which set the frame belongs to", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "frame.png")

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, false)

		// then
		require.NoError(t, err)
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Contains(t, string(data), "Sobrevoo\x00frame-set="+string(setID))
	})

	t.Run("should say, inside the image, which plan the frame was drawn from", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "frame.png")

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, false)

		// then
		require.NoError(t, err)
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Contains(t, string(data), "Sobrevoo\x00plan="+planID)
		assert.Less(t, bytes.Index(data, []byte("frame-set=")), bytes.Index(data, []byte("plan=")), "the set chunk comes first")
	})

	t.Run("should write the same bytes as the repository writes for a frame of a set", func(t *testing.T) {
		// given
		dir := t.TempDir()
		single := filepath.Join(dir, "single.png")

		// when
		require.NoError(t, pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, single, false))
		require.NoError(t, pngfile.NewFrameRepository().Save(dir, 7, frameMark, drawnFrame()))

		// then
		first, err1 := os.ReadFile(single)
		second, err2 := os.ReadFile(filepath.Join(dir, domain.FrameFileName(7)))
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, first, second)
	})

	t.Run("should leave only the frame in the directory", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, filepath.Join(dir, "frame.png"), false)

		// then
		require.NoError(t, err)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		require.Len(t, entries, 1)
		assert.Equal(t, "frame.png", entries[0].Name())
	})
}

func Test_FrameExporter_Export_Protection(t *testing.T) {
	t.Run("should refuse an existing file without overwrite, telling how to replace it and leaving it as it is", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "frame.png")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, false)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDestinationExists)
		assert.ErrorContains(t, err, path)
		assert.ErrorContains(t, err, "use --overwrite to replace it")
		content, _ := os.ReadFile(path)
		assert.Equal(t, "precious", string(content))
	})

	t.Run("should replace an existing file entirely with overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "frame.png")
		require.NoError(t, os.WriteFile(path, []byte("old and much longer than the new one, or so it would seem: "+string(make([]byte, 5000))), 0o600))

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, true)

		// then
		require.NoError(t, err)
		data, _ := os.ReadFile(path)
		decoded, decodeErr := png.Decode(bytes.NewReader(data))
		require.NoError(t, decodeErr)
		assert.Equal(t, 8, decoded.Bounds().Dx())
	})

	t.Run("should refuse a file in a directory that does not exist", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "missing", "frame.png")

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, false)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
		assert.NotErrorIs(t, err, domain.ErrFrameDestinationExists)
	})

	t.Run("should refuse a directory it cannot write to", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, filepath.Join(dir, "frame.png"), false)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
	})

	t.Run("should keep the file that was there when the new one cannot be written", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "frame.png")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		// when
		err := pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, path, true)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
		require.NoError(t, os.Chmod(dir, 0o700))
		content, _ := os.ReadFile(path)
		assert.Equal(t, "precious", string(content))
		assert.Equal(t, []string{"frame.png"}, names(t, dir))
	})
}
