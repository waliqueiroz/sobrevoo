package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_ParseVideoQuality(t *testing.T) {
	t.Run("should read low", func(t *testing.T) {
		// given / when
		quality, err := domain.ParseVideoQuality("low")

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.VideoQualityLow, quality)
	})

	t.Run("should read medium", func(t *testing.T) {
		// given / when
		quality, err := domain.ParseVideoQuality("medium")

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.VideoQualityMedium, quality)
	})

	t.Run("should read high", func(t *testing.T) {
		// given / when
		quality, err := domain.ParseVideoQuality("high")

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.VideoQualityHigh, quality)
	})

	t.Run("should refuse a name that is not a quality, quoting it and listing the ones accepted", func(t *testing.T) {
		// given / when
		_, err := domain.ParseVideoQuality("ultra")

		// then
		require.Error(t, err)
		assert.EqualError(t, err, `"ultra": use one of low, medium, high`)
	})

	t.Run("should refuse an empty name and a name in another case", func(t *testing.T) {
		// given / when
		_, empty := domain.ParseVideoQuality("")
		_, upper := domain.ParseVideoQuality("HIGH")

		// then
		assert.Error(t, empty)
		assert.Error(t, upper)
	})
}

func Test_VideoQuality_String(t *testing.T) {
	t.Run("should be the name the user types, for each quality", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, "low", domain.VideoQualityLow.String())
		assert.Equal(t, "medium", domain.VideoQualityMedium.String())
		assert.Equal(t, "high", domain.VideoQualityHigh.String())
	})

	t.Run("should be read back by ParseVideoQuality", func(t *testing.T) {
		// given
		qualities := []domain.VideoQuality{domain.VideoQualityLow, domain.VideoQualityMedium, domain.VideoQualityHigh}

		for _, quality := range qualities {
			// when
			parsed, err := domain.ParseVideoQuality(quality.String())

			// then
			require.NoError(t, err)
			assert.Equal(t, quality, parsed)
		}
	})
}

func Test_VideoQuality_Order(t *testing.T) {
	t.Run("should go from low to medium to high", func(t *testing.T) {
		// given / when / then
		assert.Less(t, domain.VideoQualityLow, domain.VideoQualityMedium)
		assert.Less(t, domain.VideoQualityMedium, domain.VideoQualityHigh)
	})
}
