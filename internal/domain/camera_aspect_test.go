package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_ParseAspectRatio(t *testing.T) {
	t.Run("should read a vertical ratio", func(t *testing.T) {
		// when
		aspect, err := domain.ParseAspectRatio("9:16")

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.AspectRatio{Width: 9, Height: 16}, aspect)
		assert.Equal(t, "9:16", aspect.String())
		assert.InDelta(t, 0.5625, aspect.Ratio(), 1e-12)
	})

	t.Run("should read a horizontal ratio, ignoring surrounding spaces", func(t *testing.T) {
		// when
		aspect, err := domain.ParseAspectRatio(" 16:9 ")

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.LandscapeAspectRatio, aspect)
	})

	t.Run("should accept the ratios at both ends of the range", func(t *testing.T) {
		// when
		tallest, tallestErr := domain.ParseAspectRatio("1:5")
		widest, widestErr := domain.ParseAspectRatio("5:1")

		// then
		assert.NoError(t, tallestErr)
		assert.NoError(t, widestErr)
		assert.Equal(t, domain.AspectRatio{Width: 1, Height: 5}, tallest)
		assert.Equal(t, domain.AspectRatio{Width: 5, Height: 1}, widest)
	})

	t.Run("should refuse text without a colon, quoting what it received", func(t *testing.T) {
		// when
		_, err := domain.ParseAspectRatio("vertical")

		// then
		require.ErrorIs(t, err, domain.ErrInvalidAspectRatio)
		assert.ErrorContains(t, err, `"vertical"`)
	})

	t.Run("should refuse sides that are not whole numbers", func(t *testing.T) {
		// when
		_, decimalErr := domain.ParseAspectRatio("9.5:16")
		_, emptyErr := domain.ParseAspectRatio("9:")
		_, threeErr := domain.ParseAspectRatio("9:16:1")

		// then
		assert.ErrorIs(t, decimalErr, domain.ErrInvalidAspectRatio)
		assert.ErrorIs(t, emptyErr, domain.ErrInvalidAspectRatio)
		assert.ErrorIs(t, threeErr, domain.ErrInvalidAspectRatio)
	})

	t.Run("should refuse zero and negative sides", func(t *testing.T) {
		// when
		_, zeroErr := domain.ParseAspectRatio("0:16")
		_, negativeErr := domain.ParseAspectRatio("-9:16")

		// then
		assert.ErrorIs(t, zeroErr, domain.ErrInvalidAspectRatio)
		assert.ErrorIs(t, negativeErr, domain.ErrInvalidAspectRatio)
	})

	t.Run("should refuse a ratio outside 1:5 to 5:1, saying the limits", func(t *testing.T) {
		// when
		_, tallErr := domain.ParseAspectRatio("1:6")
		_, wideErr := domain.ParseAspectRatio("6:1")

		// then
		require.ErrorIs(t, tallErr, domain.ErrInvalidAspectRatio)
		assert.ErrorContains(t, tallErr, "between 1:5 and 5:1")
		assert.ErrorIs(t, wideErr, domain.ErrInvalidAspectRatio)
	})

	t.Run("should refuse a side above 1000", func(t *testing.T) {
		// when
		_, err := domain.ParseAspectRatio("1001:1001")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidAspectRatio)
	})
}
