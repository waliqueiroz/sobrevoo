package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

const metersPerDegree = 111320.0

// wrapLongitude brings a longitude into [-180, 180).
func wrapLongitude(lon float64) float64 {
	return math.Mod(math.Mod(lon+180, 360)+360, 360) - 180
}

// planEastwards builds a plan of n frames whose marker starts at (lat, lon)
// and moves eastwards by eastMeters; every frame is distance meters from the
// marker, with the camera to the south of it.
func planEastwards(lat, lon, eastMeters float64, n int, distance float64) domain.CameraPlan {
	frames := make([]domain.CameraFrame, n)
	for i := range frames {
		share := 0.0
		if n > 1 {
			share = float64(i) / float64(n-1)
		}
		markerLon := wrapLongitude(lon + eastMeters*share/(metersPerDegree*math.Cos(lat*math.Pi/180)))
		frames[i] = builddomain.NewCameraFrameBuilder().
			WithIndex(i).
			WithMarkerPosition(lat, markerLon).
			WithCameraPosition(lat-0.5*distance/metersPerDegree, markerLon).
			WithCameraToMarkerDistance(distance).
			Build()
	}
	parameters := builddomain.NewPlanParametersBuilder().
		WithDuration(time.Duration(n) * time.Second / 30).WithFrameRate(30).Build()
	return builddomain.NewCameraPlanBuilder().WithParameters(parameters).WithFrames(frames...).Build()
}

func areaWidth(b domain.BoundingBox) float64 {
	width := b.MaxLongitude - b.MinLongitude
	if b.CrossesAntimeridian {
		width += 360
	}
	return width
}

func Test_CameraPlan_AreaOfInterest(t *testing.T) {
	tuning := builddomain.NewSliceTuningBuilder().Build()

	t.Run("should contain every camera and marker position of every frame", func(t *testing.T) {
		// given
		plan := planEastwards(-23.55, -46.63, 8000, 60, 600)

		// when
		area := plan.AreaOfInterest(tuning)

		// then
		for _, frame := range plan.Frames {
			assert.True(t, area.Contains(frame.MarkerLatitude, frame.MarkerLongitude), "marker of frame %d", frame.Index)
			assert.True(t, area.Contains(frame.CameraLatitude, frame.CameraLongitude), "camera of frame %d", frame.Index)
		}
	})

	t.Run("should extend the marker by the margin times the camera-to-marker distance", func(t *testing.T) {
		// given
		plan := planEastwards(0, 0, 0, 1, 1000)

		// when
		area := plan.AreaOfInterest(tuning)

		// then
		half := 1000 / metersPerDegree
		assert.InDelta(t, -half, area.MinLatitude, 1e-6)
		assert.InDelta(t, half, area.MaxLatitude, 1e-6)
		assert.InDelta(t, -half, area.MinLongitude, 1e-6)
		assert.InDelta(t, half, area.MaxLongitude, 1e-6)
		assert.False(t, area.CrossesAntimeridian)
	})

	t.Run("should scale the margin with the margin factor", func(t *testing.T) {
		// given
		plan := planEastwards(0, 0, 0, 1, 1000)
		doubled := builddomain.NewSliceTuningBuilder().WithMarginFactor(2).Build()

		// when
		area := plan.AreaOfInterest(doubled)

		// then
		assert.InDelta(t, 2000/metersPerDegree, area.MaxLatitude, 1e-6)
	})

	t.Run("should widen the area for the far frames of the opening and the closing", func(t *testing.T) {
		// given
		near := planEastwards(10, 20, 500, 30, 300)
		frames := append([]domain.CameraFrame{}, near.Frames...)
		frames[0] = builddomain.NewCameraFrameBuilder().WithIndex(0).
			WithMarkerPosition(10, 20).WithCameraToMarkerDistance(5000).Build()
		far := builddomain.NewCameraPlanBuilder().WithParameters(near.Parameters).WithFrames(frames...).Build()

		// when
		nearArea := near.AreaOfInterest(tuning)
		farArea := far.AreaOfInterest(tuning)

		// then
		assert.GreaterOrEqual(t, farArea.MaxLatitude-farArea.MinLatitude, 2*5000/metersPerDegree-1e-6)
		assert.Greater(t, farArea.MaxLatitude-farArea.MinLatitude, nearArea.MaxLatitude-nearArea.MinLatitude)
	})

	t.Run("should give a small, continuous area for a plan crossing the antimeridian", func(t *testing.T) {
		// given
		plan := planEastwards(0, 179.99, 2000, 20, 100)

		// when
		area := plan.AreaOfInterest(tuning)

		// then
		assert.True(t, area.CrossesAntimeridian)
		assert.Less(t, areaWidth(area), 0.1)
		assert.True(t, area.Contains(0, 180))
		assert.True(t, area.Contains(0, -180))
		assert.False(t, area.Contains(0, 0))
	})

	t.Run("should keep the longitude finite and within one turn near the poles", func(t *testing.T) {
		// given
		plan := planEastwards(89.9999999, 10, 0, 1, 1000)

		// when
		area := plan.AreaOfInterest(tuning)

		// then
		assert.False(t, math.IsNaN(area.MinLongitude) || math.IsNaN(area.MaxLongitude))
		assert.LessOrEqual(t, areaWidth(area), 360.0)
		assert.LessOrEqual(t, area.MaxLatitude, 90.0)
		assert.GreaterOrEqual(t, area.MinLongitude, -180.0)
		assert.LessOrEqual(t, area.MaxLongitude, 180.0)
	})

	t.Run("should widen the longitude with the latitude", func(t *testing.T) {
		// given
		plan := planEastwards(85, 10, 0, 1, 1000)

		// when
		area := plan.AreaOfInterest(tuning)

		// then
		expectedHalf := 1000 / (metersPerDegree * math.Cos(85*math.Pi/180))
		assert.InDelta(t, 2*expectedHalf, areaWidth(area), 1e-4)
	})

	t.Run("should limit the latitudes to -90 and 90", func(t *testing.T) {
		// given
		north := planEastwards(89.9999, 0, 0, 1, 10000)
		south := planEastwards(-89.9999, 0, 0, 1, 10000)

		// when
		northArea := north.AreaOfInterest(tuning)
		southArea := south.AreaOfInterest(tuning)

		// then
		assert.Equal(t, 90.0, northArea.MaxLatitude)
		assert.Equal(t, -90.0, southArea.MinLatitude)
	})

	t.Run("should round the coordinates to 1e-7 degrees", func(t *testing.T) {
		// given
		plan := planEastwards(-23.5505199, -46.6333094, 8000, 60, 617.3)

		// when
		area := plan.AreaOfInterest(tuning)

		// then
		for _, value := range []float64{area.MinLatitude, area.MaxLatitude, area.MinLongitude, area.MaxLongitude} {
			assert.InDelta(t, math.Round(value*1e7), value*1e7, 1e-6)
		}
	})

	t.Run("should give the same area, in meters, wherever on the planet the same flight is", func(t *testing.T) {
		// given
		equator := planEastwards(0, 20, 3000, 30, 400)
		midLatitude := planEastwards(60, 20, 3000, 30, 400)
		antimeridian := planEastwards(60, 179.98, 3000, 30, 400)

		// when
		equatorArea := equator.AreaOfInterest(tuning)
		midArea := midLatitude.AreaOfInterest(tuning)
		antimeridianArea := antimeridian.AreaOfInterest(tuning)

		// then
		meters := func(area domain.BoundingBox, lat float64) (width, height float64) {
			return areaWidth(area) * metersPerDegree * math.Cos(lat*math.Pi/180), (area.MaxLatitude - area.MinLatitude) * metersPerDegree
		}
		equatorWidth, equatorHeight := meters(equatorArea, 0)
		midWidth, midHeight := meters(midArea, 60)
		antimeridianWidth, antimeridianHeight := meters(antimeridianArea, 60)
		assert.InEpsilon(t, equatorWidth, midWidth, 0.01)
		assert.InEpsilon(t, equatorHeight, midHeight, 0.01)
		assert.InDelta(t, midWidth, antimeridianWidth, 1e-3*midWidth)
		assert.InDelta(t, midHeight, antimeridianHeight, 1e-3*midHeight)
		assert.True(t, antimeridianArea.CrossesAntimeridian)
	})

	t.Run("should be deterministic", func(t *testing.T) {
		// given
		plan := planEastwards(-23.55, -46.63, 8000, 60, 600)
		expected := plan.AreaOfInterest(tuning)

		for i := 0; i < 100; i++ {
			// when
			area := plan.AreaOfInterest(tuning)

			// then
			assert.Equal(t, expected, area)
		}
	})
}
