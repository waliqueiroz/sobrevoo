package domain

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var overlayTuning = RenderTuning{TrailLiftMeters: 0.3, DepthBiasMeters: 1, DepthBiasRatio: 0.002}

// overlayOn is an overlay on an empty image of 200 × 120 pixels, over terrain
// nowhere (so nothing hides anything): a camera 100 m above the plane looking
// straight down, with its top to the north.
func overlayOn(resolution Resolution, cam camera) overlay {
	depth := make([]float32, resolution.Pixels())
	for i := range depth {
		depth[i] = float32(math.Inf(1))
	}
	return overlay{image: NewFrameImage(resolution), camera: cam, depth: depth, tuning: overlayTuning}
}

func lookingDown(resolution Resolution) camera { return newCamera(0, 90, 100, resolution, 45) }

// changed lists the pixels that are not the background.
func changed(image FrameImage) [][2]int {
	var pixels [][2]int
	for y := 0; y < image.Resolution.Height; y++ {
		for x := 0; x < image.Resolution.Width; x++ {
			if image.At(x, y) != BackgroundColor {
				pixels = append(pixels, [2]int{x, y})
			}
		}
	}
	return pixels
}

func Test_overlay_drawMarker(t *testing.T) {
	resolution := Resolution{Width: 200, Height: 120}

	t.Run("should be centered within a pixel of where the point projects, for many cameras", func(t *testing.T) {
		for _, view := range [][3]float64{{0, 90, 100}, {40, 45, 300}, {217.5, 30, 500}, {300, 60, 150}} {
			// given
			cam := newCamera(view[0], view[1], view[2], resolution, 45)
			o := overlayOn(resolution, cam)
			point := [3]float64{cam.position[0] + 20*cam.forward[0] + 3*cam.right[0], cam.position[1] + 20*cam.forward[1] + 3*cam.right[1], cam.position[2] + 20*cam.forward[2] + 3*cam.right[2]}
			px, py, _, ok := cam.project(point[0], point[1], point[2])
			require.True(t, ok)

			// when
			o.drawMarker(point)

			// then
			pixels := changed(o.image)
			require.NotEmpty(t, pixels)
			var sumX, sumY float64
			for _, p := range pixels {
				sumX, sumY = sumX+float64(p[0])+0.5, sumY+float64(p[1])+0.5
			}
			assert.InDelta(t, px, sumX/float64(len(pixels)), 1.0, "view %v", view)
			assert.InDelta(t, py, sumY/float64(len(pixels)), 1.0, "view %v", view)
		}
	})

	t.Run("should be a disc of the radius of the tuning, with the color of the marker at its center and a ring around", func(t *testing.T) {
		// given
		bigger := Resolution{Width: 400, Height: 300}
		o := overlayOn(bigger, lookingDown(bigger))

		// when
		o.drawMarker([3]float64{0, 0, 0})

		// then: the radius is 1.2% of the height (3.6 px), and at least 4
		pixels := changed(o.image)
		radius := math.Max(MarkerMinRadius, MarkerRadiusRatio*300)
		assert.InDelta(t, math.Pi*radius*radius, float64(len(pixels)), 0.25*math.Pi*radius*radius)
		assert.Equal(t, MarkerColor, o.image.At(199, 149))
		edge := o.image.At(200+int(radius)-1, 150)
		assert.Greater(t, edge.G, MarkerColor.G, "the ring is whiter than the fill")
	})

	t.Run("should be hidden by terrain that is nearer than it is", func(t *testing.T) {
		// given: the marker is 100 m from the camera; the terrain at every pixel is 50 m from it
		o := overlayOn(resolution, lookingDown(resolution))
		for i := range o.depth {
			o.depth[i] = 50
		}

		// when
		o.drawMarker([3]float64{0, 0, 0})

		// then
		assert.Empty(t, changed(o.image))
	})

	t.Run("should show over terrain that is farther, or as far within the slack", func(t *testing.T) {
		// given
		farther := overlayOn(resolution, lookingDown(resolution))
		same := overlayOn(resolution, lookingDown(resolution))
		for i := range farther.depth {
			farther.depth[i], same.depth[i] = 200, 100
		}

		// when
		farther.drawMarker([3]float64{0, 0, 0})
		same.drawMarker([3]float64{0, 0, 0.6}) // 0.6 m nearer than the terrain is far: inside the bias

		// then
		assert.NotEmpty(t, changed(farther.image))
		assert.NotEmpty(t, changed(same.image))
	})

	t.Run("should be drawn whole when the terrain that is nearer is only under part of it", func(t *testing.T) {
		// given: the terrain is near on the left half of the image, but not at the center of the marker
		o := overlayOn(resolution, lookingDown(resolution))
		for y := 0; y < 120; y++ {
			for x := 0; x < 100; x++ {
				o.depth[y*200+x] = 10
			}
		}

		// when
		o.drawMarker([3]float64{0, 0, 0})

		// then: the disc, centered on the middle of the image, is over both halves
		var left, right int
		for _, p := range changed(o.image) {
			if p[0] < 100 {
				left++
			} else {
				right++
			}
		}
		assert.Greater(t, left, 0)
		assert.Greater(t, right, 0)
		assert.InDelta(t, left, right, 3)
	})

	t.Run("should be hidden when the terrain at its center is nearer, though it is not at its edge", func(t *testing.T) {
		// given: the terrain is near at the pixel of the center only
		o := overlayOn(resolution, lookingDown(resolution))
		o.depth[60*200+100] = 10

		// when
		o.drawMarker([3]float64{0, 0, 0})

		// then
		assert.Empty(t, changed(o.image))
	})

	t.Run("should not be drawn when the point is behind the camera", func(t *testing.T) {
		// given
		o := overlayOn(resolution, lookingDown(resolution))

		// when
		o.drawMarker([3]float64{0, 0, 200})

		// then
		assert.Empty(t, changed(o.image))
	})
}

func Test_overlay_drawTrail(t *testing.T) {
	resolution := Resolution{Width: 200, Height: 120}
	cam := lookingDown(resolution)

	// meters on the ground per pixel at the center of the image: 100 m away, so f_px = 60 / tan(22.5°)
	perPixel := 100 / cam.focal

	t.Run("should draw nothing for a trail that has not moved", func(t *testing.T) {
		// given
		o := overlayOn(resolution, cam)

		// when
		o.drawTrail([][3]float64{{0, 0, 0}})
		o.drawTrail([][3]float64{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}})

		// then
		assert.Empty(t, changed(o.image))
	})

	t.Run("should draw a line of the width of the tuning, with the color of the trail in the middle and a casing at the edges", func(t *testing.T) {
		// given: a line 100 pixels long along the middle row
		bigger := Resolution{Width: 400, Height: 400}
		big := lookingDown(bigger)
		o := overlayOn(bigger, big)
		step := 100 * 100 / big.focal

		// when
		o.drawTrail([][3]float64{{-step / 2, 0, 0}, {step / 2, 0, 0}})

		// then: 0.5% of 400 is 2 px, the least it has, so the core covers the two rows on the axis and the
		// casing adds a pixel at each side
		assert.Equal(t, TrailColor, o.image.At(200, 199))
		assert.Equal(t, TrailColor, o.image.At(200, 200))
		assert.Equal(t, TrailCasingColor, o.image.At(200, 198))
		assert.Equal(t, TrailCasingColor, o.image.At(200, 201))
		assert.Equal(t, BackgroundColor, o.image.At(200, 196))
		assert.Equal(t, BackgroundColor, o.image.At(200, 203))
	})

	t.Run("should be wider on a taller image", func(t *testing.T) {
		// given
		small := Resolution{Width: 200, Height: 200}
		large := Resolution{Width: 800, Height: 800}
		count := func(res Resolution) int {
			c := lookingDown(res)
			o := overlayOn(res, c)
			step := 40 * 100 / c.focal
			o.drawTrail([][3]float64{{-step / 2, 0, 0}, {step / 2, 0, 0}})
			var core int
			for y := 0; y < res.Height; y++ {
				if o.image.At(res.Width/2, y) == TrailColor {
					core++
				}
			}
			return core
		}

		// when / then
		assert.Greater(t, count(large), count(small))
	})

	t.Run("should have the trail of the frame before it and the stretch since, and nothing ahead", func(t *testing.T) {
		// given: points along a line, to the east
		points := [][3]float64{}
		for i := 0; i < 20; i++ {
			points = append(points, [3]float64{float64(i) * 4 * perPixel, 0, 0})
		}
		before, after := overlayOn(resolution, cam), overlayOn(resolution, cam)

		// when
		before.drawTrail(points[:10])
		after.drawTrail(points[:15])

		// then
		inBefore, inAfter := map[[2]int]bool{}, map[[2]int]bool{}
		for _, p := range changed(before.image) {
			inBefore[p] = true
		}
		for _, p := range changed(after.image) {
			inAfter[p] = true
		}
		require.NotEmpty(t, inBefore)
		for p := range inBefore {
			assert.True(t, inAfter[p], "pixel %v is in the shorter trail but not in the longer", p)
		}
		assert.Greater(t, len(inAfter), len(inBefore))
		for p := range inAfter {
			assert.Less(t, float64(p[0])-100, 14*4+4.0, "nothing past the last point")
		}
	})

	t.Run("should draw the trail under the marker", func(t *testing.T) {
		// given
		o := overlayOn(resolution, cam)

		// when
		o.drawTrail([][3]float64{{-20 * perPixel, 0, 0}, {20 * perPixel, 0, 0}})
		o.drawMarker([3]float64{0, 0, 0})

		// then
		assert.Equal(t, MarkerColor, o.image.At(100, 60))
	})

	t.Run("should hide the part of the trail that terrain nearer to the camera hides", func(t *testing.T) {
		// given: the terrain is near in the right half
		o := overlayOn(resolution, cam)
		for y := 0; y < 120; y++ {
			for x := 100; x < 200; x++ {
				o.depth[y*200+x] = 10
			}
		}

		// when
		o.drawTrail([][3]float64{{-40 * perPixel, 0, 0}, {40 * perPixel, 0, 0}})

		// then
		pixels := changed(o.image)
		require.NotEmpty(t, pixels)
		for _, p := range pixels {
			assert.Less(t, p[0], 100)
		}
	})

	t.Run("should not draw a stretch that is behind the camera", func(t *testing.T) {
		// given
		o := overlayOn(resolution, cam)

		// when
		o.drawTrail([][3]float64{{0, 0, 200}, {30, 0, 300}})

		// then
		assert.Empty(t, changed(o.image))
	})

	t.Run("should cut a stretch that goes through the plane in front of the camera, and draw the part ahead", func(t *testing.T) {
		// given: one end is 50 m below the camera and the other 50 m above it
		o := overlayOn(resolution, cam)

		// when
		o.drawTrail([][3]float64{{5 * perPixel, 0, 50}, {5 * perPixel, 0, 150}})

		// then: nothing panics, and what is drawn is in front (the far end is dropped)
		assert.NotEmpty(t, changed(o.image))
	})
}
