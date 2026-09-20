package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func followingFrames(plan domain.CameraPlan) []domain.CameraFrame {
	var frames []domain.CameraFrame
	for _, f := range plan.Frames {
		if f.Phase == domain.PhaseFollowing {
			frames = append(frames, f)
		}
	}
	return frames
}

func Test_PlanCamera_DistanceLevels(t *testing.T) {
	tuning := defaultTuning()
	points := lineOfKm(10)

	plan := func(t *testing.T, level domain.Level) domain.CameraPlan {
		t.Helper()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(120e9).WithDistance(level).Build()
		result, err := domain.PlanCamera(treatedOf(points), parameters, tuning)
		require.NoError(t, err)
		return result
	}

	t.Run("should keep the camera farther from the marker with a high distance than with a low one, in every following frame", func(t *testing.T) {
		// given
		low, high := followingFrames(plan(t, domain.LevelLow)), followingFrames(plan(t, domain.LevelHigh))
		require.Len(t, high, len(low))

		for i := range low {
			// then
			assert.Greater(t, high[i].CameraToMarkerDistance, low[i].CameraToMarkerDistance, "following frame %d", i)
		}
	})

	t.Run("should order low, medium and high by distance", func(t *testing.T) {
		// given
		low, medium, high := plan(t, domain.LevelLow), plan(t, domain.LevelMedium), plan(t, domain.LevelHigh)

		// then
		assert.Less(t, low.Summary.MinCameraDistance, medium.Summary.MinCameraDistance)
		assert.Less(t, medium.Summary.MinCameraDistance, high.Summary.MinCameraDistance)
	})
}

func Test_PlanCamera_TiltLevels(t *testing.T) {
	tuning := defaultTuning()
	points := lineOfKm(10)

	middleTilt := func(t *testing.T, level domain.Level) float64 {
		t.Helper()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(120e9).WithTilt(level).Build()
		plan, err := domain.PlanCamera(treatedOf(points), parameters, tuning)
		require.NoError(t, err)
		frames := followingFrames(plan)
		return frames[len(frames)/2].Tilt
	}

	t.Run("should look from more vertical with a high tilt than with a low one", func(t *testing.T) {
		// when
		low, medium, high := middleTilt(t, domain.LevelLow), middleTilt(t, domain.LevelMedium), middleTilt(t, domain.LevelHigh)

		// then
		assert.InDelta(t, 25.0, low, 0.01)
		assert.InDelta(t, 45.0, medium, 0.01)
		assert.InDelta(t, 65.0, high, 0.01)
	})

	t.Run("should keep the tilt constant through the following phase", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(120e9).WithTilt(domain.LevelHigh).Build()
		plan, err := domain.PlanCamera(treatedOf(points), parameters, tuning)
		require.NoError(t, err)

		// when
		frames := followingFrames(plan)

		// then: away from the junctions with the opening and the closing
		for _, f := range frames[len(frames)/10 : len(frames)-len(frames)/10] {
			assert.InDelta(t, 65.0, f.Tilt, 0.01)
		}
	})
}
