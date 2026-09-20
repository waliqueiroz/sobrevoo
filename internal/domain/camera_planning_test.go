package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

// routeShapes are the synthetic routes the smoothness properties run on.
func routeShapes() map[string][]domain.TrackPoint {
	return map[string][]domain.TrackPoint{
		"a straight line":                   builddomain.NewSyntheticRouteBuilder().WithLine(10000, 45).WithConstantSpeed(4).Build(),
		"a U-turn":                          builddomain.NewSyntheticRouteBuilder().WithUTurn(3000).WithConstantSpeed(4).Build(),
		"an out-and-back on the same path":  builddomain.NewSyntheticRouteBuilder().WithOutAndBack(3000).Build(),
		"a circle of five laps":             builddomain.NewSyntheticRouteBuilder().WithCircle(150, 5).Build(),
		"a route crossing the antimeridian": builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(10000, 90).Build(),
		"a route at latitude 85":            builddomain.NewSyntheticRouteBuilder().WithOrigin(85, 10).WithLine(10000, 45).Build(),
		"a route at latitude -85":           builddomain.NewSyntheticRouteBuilder().WithOrigin(-85, -30).WithOutAndBack(4000).Build(),
	}
}

func Test_PlanCamera(t *testing.T) {
	tuning := defaultTuning()

	t.Run("should produce exactly one frame per instant of the video", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).WithConstantSpeed(5).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(60 * time.Second).WithFrameRate(30).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Len(t, plan.Frames, 1800)
		assert.Equal(t, 1800, plan.Summary.FrameCount)
		for i, f := range plan.Frames {
			assert.Equal(t, i, f.Index)
		}
	})

	t.Run("should time frames by their index over the frame rate", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(60 * time.Second).WithFrameRate(24).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), plan.Frames[0].Time)
		assert.Equal(t, time.Second, plan.Frames[24].Time)
	})

	t.Run("should start the marker at zero and end it at the total length, never going back", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).WithConstantSpeed(5).Build()
		parameters := builddomain.NewPlanParametersBuilder().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0.0, plan.Frames[0].MarkerDistance)
		assert.InDelta(t, (domain.Route{Points: points}).Length(), plan.Frames[len(plan.Frames)-1].MarkerDistance, 0.001)
		assertMarkerMonotonic(t, plan)
	})

	t.Run("should be deterministic: a hundred calls with the same input give identical plans", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithUTurn(2000).WithConstantSpeed(4).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(40 * time.Second).Build()
		first, err := treatedOf(points).PlanCamera(parameters, tuning)
		require.NoError(t, err)

		for i := 0; i < 100; i++ {
			// when
			plan, err := treatedOf(points).PlanCamera(parameters, tuning)

			// then
			require.NoError(t, err)
			require.Equal(t, first, plan)
		}
	})

	t.Run("should quantize every value to the documented precision", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(-23.55, -46.63).WithLine(5000, 30).WithConstantSpeed(5).Build()
		parameters := builddomain.NewPlanParametersBuilder().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		onGrid := func(v, step float64) bool {
			return math.Abs(v/step-math.Round(v/step)) < 1e-6
		}
		for _, f := range plan.Frames {
			assert.True(t, onGrid(f.CameraLatitude, 1e-7), "lat %v", f.CameraLatitude)
			assert.True(t, onGrid(f.CameraLongitude, 1e-7), "lon %v", f.CameraLongitude)
			assert.True(t, onGrid(f.CameraAltitude, 1e-3), "alt %v", f.CameraAltitude)
			assert.True(t, onGrid(f.Heading, 1e-3), "heading %v", f.Heading)
			assert.True(t, onGrid(f.Tilt, 1e-3), "tilt %v", f.Tilt)
			assert.True(t, onGrid(f.MarkerDistance, 1e-3), "marker distance %v", f.MarkerDistance)
			assert.True(t, onGrid(f.CameraToMarkerDistance, 1e-3), "camera to marker %v", f.CameraToMarkerDistance)
		}
	})

	t.Run("should keep longitudes in [-180, 180) and produce no NaN or infinite values, even across the antimeridian", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(10000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		for _, f := range plan.Frames {
			for _, v := range []float64{f.CameraLatitude, f.CameraLongitude, f.CameraAltitude, f.Heading, f.Tilt, f.MarkerLatitude, f.MarkerLongitude, f.MarkerDistance, f.CameraToMarkerDistance} {
				assert.False(t, math.IsNaN(v) || math.IsInf(v, 0))
			}
			assert.GreaterOrEqual(t, f.CameraLongitude, -180.0)
			assert.Less(t, f.CameraLongitude, 180.0)
			assert.GreaterOrEqual(t, f.MarkerLongitude, -180.0)
			assert.Less(t, f.MarkerLongitude, 180.0)
			assert.GreaterOrEqual(t, f.Heading, 0.0)
			assert.Less(t, f.Heading, 360.0)
		}
	})

	t.Run("should cross the antimeridian without a longitude jump of a whole turn", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(10000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then: consecutive markers are never more than a fraction of a degree apart (mod 360)
		require.NoError(t, err)
		for i := 1; i < len(plan.Frames); i++ {
			step := math.Abs(math.Mod(plan.Frames[i].MarkerLongitude-plan.Frames[i-1].MarkerLongitude+540, 360) - 180)
			assert.Less(t, step, 0.05)
		}
	})

	t.Run("should use the distance as the time reference when the route has no time", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.TimeReferenceDistance, plan.TimeReference)
		assert.Equal(t, "no time data", plan.TimeFallbackReason)
		assert.Equal(t, domain.TimeReferenceDistance, plan.Summary.TimeReference)
	})

	t.Run("should use the clock as the time reference when the route has usable times", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).WithConstantSpeed(5).Build()
		parameters := builddomain.NewPlanParametersBuilder().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.TimeReferenceClock, plan.TimeReference)
		assert.Empty(t, plan.TimeFallbackReason)
	})

	t.Run("should report the effective duration and an automatic mode when no duration is requested", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		require.NotNil(t, plan.Parameters.Duration)
		assert.Equal(t, 28*time.Second, *plan.Parameters.Duration) // 15 s + 6 s × √5, rounded
		assert.Equal(t, domain.DurationModeAutomatic, plan.Summary.DurationMode)
		assert.Equal(t, 28*time.Second, plan.Summary.Duration)
	})

	t.Run("should use the requested duration exactly and an explicit mode", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(45 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, 45*time.Second, *plan.Parameters.Duration)
		assert.Equal(t, domain.DurationModeExplicit, plan.Summary.DurationMode)
		assert.Len(t, plan.Frames, 1350)
	})

	t.Run("should reject invalid parameters before doing any work", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().Build()
		parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(0).Build()

		// when
		_, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidFrameRate)
	})

	t.Run("should not change its input parameters", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

		// when
		_, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Nil(t, parameters.Duration)
	})

	t.Run("should report a leftover duration too short to leave a frame to follow the track", func(t *testing.T) {
		// given: a tuning whose opening and closing take the whole video
		tightTuning := tuning
		tightTuning.OpeningFraction = 0.5
		tightTuning.ClosingFraction = 0.5
		tightTuning.MinFollowDuration = 0
		tightTuning.MinPhaseDuration = 0
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(2 * time.Second).WithFrameRate(1).Build()

		// when
		_, err := treatedOf(points).PlanCamera(parameters, tightTuning)

		// then
		assert.ErrorIs(t, err, domain.ErrDurationTooShort)
	})
}

func Test_PlanCamera_Smoothness(t *testing.T) {
	tuning := defaultTuning()

	for name, points := range routeShapes() {
		t.Run("should respect every smoothness limit and the phase order on "+name, func(t *testing.T) {
			// given: a comfortable explicit duration
			parameters := builddomain.NewPlanParametersBuilder().WithDuration(120 * time.Second).Build()

			// when
			plan, err := treatedOf(points).PlanCamera(parameters, tuning)

			// then
			require.NoError(t, err)
			assertSmooth(t, plan, tuning)
			assertPhaseOrder(t, plan)
			assertMarkerMonotonic(t, plan)
		})
	}

	t.Run("should turn the camera gradually through a turn of more than 150 degrees", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithUTurn(3000).WithConstantSpeed(4).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(120 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then: heading changes by well over 150 degrees, over many frames, none of them abrupt
		require.NoError(t, err)
		first, last := plan.Frames[0].Heading, plan.Frames[len(plan.Frames)-1].Heading
		assert.Greater(t, angleStep(first, last), 150.0)
		assertSmooth(t, plan, tuning)
	})

	t.Run("should not rotate at high frequency on a route of many laps in the same place", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithCircle(100, 8).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(120 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then: the total heading rotation over the following phase is a small fraction of 8 full turns
		require.NoError(t, err)
		var rotation float64
		for i := 1; i < len(plan.Frames); i++ {
			rotation += angleStep(plan.Frames[i-1].Heading, plan.Frames[i].Heading)
		}
		assert.Less(t, rotation, 8*360.0/2)
	})
}

func Test_PlanCamera_OpeningAndClosing(t *testing.T) {
	tuning := defaultTuning()
	points := builddomain.NewSyntheticRouteBuilder().WithLine(10000, 45).WithConstantSpeed(4).Build()

	t.Run("should spend the configured share of frames on the opening and the closing", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(100 * time.Second).WithFrameRate(30).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then: 3000 frames, 10% each
		require.NoError(t, err)
		counts := map[domain.Phase]int{}
		for _, f := range plan.Frames {
			counts[f.Phase]++
		}
		assert.Equal(t, 300, counts[domain.PhaseOpening])
		assert.Equal(t, 300, counts[domain.PhaseClosing])
		assert.Equal(t, 2400, counts[domain.PhaseFollowing])
	})

	t.Run("should hold the marker at the start during the opening and at the end during the closing", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(100 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		total := (domain.Route{Points: points}).Length()
		for _, f := range plan.Frames {
			switch f.Phase {
			case domain.PhaseOpening:
				assert.Equal(t, 0.0, f.MarkerDistance)
			case domain.PhaseClosing:
				assert.InDelta(t, total, f.MarkerDistance, 0.001)
			}
		}
	})

	t.Run("should frame the whole track at the first and last frames, tilted 60 degrees, without turning", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(100 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		first, last := plan.Frames[0], plan.Frames[len(plan.Frames)-1]
		assert.Equal(t, domain.PhaseOpening, first.Phase)
		assert.Equal(t, domain.PhaseClosing, last.Phase)
		assert.InDelta(t, 60.0, first.Tilt, 0.001)
		assert.InDelta(t, 60.0, last.Tilt, 0.001)

		var firstFollowing, lastFollowing domain.CameraFrame
		for _, f := range plan.Frames {
			if f.Phase == domain.PhaseFollowing {
				if firstFollowing.Phase == "" {
					firstFollowing = f
				}
				lastFollowing = f
			}
		}
		assert.InDelta(t, firstFollowing.Heading, first.Heading, 0.5)
		assert.InDelta(t, lastFollowing.Heading, last.Heading, 0.5)

		plane := domain.NewLocalPlane(domain.Route{Points: points})
		for _, frame := range []domain.CameraFrame{first, last} {
			view := cameraViewOf(plane, frame)
			for _, p := range points {
				projected := plane.Project(p.Latitude, p.Longitude)
				assert.LessOrEqual(t, angleFromAxis(view, projected), tuning.OverviewVerticalFOVDegrees/2, "frame %d", frame.Index)
			}
		}
	})

	t.Run("should be farther from the marker at the first frame than in the middle of the following phase", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(100 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Greater(t, plan.Frames[0].CameraToMarkerDistance, plan.Frames[len(plan.Frames)/2].CameraToMarkerDistance)
	})
}

// cameraViewOf recovers the camera view of a frame for framing checks: the
// point the camera looks at is along its heading, at the tilt and altitude of
// the frame.
func cameraViewOf(plane domain.LocalPlane, f domain.CameraFrame) domain.CameraView {
	position := plane.Project(f.CameraLatitude, f.CameraLongitude)
	horizontal := f.CameraAltitude / math.Tan(f.Tilt*math.Pi/180)
	heading := f.Heading * math.Pi / 180
	return domain.CameraView{
		Target:      domain.PlanePoint{X: position.X + horizontal*math.Sin(heading), Y: position.Y + horizontal*math.Cos(heading)},
		Heading:     f.Heading,
		TiltDegrees: f.Tilt,
		Distance:    f.CameraAltitude / math.Sin(f.Tilt*math.Pi/180),
	}
}

func Test_PlanCamera_SmoothedSpans(t *testing.T) {
	tuning := defaultTuning()

	t.Run("should register the stretches where a limit had to act, with start no later than end", func(t *testing.T) {
		// given: a turn back forces the heading limit to act
		points := builddomain.NewSyntheticRouteBuilder().WithOutAndBack(3000).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(30 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, plan.Summary.SmoothedSpans)
		for _, span := range plan.Summary.SmoothedSpans {
			assert.LessOrEqual(t, span.Start, span.End)
			assert.Equal(t, domain.QuantityHeading, span.Quantity)
		}
	})

	t.Run("should register nothing for a straight route", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(5000, 90).WithConstantSpeed(5).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(60 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Empty(t, plan.Summary.SmoothedSpans)
	})

	t.Run("should list spans ordered by start time", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithCircle(150, 5).Build()
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(60 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		for i := 1; i < len(plan.Summary.SmoothedSpans); i++ {
			assert.LessOrEqual(t, plan.Summary.SmoothedSpans[i-1].Start, plan.Summary.SmoothedSpans[i].Start)
		}
	})
}

func Test_PlanCamera_RegionEquivalence(t *testing.T) {
	tuning := defaultTuning()
	parameters := builddomain.NewPlanParametersBuilder().WithDuration(60 * time.Second).Build()

	plan := func(t *testing.T, lat, lon float64) domain.CameraPlan {
		t.Helper()
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(lat, lon).WithUTurn(3000).WithConstantSpeed(4).Build()
		result, err := treatedOf(points).PlanCamera(parameters, tuning)
		require.NoError(t, err)
		return result
	}

	reference := plan(t, 0, 0)

	for name, origin := range map[string][2]float64{
		"the antimeridian": {10, 179.98},
		"a high latitude":  {85, 10},
		"the south":        {-45, -60},
	} {
		t.Run("should give an equivalent plan at "+name, func(t *testing.T) {
			// when
			other := plan(t, origin[0], origin[1])

			// then: same frame count, same smoothness, same summary within one percent
			assert.Len(t, other.Frames, len(reference.Frames))
			assertSmooth(t, other, tuning)
			assert.InEpsilon(t, reference.Summary.MinCameraAltitude, other.Summary.MinCameraAltitude, 0.01)
			assert.InEpsilon(t, reference.Summary.MaxCameraAltitude, other.Summary.MaxCameraAltitude, 0.01)
			assert.InEpsilon(t, reference.Summary.MinCameraDistance, other.Summary.MinCameraDistance, 0.01)
			assert.InEpsilon(t, reference.Summary.MaxCameraDistance, other.Summary.MaxCameraDistance, 0.01)
		})
	}
}

func Test_DefaultDuration(t *testing.T) {
	tuning := defaultTuning()
	durationFor := func(km float64) time.Duration {
		points := builddomain.NewSyntheticRouteBuilder().WithLine(km*1000, 90).Build()
		return defaultDuration(points, 30, domain.LevelMedium, domain.LevelMedium, tuning)
	}

	t.Run("should follow the curve 15 s + 6 s × sqrt(km), rounded to the second", func(t *testing.T) {
		// when / then
		assert.Equal(t, 21*time.Second, durationFor(1))
		assert.Equal(t, 28*time.Second, durationFor(5))
		assert.Equal(t, 42*time.Second, durationFor(20))
		assert.Equal(t, 57*time.Second, durationFor(50))
		assert.Equal(t, 100*time.Second, durationFor(200))
	})

	t.Run("should be at most 120 seconds for very long tracks", func(t *testing.T) {
		// when / then
		assert.Equal(t, 120*time.Second, durationFor(400))
	})

	t.Run("should be at least 20 seconds for very short tracks", func(t *testing.T) {
		// when / then
		assert.GreaterOrEqual(t, durationFor(0.1), 20*time.Second)
	})

	t.Run("should never decrease as the track grows", func(t *testing.T) {
		// given
		previous := time.Duration(0)
		for _, km := range []float64{0.1, 0.5, 1, 2, 5, 10, 20, 50, 100, 200, 400, 1000} {
			// when
			current := durationFor(km)

			// then
			assert.GreaterOrEqual(t, current, previous, "%v km", km)
			previous = current
		}
	})

	t.Run("should grow proportionally less than the length: ten times the length is not ten times the video", func(t *testing.T) {
		// when / then
		assert.Less(t, durationFor(50).Seconds(), 10*durationFor(5).Seconds())
		assert.Less(t, durationFor(20).Seconds(), 2*durationFor(10).Seconds())
	})

	t.Run("should be deterministic", func(t *testing.T) {
		// when / then
		assert.Equal(t, durationFor(20), durationFor(20))
	})
}

func Test_PlanCamera_LongStopHiddenBySimplification(t *testing.T) {
	tuning := defaultTuning()
	parameters := builddomain.NewPlanParametersBuilder().WithDuration(60 * time.Second).Build()

	// The cleaned points show a ten minute stop; the treated route is the
	// straight line a simplifier would reduce them to, which has no trace of it.
	cleaned := routeWithStop(600)
	route := []domain.TrackPoint{cleaned[0], cleaned[len(cleaned)/2], cleaned[len(cleaned)-1]}

	stillFrames := func(plan domain.CameraPlan) int {
		var still int
		following := followingFrames(plan)
		for i := 1; i < len(following); i++ {
			if following[i].MarkerDistance-following[i-1].MarkerDistance < 0.002 {
				still++
			}
		}
		return still
	}

	t.Run("should compress the stop found in the cleaned points even though the route no longer shows it", func(t *testing.T) {
		// given
		treated := domain.TreatedTrack{Cleaned: domain.Route{Points: cleaned}, Route: domain.Route{Points: route}}

		// when
		plan, err := treated.PlanCamera(parameters, tuning)

		// then: the marker stands still for a few frames only, at most 5 percent of the following phase
		require.NoError(t, err)
		still := stillFrames(plan)
		assert.Positive(t, still)
		assert.LessOrEqual(t, float64(still), 0.05*float64(len(followingFrames(plan))))
	})

	t.Run("should not find any stop when the cleaned points are the simplified route itself", func(t *testing.T) {
		// given
		treated := domain.TreatedTrack{Cleaned: domain.Route{Points: route}, Route: domain.Route{Points: route}}

		// when
		plan, err := treated.PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Zero(t, stillFrames(plan))
	})

	t.Run("should still move the marker along the treated route, from its start to its end", func(t *testing.T) {
		// given
		treated := domain.TreatedTrack{Cleaned: domain.Route{Points: cleaned}, Route: domain.Route{Points: route}}

		// when
		plan, err := treated.PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.InDelta(t, (domain.Route{Points: route}).Length(), plan.Frames[len(plan.Frames)-1].MarkerDistance, 0.001)
		assertMarkerMonotonic(t, plan)
	})
}
