package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_FlightStage_String(t *testing.T) {
	t.Run("should say treating the track for the first stage", func(t *testing.T) {
		assert.Equal(t, "treating the track", domain.StageTrackProcessing.String())
	})

	t.Run("should say planning the camera for the second stage", func(t *testing.T) {
		assert.Equal(t, "planning the camera", domain.StageCameraPlanning.String())
	})

	t.Run("should say slicing the geo data for the third stage", func(t *testing.T) {
		assert.Equal(t, "slicing the geo data", domain.StageGeoDataSlicing.String())
	})

	t.Run("should say drawing the frames for the fourth stage", func(t *testing.T) {
		assert.Equal(t, "drawing the frames", domain.StageFrameRendering.String())
	})

	t.Run("should say encoding the video for the fifth stage", func(t *testing.T) {
		assert.Equal(t, "encoding the video", domain.StageVideoEncoding.String())
	})
}
