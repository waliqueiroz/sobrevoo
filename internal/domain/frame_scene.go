package domain

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
)

// bandRows is how many rows of the image a goroutine takes at a time.
const bandRows = 16

// Scene is the terrain and the map of a slice, ready to be drawn from any point
// of view: the surfaces of its elevation grids, with the holes filled, and the
// imagery of its base maps. It is built once for all the frames of a drawing, and
// it is safe for use by many goroutines at once: all that changes in it is the
// cache of decoded tiles, which changes no result.
type Scene struct {
	surfaces      []*surface
	imagery       *imagery
	tuning        RenderTuning
	appearance    Appearance
	overlayConfig OverlayConfig

	// face rasterizes the screen overlay's text (frame_screen_overlay.go),
	// built once here and reused by every Render of this Scene — the same
	// pixel size is used for every frame of one drawing, so every glyph it
	// ever rasterizes stays in its cache for the rest of the execution.
	face *vectorFace
}

// NewScene prepares a slice for drawing. It fails with ErrNoElevationData when
// no sample of the slice has a value: there would be no terrain to draw.
func NewScene(slice GeoSlice, decoder TileDecoder, tuning RenderTuning, appearance Appearance, overlay OverlayConfig) (*Scene, error) {
	lowest := math.Inf(1)
	for _, grid := range slice.Elevation {
		if minimum, _, ok := grid.Range(); ok {
			lowest = math.Min(lowest, minimum)
		}
	}
	if math.IsInf(lowest, 1) {
		return nil, fmt.Errorf("%w: no elevation sample has a value", ErrNoElevationData)
	}

	scene := &Scene{
		imagery:       newImagery(slice.TileSets, decoder, tuning.TileCacheBytes, appearance.BackgroundColor),
		tuning:        tuning,
		appearance:    appearance,
		overlayConfig: overlay,
		face:          newVectorFace(),
	}
	for _, grid := range slice.Elevation {
		scene.surfaces = append(scene.surfaces, newSurface(grid, lowest))
	}
	return scene, nil
}

// place puts the terrain on the plane of a frame.
func (s *Scene) place(plane framePlane) terrain {
	ground := make(terrain, len(s.surfaces))
	for i, surface := range s.surfaces {
		ground[i] = surface.place(plane)
	}
	return ground
}

// Render draws the frame index of plan, at the resolution, as the camera of the
// frame sees the slice: the terrain in perspective, dressed in the map, the trail
// followed so far — the marker of every frame up to this one — and the marker of
// this frame (research.md items 1 to 10). The same plan, slice and resolution
// always give the same image, however many goroutines draw it.
//
// The drawing stops when ctx is done and returns its error; a tile that is not
// an image fails it with ErrSliceFileInvalid.
func (s *Scene) Render(ctx context.Context, plan CameraPlan, index int, resolution Resolution) (FrameImage, FrameStats, error) {
	if index < 0 || index >= len(plan.Frames) {
		return FrameImage{}, FrameStats{}, fmt.Errorf("%w: frame %d, the plan has frames 0 to %d", ErrFrameOutOfRange, index, len(plan.Frames)-1)
	}
	if err := ctx.Err(); err != nil {
		return FrameImage{}, FrameStats{}, err
	}

	frame := plan.Frames[index]
	plane := newFramePlane(frame.CameraLatitude, frame.CameraLongitude)
	ground := s.place(plane)
	height := cameraHeight(ground, plane, frame, s.tuning)
	cam := newCamera(frame.Heading, frame.Tilt, height, resolution, s.tuning.VerticalFOVDegrees)

	image := NewFrameImage(resolution, s.appearance.BackgroundColor)
	depth := make([]float32, resolution.Pixels())

	stats, err := s.drawTerrain(ctx, ground, plane, cam, image, depth)
	if err != nil {
		return FrameImage{}, FrameStats{}, err
	}

	over := overlay{image: image, camera: cam, depth: depth, tuning: s.tuning, appearance: s.appearance}
	trail := make([][3]float64, index+1)
	for i := range trail {
		trail[i] = s.onGround(ground, plane, plan.Frames[i].MarkerLatitude, plan.Frames[i].MarkerLongitude)
	}
	over.drawTrail(trail)
	over.drawMarker(trail[index])

	screenOverlay{image: image, config: s.overlayConfig, appearance: s.appearance, face: s.face}.draw(plan, index)

	return image, stats, nil
}

// onGround is a position of the plane on the terrain, lifted from it a little.
func (s *Scene) onGround(ground terrain, plane framePlane, lat, lon float64) [3]float64 {
	x, y := plane.toPlane(lat, lon)
	return [3]float64{x, y, ground.groundAt(x, y) + s.tuning.TrailLiftMeters}
}

// drawTerrain draws the terrain, a ray for each pixel, in bands of rows taken by
// the goroutines as they finish. A pixel depends on nothing but itself, so the
// number of goroutines changes nothing in the image, nor in what it says about
// the terrain it shows: a frame has a hole of the map, or of the elevation, when
// some pixel of terrain does — an OR, which does not depend on the order.
func (s *Scene) drawTerrain(ctx context.Context, ground terrain, plane framePlane, cam camera, image FrameImage, depth []float32) (FrameStats, error) {
	width, height := image.Resolution.Width, image.Resolution.Height
	bands := (height + bandRows - 1) / bandRows
	workers := max(min(s.tuning.Workers, bands), 1)

	var (
		next                   atomic.Int32
		failure                atomic.Pointer[error]
		finished               sync.WaitGroup
		mapHole, elevationHole atomic.Bool
	)
	fail := func(err error) { failure.CompareAndSwap(nil, &err) }

	for w := 0; w < workers; w++ {
		finished.Add(1)
		go func() {
			defer finished.Done()
			sampler := s.imagery.newSampler(1 / cam.focal)

			for failure.Load() == nil {
				band := int(next.Add(1)) - 1
				if band >= bands {
					return
				}
				if err := ctx.Err(); err != nil {
					fail(err)
					return
				}

				for y := band * bandRows; y < min((band+1)*bandRows, height); y++ {
					for x := 0; x < width; x++ {
						state, seen, err := s.drawPixel(ground, plane, cam, sampler, image, depth, x, y)
						if err != nil {
							fail(err)
							return
						}
						if seen && state == stateNoMap {
							mapHole.Store(true)
						}
						if seen && state == stateNoElevation {
							elevationHole.Store(true)
						}
					}
				}
			}
		}()
	}
	finished.Wait()

	if err := failure.Load(); err != nil {
		return FrameStats{}, *err
	}
	return FrameStats{MapHole: mapHole.Load(), ElevationHole: elevationHole.Load()}, ctx.Err()
}

// drawPixel draws the pixel (x, y): the ray of the camera through it, the first
// terrain it meets, and what the terrain shows there — the color of the map,
// lightened or darkened by the terrain's fixed directional light according to
// the surface's own normal at that point (016-terrain-lighting); the hatch of
// terrain with no map image; or the checkerboard of terrain over a cell with
// no elevation, which comes first and is never lit, like the hatch. seen is
// false for a ray that meets nothing, which leaves the background: that is no
// hole.
func (s *Scene) drawPixel(ground terrain, plane framePlane, cam camera, sampler *sampler, image FrameImage, depth []float32, x, y int) (state pixelState, seen bool, err error) {
	dx, dy, dz := cam.ray(x, y)
	hit, ok := ground.trace(cam.position[0], cam.position[1], cam.position[2], dx, dy, dz)
	if !ok {
		depth[y*image.Resolution.Width+x] = float32(math.Inf(1))
		return stateImage, false, nil
	}
	depth[y*image.Resolution.Width+x] = float32(hit.t)

	lat, lon := plane.toGeo(hit.x, hit.y)
	color, state, err := sampler.color(lat, lon, hit.t, math.Abs(dz))
	if err != nil {
		return stateImage, true, err
	}
	if ground[hit.grid].isHole(hit.row, hit.col) {
		state = stateNoElevation
	}

	switch state {
	case stateImage:
		// footprint is how much ground, in meters, this pixel actually covers at the hit's
		// distance — the same "how many ground units a screen pixel covers here" sampler.color
		// already computes for the texture's level of detail (research.md item 6), with the same
		// slant correction (a ray that grazes the ground covers more of it per pixel).
		footprint := float64(sampler.pixelAngle*hit.t) / math.Sqrt(math.Max(math.Abs(dz), 0.1))
		nx, ny, nz := ground[hit.grid].normalAt(hit.x, hit.y, footprint)
		factor := terrainLightFactor(nx, ny, nz)
		image.Set(x, y, RGB{
			rounded(float64(float64(color.R) * factor)),
			rounded(float64(float64(color.G) * factor)),
			rounded(float64(float64(color.B) * factor)),
		})
	case stateNoMap:
		image.Set(x, y, NoMapColors[hatchTone(x, y)])
	case stateNoElevation:
		image.Set(x, y, NoElevationColors[checkerTone(x, y)])
	}
	return state, true, nil
}

// hatchTone is which of the two tones of the hatch a pixel has: it alternates
// along the diagonals, every half of PatternPeriod.
func hatchTone(x, y int) int {
	if (x+y)%PatternPeriod < PatternPeriod/2 {
		return 0
	}
	return 1
}

// checkerTone is which of the two tones of the checkerboard a pixel has: squares
// of PatternPeriod pixels.
func checkerTone(x, y int) int {
	return (x/PatternPeriod + y/PatternPeriod) % 2
}
