package domain

import "math"

// overlay draws what is on top of the terrain in a frame: the trail already
// followed and the marker of the activity. Both are drawn on the image of the
// terrain and are hidden by the terrain that is nearer to the camera than they
// are, as the terrain hides itself (research.md item 9).
type overlay struct {
	image  FrameImage
	camera camera

	// depth is how far the terrain is from the camera at each pixel, in meters;
	// +Inf where there is none.
	depth  []float32
	tuning RenderTuning

	// appearance is the color and the size of the trail and the marker this
	// overlay draws with; TrailCasingColor, MarkerRingColor, MarkerRingRatio
	// and MarkerRingMin stay fixed, not part of it.
	appearance Appearance
}

// visible says whether something at distance meters from the camera shows at
// pixel (x, y): it does unless the terrain there is nearer, with some slack
// (DepthBiasMeters plus a share of the distance) for the height the things are
// lifted from the ground.
func (o overlay) visible(x, y int, distance float64) bool {
	terrain := float64(o.depth[y*o.image.Resolution.Width+x])
	if math.IsInf(terrain, 1) {
		return true
	}
	return distance <= terrain+o.tuning.DepthBiasMeters+float64(o.tuning.DepthBiasRatio*terrain)
}

// blend paints the pixel (x, y) with a color, over what it has, by a coverage
// from 0 to 1.
func (o overlay) blend(x, y int, c RGB, coverage float64) {
	current := o.image.At(x, y)
	mix := func(old, painted uint8) uint8 {
		return rounded(float64(float64(old)*(1-coverage)) + float64(float64(painted)*coverage))
	}
	o.image.Set(x, y, RGB{mix(current.R, c.R), mix(current.G, c.G), mix(current.B, c.B)})
}

// screenPoint is a point of the trail on the image, and how far it is from the
// camera.
type screenPoint struct {
	x, y, distance float64
}

// cameraPoint is a point of the plane in the base of the camera.
type cameraPoint struct {
	right, up, forward float64
}

func (o overlay) toScreen(p cameraPoint) screenPoint {
	x, y := o.camera.screen(p.right, p.up, p.forward)
	return screenPoint{x: x, y: y, distance: math.Sqrt(float64(p.right*p.right) + float64(p.up*p.up) + float64(p.forward*p.forward))}
}

// drawTrail draws the line through the points (positions on the plane of the
// frame, with their height): a dark casing under the whole line, then the line.
// A stretch behind the camera is not drawn, and one that crosses the plane in
// front of it is cut there. Two points in the same place make no stretch: the
// trail of a frame at the start of the track has nothing to show.
func (o overlay) drawTrail(points [][3]float64) {
	height := float64(o.image.Resolution.Height)
	core := math.Max(TrailMinWidth, float64(o.appearance.TrailWidthRatio*height)) / 2

	var segments [][2]screenPoint
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if a == b {
			continue
		}

		ar, au, af := o.camera.toCamera(a[0], a[1], a[2])
		br, bu, bf := o.camera.toCamera(b[0], b[1], b[2])
		from, to := cameraPoint{ar, au, af}, cameraPoint{br, bu, bf}
		switch {
		case af < nearPlane && bf < nearPlane:
			continue
		case af < nearPlane:
			from = cutAtNearPlane(from, to)
		case bf < nearPlane:
			to = cutAtNearPlane(to, from)
		}
		segments = append(segments, [2]screenPoint{o.toScreen(from), o.toScreen(to)})
	}

	for _, segment := range segments {
		o.drawCapsule(segment[0], segment[1], core+1, TrailCasingColor)
	}
	for _, segment := range segments {
		o.drawCapsule(segment[0], segment[1], core, o.appearance.TrailColor)
	}
}

// cutAtNearPlane moves the point behind the near plane, along the line to the
// other one, onto the plane.
func cutAtNearPlane(behind, ahead cameraPoint) cameraPoint {
	t := (nearPlane - behind.forward) / (ahead.forward - behind.forward)
	return cameraPoint{
		right:   behind.right + float64(t*(ahead.right-behind.right)),
		up:      behind.up + float64(t*(ahead.up-behind.up)),
		forward: nearPlane,
	}
}

// drawCapsule draws a segment with round ends, halfWidth from its axis to each
// side, softening the edge over a pixel.
func (o overlay) drawCapsule(a, b screenPoint, halfWidth float64, c RGB) {
	width, height := o.image.Resolution.Width, o.image.Resolution.Height
	margin := halfWidth + 1

	x0 := max(int(math.Floor(math.Min(a.x, b.x)-margin)), 0)
	x1 := min(int(math.Ceil(math.Max(a.x, b.x)+margin)), width-1)
	y0 := max(int(math.Floor(math.Min(a.y, b.y)-margin)), 0)
	y1 := min(int(math.Ceil(math.Max(a.y, b.y)+margin)), height-1)

	abx, aby := b.x-a.x, b.y-a.y
	length2 := float64(abx*abx) + float64(aby*aby)

	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			along := 0.0
			if length2 > 0 {
				along = math.Min(math.Max((float64((px-a.x)*abx)+float64((py-a.y)*aby))/length2, 0), 1)
			}
			dx, dy := px-(a.x+float64(along*abx)), py-(a.y+float64(along*aby))
			coverage := math.Min(math.Max(halfWidth+0.5-math.Sqrt(float64(dx*dx)+float64(dy*dy)), 0), 1)
			if coverage <= 0 {
				continue
			}

			if o.visible(x, y, a.distance+float64(along*(b.distance-a.distance))) {
				o.blend(x, y, c, coverage)
			}
		}
	}
}

// drawMarker draws the marker: a disc with a ring around it, centered where the
// point falls on the image. Whether terrain hides it is decided once, at the
// pixel of its center: the disc is a symbol of a fixed size on the screen, and
// the ground under its lower half is nearer to the camera than the point it
// marks, so testing each pixel would cut the disc in half.
func (o overlay) drawMarker(point [3]float64) {
	px, py, distance, ok := o.camera.project(point[0], point[1], point[2])
	if !ok {
		return
	}

	width, height := o.image.Resolution.Width, o.image.Resolution.Height
	centerX, centerY := min(max(int(math.Floor(px)), 0), width-1), min(max(int(math.Floor(py)), 0), height-1)
	if !o.visible(centerX, centerY, distance) {
		return
	}

	radius := math.Max(MarkerMinRadius, float64(o.appearance.MarkerRadiusRatio*float64(height)))
	ring := math.Max(MarkerRingMin, float64(MarkerRingRatio*float64(height)))

	x0 := max(int(math.Floor(px-radius-1)), 0)
	x1 := min(int(math.Ceil(px+radius+1)), width-1)
	y0 := max(int(math.Floor(py-radius-1)), 0)
	y1 := min(int(math.Ceil(py+radius+1)), height-1)

	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx, dy := float64(x)+0.5-px, float64(y)+0.5-py
			from := math.Sqrt(float64(dx*dx) + float64(dy*dy))

			coverage := math.Min(math.Max(radius+0.5-from, 0), 1)
			if coverage <= 0 {
				continue
			}

			ringShare := math.Min(math.Max(from-(radius-ring)+0.5, 0), 1)
			mix := func(fill, edge uint8) uint8 {
				return rounded(float64(float64(fill)*(1-ringShare)) + float64(float64(edge)*ringShare))
			}
			o.blend(x, y, RGB{mix(o.appearance.MarkerColor.R, MarkerRingColor.R), mix(o.appearance.MarkerColor.G, MarkerRingColor.G), mix(o.appearance.MarkerColor.B, MarkerRingColor.B)}, coverage)
		}
	}
}
