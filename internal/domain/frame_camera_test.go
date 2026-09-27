package domain

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const metersPerDegreeOfArc = math.Pi / 180 * earthRadiusMeters

func Test_framePlane(t *testing.T) {
	t.Run("should put the camera at the origin, with x to the east and y to the north", func(t *testing.T) {
		// given
		plane := newFramePlane(0, 10)

		// when
		originX, originY := plane.toPlane(0, 10)
		eastX, eastY := plane.toPlane(0, 11)
		northX, northY := plane.toPlane(1, 10)

		// then
		assert.Equal(t, 0.0, originX)
		assert.Equal(t, 0.0, originY)
		assert.InDelta(t, metersPerDegreeOfArc, eastX, 1e-6)
		assert.InDelta(t, 0.0, eastY, 1e-9)
		assert.InDelta(t, 0.0, northX, 1e-9)
		assert.InDelta(t, metersPerDegreeOfArc, northY, 1e-6)
	})

	t.Run("should shrink a degree of longitude with the cosine of the latitude of the camera", func(t *testing.T) {
		// given
		plane := newFramePlane(60, 0)

		// when
		x, y := plane.toPlane(60, 0.01)

		// then
		assert.InDelta(t, 0.01*metersPerDegreeOfArc*math.Cos(60*math.Pi/180), x, 1e-6)
		assert.InDelta(t, 0.0, y, 1e-9)
	})

	t.Run("should go there and back between degrees and meters", func(t *testing.T) {
		// given
		plane := newFramePlane(-23.5, -46.6)

		// when
		x, y := plane.toPlane(-23.49, -46.58)
		lat, lon := plane.toGeo(x, y)

		// then
		assert.InDelta(t, -23.49, lat, 1e-9)
		assert.InDelta(t, -46.58, lon, 1e-9)
	})

	t.Run("should be continuous across the antimeridian, with no jump", func(t *testing.T) {
		// given
		plane := newFramePlane(0, 179.99)

		// when
		xWest, _ := plane.toPlane(0, 179.995)
		xEast, _ := plane.toPlane(0, -179.995)
		lat, lon := plane.toGeo(xEast, 0)

		// then
		assert.InDelta(t, 0.005*metersPerDegreeOfArc, xWest, 1e-6)
		assert.InDelta(t, 0.015*metersPerDegreeOfArc, xEast, 1e-6)
		assert.InDelta(t, 0.0, lat, 1e-9)
		assert.InDelta(t, -179.995, lon, 1e-9)
	})

	t.Run("should keep everything finite near the pole", func(t *testing.T) {
		// given
		plane := newFramePlane(89.9999999, 10)

		// when
		x, y := plane.toPlane(89.9999999, 11)
		lat, lon := plane.toGeo(x, y)

		// then
		assert.False(t, math.IsNaN(x) || math.IsInf(x, 0))
		assert.False(t, math.IsNaN(y) || math.IsInf(y, 0))
		assert.InDelta(t, 89.9999999, lat, 1e-9)
		assert.InDelta(t, 11.0, lon, 1e-9)
	})
}

func Test_camera(t *testing.T) {
	dot := func(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
	resolution := Resolution{Width: 192, Height: 108}

	t.Run("should have an orthonormal base for any heading and tilt", func(t *testing.T) {
		for _, heading := range []float64{0, 90, 217.5} {
			for _, tilt := range []float64{0, 25, 45, 90} {
				// given / when
				c := newCamera(heading, tilt, 100, resolution, 45)

				// then
				assert.InDelta(t, 1.0, dot(c.forward, c.forward), 1e-12)
				assert.InDelta(t, 1.0, dot(c.right, c.right), 1e-12)
				assert.InDelta(t, 1.0, dot(c.up, c.up), 1e-12)
				assert.InDelta(t, 0.0, dot(c.forward, c.right), 1e-12)
				assert.InDelta(t, 0.0, dot(c.forward, c.up), 1e-12)
				assert.InDelta(t, 0.0, dot(c.right, c.up), 1e-12)
			}
		}
	})

	t.Run("should look at the heading, down by the tilt, with the right of the image to the east when facing north", func(t *testing.T) {
		// given
		c := newCamera(0, 30, 100, resolution, 45)

		// when / then
		assert.InDelta(t, 0.0, c.forward[0], 1e-12)
		assert.InDelta(t, math.Cos(30*math.Pi/180), c.forward[1], 1e-12)
		assert.InDelta(t, -math.Sin(30*math.Pi/180), c.forward[2], 1e-12)
		assert.InDelta(t, 1.0, c.right[0], 1e-12)
		assert.InDelta(t, 0.0, c.right[1], 1e-12)
		assert.Greater(t, c.up[2], 0.0, "the top of the image points up")
	})

	t.Run("should have the heading on top of the image when looking straight down", func(t *testing.T) {
		// given
		c := newCamera(90, 90, 100, resolution, 45)

		// when / then
		assert.InDelta(t, 1.0, c.up[0], 1e-12)
		assert.InDelta(t, 0.0, c.up[1], 1e-12)
		assert.InDelta(t, -1.0, c.forward[2], 1e-12)
	})

	t.Run("should put the camera at the height it was given", func(t *testing.T) {
		// given / when
		c := newCamera(0, 45, 123.5, resolution, 45)

		// then
		assert.Equal(t, 123.5, c.position[2])
		assert.Equal(t, 0.0, c.position[0])
		assert.Equal(t, 0.0, c.position[1])
	})

	t.Run("should project the ray of a pixel back to the center of that pixel", func(t *testing.T) {
		// given
		c := newCamera(217.5, 35, 500, resolution, 45)

		for _, pixel := range [][2]int{{0, 0}, {191, 107}, {96, 54}, {10, 100}, {150, 3}} {
			// when
			dx, dy, dz := c.ray(pixel[0], pixel[1])
			px, py, depth, ok := c.project(c.position[0]+300*dx, c.position[1]+300*dy, c.position[2]+300*dz)

			// then
			require.True(t, ok)
			assert.InDelta(t, float64(pixel[0])+0.5, px, 1e-6)
			assert.InDelta(t, float64(pixel[1])+0.5, py, 1e-6)
			assert.InDelta(t, 300.0, depth, 1e-6)
			assert.InDelta(t, 1.0, dx*dx+dy*dy+dz*dz, 1e-12)
		}
	})

	t.Run("should project the point ahead of the camera to the middle of the image", func(t *testing.T) {
		// given
		c := newCamera(40, 50, 200, resolution, 45)

		// when
		px, py, _, ok := c.project(c.position[0]+100*c.forward[0], c.position[1]+100*c.forward[1], c.position[2]+100*c.forward[2])

		// then
		require.True(t, ok)
		assert.InDelta(t, 96.0, px, 1e-9)
		assert.InDelta(t, 54.0, py, 1e-9)
	})

	t.Run("should see half the vertical field of view above and below the middle, whatever the width", func(t *testing.T) {
		for _, size := range []Resolution{{Width: 192, Height: 108}, {Width: 384, Height: 108}, {Width: 108, Height: 192}} {
			// given
			c := newCamera(0, 45, 100, size, 45)
			half := 45.0 / 2 * math.Pi / 180

			// when
			topX, topY, _, okTop := c.project(
				c.position[0]+10*(c.forward[0]*math.Cos(half)+c.up[0]*math.Sin(half)),
				c.position[1]+10*(c.forward[1]*math.Cos(half)+c.up[1]*math.Sin(half)),
				c.position[2]+10*(c.forward[2]*math.Cos(half)+c.up[2]*math.Sin(half)))
			sideAngle := math.Atan(math.Tan(half) * float64(size.Width) / float64(size.Height))
			sideX, _, _, okSide := c.project(
				c.position[0]+10*(c.forward[0]*math.Cos(sideAngle)+c.right[0]*math.Sin(sideAngle)),
				c.position[1]+10*(c.forward[1]*math.Cos(sideAngle)+c.right[1]*math.Sin(sideAngle)),
				c.position[2]+10*(c.forward[2]*math.Cos(sideAngle)+c.right[2]*math.Sin(sideAngle)))

			// then
			require.True(t, okTop)
			require.True(t, okSide)
			assert.InDelta(t, 0.0, topY, 1e-9, "the top edge is half the field of view up")
			assert.InDelta(t, float64(size.Width)/2, topX, 1e-9)
			assert.InDelta(t, float64(size.Width), sideX, 1e-9, "the right edge of a wider image sees more to the side")
		}
	})

	t.Run("should reject a point behind the camera", func(t *testing.T) {
		// given
		c := newCamera(0, 45, 100, resolution, 45)

		// when
		_, _, _, ok := c.project(c.position[0]-10*c.forward[0], c.position[1]-10*c.forward[1], c.position[2]-10*c.forward[2])

		// then
		assert.False(t, ok)
	})

	t.Run("should have the focal length of half the height over the tangent of half the field of view", func(t *testing.T) {
		// given / when
		c := newCamera(0, 45, 100, resolution, 45)

		// then
		assert.InDelta(t, 54/math.Tan(22.5*math.Pi/180), c.focal, 1e-9)
	})
}

func Test_frameTarget(t *testing.T) {
	tuning := RenderTuning{MinTiltForTargetDegrees: 1}
	plane := newFramePlane(-23.5, -46.6)

	t.Run("should be behind the camera's look, at altitude over tan of tilt along the heading", func(t *testing.T) {
		// given
		frame := CameraFrame{Heading: 90, Tilt: 45, CameraAltitude: 100}

		// when
		x, y := frameTarget(plane, frame, tuning)

		// then
		assert.InDelta(t, 100.0, x, 1e-9)
		assert.InDelta(t, 0.0, y, 1e-9)
	})

	t.Run("should follow the heading to the north for a tilt of 30 degrees", func(t *testing.T) {
		// given
		frame := CameraFrame{Heading: 0, Tilt: 30, CameraAltitude: 100}

		// when
		x, y := frameTarget(plane, frame, tuning)

		// then
		assert.InDelta(t, 0.0, x, 1e-9)
		assert.InDelta(t, 100/math.Tan(30*math.Pi/180), y, 1e-9)
	})

	t.Run("should be the marker when the tilt is too low to recover the target", func(t *testing.T) {
		// given
		frame := CameraFrame{Heading: 0, Tilt: 0.5, CameraAltitude: 0.3, MarkerLatitude: -23.499, MarkerLongitude: -46.6}

		// when
		x, y := frameTarget(plane, frame, tuning)

		// then
		markerX, markerY := plane.toPlane(-23.499, -46.6)
		assert.InDelta(t, markerX, x, 1e-9)
		assert.InDelta(t, markerY, y, 1e-9)
	})
}
