package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// treatedOf wraps points as a treated track whose cleaned points and route are
// the same, for tests that do not depend on their difference.
func treatedOf(points []domain.TrackPoint) domain.TreatedTrack {
	return domain.TreatedTrack{CleanedPoints: points, Route: domain.Route{Points: points}}
}

func abs(v float64) float64 { return math.Abs(v) }

func hypot(x, y float64) float64 { return math.Hypot(x, y) }

// angleStep is the shortest-arc difference between two headings, in degrees.
func angleStep(from, to float64) float64 {
	return math.Abs(math.Mod(to-from+540, 360) - 180)
}

// assertSmooth checks every pair of consecutive frames against the
// smoothness limits (with a small tolerance for the plan's quantization).
func assertSmooth(t *testing.T, plan domain.CameraPlan, tuning domain.CameraTuning) {
	t.Helper()

	fps := plan.Parameters.FrameRate
	const tolerance = 0.005

	for i := 1; i < len(plan.Frames); i++ {
		a, b := plan.Frames[i-1], plan.Frames[i]

		assert.LessOrEqual(t, angleStep(a.Heading, b.Heading), tuning.MaxHeadingRateDegPerSecond/fps+tolerance, "heading step at frame %d", i)
		assert.LessOrEqual(t, abs(b.Tilt-a.Tilt), tuning.MaxTiltRateDegPerSecond/fps+tolerance, "tilt step at frame %d", i)
		assert.LessOrEqual(t, abs(math.Log(b.CameraToMarkerDistance/a.CameraToMarkerDistance)), tuning.MaxLogDistanceRatePerSecond/fps+tolerance/100, "zoom step at frame %d", i)

		horizontal := domain.Haversine(
			domain.TrackPoint{Latitude: a.CameraLatitude, Longitude: a.CameraLongitude},
			domain.TrackPoint{Latitude: b.CameraLatitude, Longitude: b.CameraLongitude},
		)
		move := math.Hypot(horizontal, b.CameraAltitude-a.CameraAltitude)
		limit := 1.5 * tuning.MaxTargetSpeedInDistances * math.Max(a.CameraToMarkerDistance, b.CameraToMarkerDistance) / fps
		assert.LessOrEqual(t, move, limit, "camera movement at frame %d", i)
	}
}

// assertPhaseOrder checks the frames are Opening*, Following+, Closing*.
func assertPhaseOrder(t *testing.T, plan domain.CameraPlan) {
	t.Helper()

	rank := map[domain.Phase]int{domain.PhaseOpening: 0, domain.PhaseFollowing: 1, domain.PhaseClosing: 2}
	seenFollowing := false
	for i, f := range plan.Frames {
		if i > 0 {
			assert.GreaterOrEqual(t, rank[f.Phase], rank[plan.Frames[i-1].Phase], "phase order at frame %d", i)
		}
		seenFollowing = seenFollowing || f.Phase == domain.PhaseFollowing
	}
	assert.True(t, seenFollowing, "at least one following frame")
}

// assertMarkerMonotonic checks the marker never moves backwards.
func assertMarkerMonotonic(t *testing.T, plan domain.CameraPlan) {
	t.Helper()

	for i := 1; i < len(plan.Frames); i++ {
		assert.GreaterOrEqual(t, plan.Frames[i].MarkerDistance, plan.Frames[i-1].MarkerDistance, "marker distance at frame %d", i)
	}
}
