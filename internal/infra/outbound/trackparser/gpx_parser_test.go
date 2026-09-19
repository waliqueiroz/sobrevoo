package trackparser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/trackparser"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

// erroringReader always fails to read, simulating an I/O failure while
// streaming the track file's content.
type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

func Test_GPXParser_Parse(t *testing.T) {
	t.Run("should parse a valid GPX with altitude and time in every point", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.ValidGPXWithAltitudeAndTime())

		// when
		track, err := parser.Parse(content)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.FormatGPX, track.Format)
		require.Len(t, track.Points, 3)
		for _, p := range track.Points {
			assert.True(t, p.HasElevation())
			assert.True(t, p.HasTime())
		}
		assert.InDelta(t, 40.4168, track.Points[0].Latitude, 0.0001)
		assert.InDelta(t, -3.7038, track.Points[0].Longitude, 0.0001)
	})

	t.Run("should parse a valid GPX without altitude", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.ValidGPXWithoutAltitude())

		// when
		track, err := parser.Parse(content)

		// then
		require.NoError(t, err)
		require.Len(t, track.Points, 3)
		for _, p := range track.Points {
			assert.False(t, p.HasElevation())
			assert.True(t, p.HasTime())
		}
	})

	t.Run("should parse a valid GPX without time", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.ValidGPXWithoutTime())

		// when
		track, err := parser.Parse(content)

		// then
		require.NoError(t, err)
		require.Len(t, track.Points, 3)
		for _, p := range track.Points {
			assert.True(t, p.HasElevation())
			assert.False(t, p.HasTime())
		}
	})

	t.Run("should return ErrEmptyFile for empty content", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.EmptyContent())

		// when
		_, err := parser.Parse(content)

		// then
		assert.ErrorIs(t, err, domain.ErrEmptyFile)
	})

	t.Run("should return ErrUnsupportedFormat for content that is not GPX", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.NonGPXContent())

		// when
		_, err := parser.Parse(content)

		// then
		assert.ErrorIs(t, err, domain.ErrUnsupportedFormat)
	})

	t.Run("should return a distinct error for recognized but malformed GPX content", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.MalformedGPX())

		// when
		_, err := parser.Parse(content)

		// then: malformed GPX is a distinct failure from unrecognized format (research.md item 3)
		require.Error(t, err)
		assert.False(t, errors.Is(err, domain.ErrUnsupportedFormat))
		assert.False(t, errors.Is(err, domain.ErrEmptyFile))
	})

	t.Run("should propagate a read error from the underlying reader", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := erroringReader{}

		// when
		_, err := parser.Parse(content)

		// then
		require.Error(t, err)
		assert.False(t, errors.Is(err, domain.ErrEmptyFile))
		assert.False(t, errors.Is(err, domain.ErrUnsupportedFormat))
	})

	t.Run("should not reject a single point itself, leaving that to the service layer", func(t *testing.T) {
		// given
		parser := trackparser.NewGPXParser()
		content := strings.NewReader(helper.GPXWithSinglePoint())

		// when
		track, err := parser.Parse(content)

		// then
		require.NoError(t, err)
		assert.Len(t, track.Points, 1)
	})
}
