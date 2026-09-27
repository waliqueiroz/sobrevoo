package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_VideoSummary_Duration(t *testing.T) {
	t.Run("should be the frames over the frame rate, for a whole rate", func(t *testing.T) {
		// given
		summary := domain.VideoSummary{Frames: 380, FrameRate: 10}

		// when / then
		assert.Equal(t, 38*time.Second, summary.Duration())
	})

	t.Run("should be 45 seconds for 1350 frames at 30 frames per second", func(t *testing.T) {
		// given
		summary := domain.VideoSummary{Frames: 1350, FrameRate: 30}

		// when / then
		assert.Equal(t, 45*time.Second, summary.Duration())
	})

	t.Run("should be exact to the millisecond for a rate that is not whole", func(t *testing.T) {
		// given
		summary := domain.VideoSummary{Frames: 1350, FrameRate: 29.97}

		// when / then
		assert.Equal(t, 45045*time.Millisecond, summary.Duration())
	})

	t.Run("should round a frame that does not fill a millisecond to the nearest one", func(t *testing.T) {
		// given
		summary := domain.VideoSummary{Frames: 1, FrameRate: 30}

		// when / then
		assert.Equal(t, 33*time.Millisecond, summary.Duration())
	})

	t.Run("should hold the longest plan there is", func(t *testing.T) {
		// given
		summary := domain.VideoSummary{Frames: 432000, FrameRate: 1}

		// when / then
		assert.Equal(t, 432000*time.Second, summary.Duration())
	})

	t.Run("should be zero when the frame rate is zero or negative, and not fail", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, time.Duration(0), domain.VideoSummary{Frames: 100, FrameRate: 0}.Duration())
		assert.Equal(t, time.Duration(0), domain.VideoSummary{Frames: 100, FrameRate: -30}.Duration())
	})

	t.Run("should be zero when there are no frames", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, time.Duration(0), domain.VideoSummary{FrameRate: 30}.Duration())
	})
}

func Test_EncoderInfo_String(t *testing.T) {
	t.Run("should say the program, its version and the codec", func(t *testing.T) {
		// given
		info := domain.EncoderInfo{Name: "ffmpeg", Version: "7.1", Codec: "libx264"}

		// when / then
		assert.Equal(t, "ffmpeg 7.1 (libx264)", info.String())
	})

	t.Run("should leave the version out when it is not known", func(t *testing.T) {
		// given
		info := domain.EncoderInfo{Name: "ffmpeg", Codec: "libx264"}

		// when / then
		assert.Equal(t, "ffmpeg (libx264)", info.String())
	})
}
