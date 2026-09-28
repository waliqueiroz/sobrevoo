package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_NewFrameImage(t *testing.T) {
	t.Run("should fill every pixel with the given background color", func(t *testing.T) {
		// given
		background := domain.RGB{R: 0x20, G: 0x26, B: 0x2E}

		// when
		image := domain.NewFrameImage(domain.Resolution{Width: 4, Height: 2}, background)

		// then
		for y := 0; y < 2; y++ {
			for x := 0; x < 4; x++ {
				assert.Equal(t, background, image.At(x, y))
			}
		}
	})

	t.Run("should fill with a different color when a different background is given", func(t *testing.T) {
		// given
		background := domain.RGB{R: 0xFF, G: 0xFF, B: 0xFF}

		// when
		image := domain.NewFrameImage(domain.Resolution{Width: 2, Height: 2}, background)

		// then
		assert.Equal(t, background, image.At(0, 0))
		assert.Equal(t, background, image.At(1, 1))
	})
}
