package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_ParseColor(t *testing.T) {
	t.Run("should accept a hex color, upper or lower case", func(t *testing.T) {
		// given / when
		upper, errUpper := domain.ParseColor("#FFB000")
		lower, errLower := domain.ParseColor("#ffb000")

		// then
		require.NoError(t, errUpper)
		require.NoError(t, errLower)
		assert.Equal(t, domain.RGB{R: 0xFF, G: 0xB0, B: 0x00}, upper)
		assert.Equal(t, domain.RGB{R: 0xFF, G: 0xB0, B: 0x00}, lower)
	})

	t.Run("should refuse a text with no leading #", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("FFB000")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
		assert.ErrorContains(t, err, `"FFB000"`)
		assert.ErrorContains(t, err, "#RRGGBB")
	})

	t.Run("should refuse a text with fewer than 6 digits after #", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("#FB0")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should refuse a text with more than 6 digits after #", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("#FFB0001")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should refuse a digit outside the hexadecimal alphabet", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("#GGGGGG")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should refuse a 3-digit shorthand", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("#FB0")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should refuse a color with an alpha channel", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("#FFB000FF")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should refuse a named color", func(t *testing.T) {
		// given / when
		_, err := domain.ParseColor("orange")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})
}

func Test_NewAppearance(t *testing.T) {
	trail, marker, background := domain.RGB{R: 0xFF, G: 0xB0}, domain.RGB{R: 0xE5, G: 0x25, B: 0x2A}, domain.RGB{R: 0x20, G: 0x26, B: 0x2E}

	t.Run("should accept the trail width ratio and the marker radius ratio at their limits", func(t *testing.T) {
		// given / when
		low, errLow := domain.NewAppearance(trail, domain.MinTrailWidthRatio, marker, domain.MinMarkerRadiusRatio, background)
		high, errHigh := domain.NewAppearance(trail, domain.MaxTrailWidthRatio, marker, domain.MaxMarkerRadiusRatio, background)

		// then
		require.NoError(t, errLow)
		require.NoError(t, errHigh)
		assert.Equal(t, domain.MinTrailWidthRatio, low.TrailWidthRatio)
		assert.Equal(t, domain.MaxTrailWidthRatio, high.TrailWidthRatio)
	})

	t.Run("should refuse a trail width ratio under the minimum", func(t *testing.T) {
		// given / when
		_, err := domain.NewAppearance(trail, 0.0001, marker, 0.01, background)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidTrailWidth)
		assert.ErrorContains(t, err, "0.0001")
		assert.ErrorContains(t, err, "0.0005")
		assert.ErrorContains(t, err, "0.05")
	})

	t.Run("should refuse a trail width ratio over the maximum", func(t *testing.T) {
		// given / when
		_, err := domain.NewAppearance(trail, 0.1, marker, 0.01, background)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidTrailWidth)
	})

	t.Run("should refuse a marker radius ratio under the minimum", func(t *testing.T) {
		// given / when
		_, err := domain.NewAppearance(trail, 0.01, marker, 0.0001, background)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidMarkerRadius)
		assert.ErrorContains(t, err, "0.0001")
		assert.ErrorContains(t, err, "0.001")
		assert.ErrorContains(t, err, "0.1")
	})

	t.Run("should refuse a marker radius ratio over the maximum", func(t *testing.T) {
		// given / when
		_, err := domain.NewAppearance(trail, 0.01, marker, 0.2, background)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidMarkerRadius)
	})
}

func Test_Appearance_Fingerprint(t *testing.T) {
	base := func() domain.Appearance {
		a, err := domain.NewAppearance(domain.RGB{R: 0xFF, G: 0xB0}, 0.005, domain.RGB{R: 0xE5, G: 0x25, B: 0x2A}, 0.012, domain.RGB{R: 0x20, G: 0x26, B: 0x2E})
		require.NoError(t, err)
		return a
	}

	t.Run("should be the same text for the same five values, called twice", func(t *testing.T) {
		// given
		a := base()

		// when / then
		assert.Equal(t, a.Fingerprint(), a.Fingerprint())
	})

	t.Run("should differ when the trail color differs", func(t *testing.T) {
		// given
		a, b := base(), base()
		b.TrailColor = domain.RGB{R: 0x00, G: 0xFF, B: 0x00}

		// when / then
		assert.NotEqual(t, a.Fingerprint(), b.Fingerprint())
	})

	t.Run("should differ when the trail width ratio differs", func(t *testing.T) {
		// given
		a, b := base(), base()
		b.TrailWidthRatio = 0.02

		// when / then
		assert.NotEqual(t, a.Fingerprint(), b.Fingerprint())
	})

	t.Run("should differ when the marker color differs", func(t *testing.T) {
		// given
		a, b := base(), base()
		b.MarkerColor = domain.RGB{R: 0x00, G: 0x00, B: 0xFF}

		// when / then
		assert.NotEqual(t, a.Fingerprint(), b.Fingerprint())
	})

	t.Run("should differ when the marker radius ratio differs", func(t *testing.T) {
		// given
		a, b := base(), base()
		b.MarkerRadiusRatio = 0.05

		// when / then
		assert.NotEqual(t, a.Fingerprint(), b.Fingerprint())
	})

	t.Run("should differ when the background color differs", func(t *testing.T) {
		// given
		a, b := base(), base()
		b.BackgroundColor = domain.RGB{R: 0xFF, G: 0xFF, B: 0xFF}

		// when / then
		assert.NotEqual(t, a.Fingerprint(), b.Fingerprint())
	})
}
