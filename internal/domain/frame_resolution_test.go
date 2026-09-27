package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_NewResolution(t *testing.T) {
	t.Run("should accept the usual resolutions and the limits", func(t *testing.T) {
		// given / when
		full, errFull := domain.NewResolution(1920, 1080)
		smallest, errSmallest := domain.NewResolution(180, 180)
		biggest, errBiggest := domain.NewResolution(3840, 2160)
		portrait, errPortrait := domain.NewResolution(2160, 3840)
		ultrawide, errUltrawide := domain.NewResolution(3840, 1080)

		// then
		require.NoError(t, errFull)
		require.NoError(t, errSmallest)
		require.NoError(t, errBiggest)
		require.NoError(t, errPortrait)
		require.NoError(t, errUltrawide)
		assert.Equal(t, domain.Resolution{Width: 1920, Height: 1080}, full)
		assert.Equal(t, domain.Resolution{Width: 180, Height: 180}, smallest)
		assert.Equal(t, domain.Resolution{Width: 3840, Height: 2160}, biggest)
		assert.Equal(t, domain.Resolution{Width: 2160, Height: 3840}, portrait)
		assert.Equal(t, domain.Resolution{Width: 3840, Height: 1080}, ultrawide)
	})

	t.Run("should refuse an odd dimension", func(t *testing.T) {
		// given / when
		_, errWidth := domain.NewResolution(1921, 1080)
		_, errHeight := domain.NewResolution(1920, 1081)

		// then
		assert.ErrorIs(t, errWidth, domain.ErrInvalidResolution)
		assert.ErrorContains(t, errWidth, "1921x1080")
		assert.ErrorContains(t, errWidth, "even")
		assert.ErrorIs(t, errHeight, domain.ErrInvalidResolution)
	})

	t.Run("should refuse a zero or negative dimension", func(t *testing.T) {
		// given / when
		_, errZero := domain.NewResolution(0, 0)
		_, errNegative := domain.NewResolution(-1920, 1080)

		// then
		assert.ErrorIs(t, errZero, domain.ErrInvalidResolution)
		assert.ErrorIs(t, errNegative, domain.ErrInvalidResolution)
	})

	t.Run("should refuse a dimension under 180 or over 3840", func(t *testing.T) {
		// given / when
		_, errSmall := domain.NewResolution(178, 1080)
		_, errBigWidth := domain.NewResolution(3842, 1080)
		_, errBigHeight := domain.NewResolution(1080, 3842)

		// then
		assert.ErrorIs(t, errSmall, domain.ErrInvalidResolution)
		assert.ErrorContains(t, errSmall, "180")
		assert.ErrorContains(t, errSmall, "3840")
		assert.ErrorIs(t, errBigWidth, domain.ErrInvalidResolution)
		assert.ErrorIs(t, errBigHeight, domain.ErrInvalidResolution)
	})

	t.Run("should refuse a resolution over 8294400 pixels", func(t *testing.T) {
		// given / when
		_, err := domain.NewResolution(3840, 2200)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidResolution)
		assert.ErrorContains(t, err, "8294400")
	})

	t.Run("should count its pixels", func(t *testing.T) {
		// given
		resolution := domain.Resolution{Width: 1920, Height: 1080}

		// when / then
		assert.Equal(t, 2073600, resolution.Pixels())
	})
}

func Test_ParseResolution(t *testing.T) {
	t.Run("should read WIDTHxHEIGHT with a lowercase or an uppercase x", func(t *testing.T) {
		// given / when
		lower, errLower := domain.ParseResolution("1920x1080")
		upper, errUpper := domain.ParseResolution("1920X1080")

		// then
		require.NoError(t, errLower)
		require.NoError(t, errUpper)
		assert.Equal(t, domain.Resolution{Width: 1920, Height: 1080}, lower)
		assert.Equal(t, lower, upper)
	})

	t.Run("should refuse text that is not two whole numbers around an x", func(t *testing.T) {
		for _, text := range []string{"", "abc", "1920", "1920x", "x1080", "1920x1080x2", "19.5x1080", "-1920x1080", "1920 x 1080", " 1920x1080", "+1920x1080", "1e3x1080"} {
			// given / when
			_, err := domain.ParseResolution(text)

			// then
			assert.ErrorIs(t, err, domain.ErrInvalidResolution, "%q", text)
			assert.ErrorContains(t, err, "WIDTHxHEIGHT", "%q", text)
		}
	})

	t.Run("should apply the limits of NewResolution to what it reads", func(t *testing.T) {
		// given / when
		_, errOdd := domain.ParseResolution("1921x1080")
		_, errBig := domain.ParseResolution("8000x4500")
		_, errHuge := domain.ParseResolution("99999999999999999999x1080")

		// then
		assert.ErrorIs(t, errOdd, domain.ErrInvalidResolution)
		assert.ErrorIs(t, errBig, domain.ErrInvalidResolution)
		assert.ErrorIs(t, errHuge, domain.ErrInvalidResolution)
	})
}

func Test_Resolution_NarrowerThan(t *testing.T) {
	vertical := domain.AspectRatio{Width: 9, Height: 16}

	t.Run("should be narrower only when a vertical image is drawn for a horizontal video, not the other way round", func(t *testing.T) {
		// given
		horizontal := domain.Resolution{Width: 1920, Height: 1080}
		portrait := domain.Resolution{Width: 1080, Height: 1920}

		// then
		assert.False(t, horizontal.NarrowerThan(vertical))
		assert.True(t, portrait.NarrowerThan(domain.LandscapeAspectRatio))
	})

	t.Run("should not be narrower when the image has the shape of the video", func(t *testing.T) {
		// then
		assert.False(t, domain.Resolution{Width: 1080, Height: 1920}.NarrowerThan(vertical))
		assert.False(t, domain.Resolution{Width: 1920, Height: 1080}.NarrowerThan(domain.LandscapeAspectRatio))
		assert.False(t, domain.Resolution{Width: 960, Height: 540}.NarrowerThan(domain.LandscapeAspectRatio))
	})

	t.Run("should tolerate a difference of one percent", func(t *testing.T) {
		// then: 1076 x 1920 is 0.4% narrower than 9:16
		assert.False(t, domain.Resolution{Width: 1076, Height: 1920}.NarrowerThan(vertical))
		assert.True(t, domain.Resolution{Width: 1000, Height: 1920}.NarrowerThan(vertical))
	})

	t.Run("should not warn about a plan without an aspect ratio", func(t *testing.T) {
		// then
		assert.False(t, domain.Resolution{Width: 1080, Height: 1920}.NarrowerThan(domain.AspectRatio{}))
	})
}
