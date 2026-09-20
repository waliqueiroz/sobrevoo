package domain

import "math"

// OverviewView returns the view that frames a whole route: it looks at the
// center of the route's bounding box, tilted overviewTilt degrees, pointing
// along heading (the opening and closing pass no rotation: they keep the
// heading of the following phase), from far enough away that every point of
// the route is within half the vertical field of view of the optical axis,
// plus a margin. The distance is never below minDistance, so that the
// overview is always farther than the following distance.
func OverviewView(route []PlanePoint, heading float64, minDistance float64, tuning CameraTuning) CameraView {
	minX, maxX, minY, maxY := route[0].X, route[0].X, route[0].Y, route[0].Y
	for _, p := range route {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	center := PlanePoint{X: (minX + maxX) / 2, Y: (minY + maxY) / 2}

	var radius float64
	for _, p := range route {
		radius = math.Max(radius, math.Hypot(p.X-center.X, p.Y-center.Y))
	}

	halfFOV := degreesToRadians(tuning.OverviewVerticalFOVDegrees) / 2
	framing := tuning.OverviewMargin * radius / math.Tan(halfFOV)

	return CameraView{
		Target:      center,
		Heading:     heading,
		TiltDegrees: tuning.OverviewTiltDegrees,
		Distance:    math.Max(framing, minDistance),
	}
}

// BlendView interpolates from a to b for u in [0, 1] with a smoothstep
// (3u² - 2u³), whose derivative is zero at both ends, so the motion starts
// and stops gently. Heading follows the shortest arc, and the distance
// interpolates in log space so zooming is uniform.
func BlendView(a, b CameraView, u float64) CameraView {
	s := smoothstep(clamp(u, 0, 1))

	delta := math.Mod(b.Heading-a.Heading+180, 360)
	if delta < 0 {
		delta += 360
	}
	delta -= 180

	return CameraView{
		Target: PlanePoint{
			X: a.Target.X + s*(b.Target.X-a.Target.X),
			Y: a.Target.Y + s*(b.Target.Y-a.Target.Y),
		},
		Heading:     normalizeDegrees(a.Heading + s*delta),
		TiltDegrees: a.TiltDegrees + s*(b.TiltDegrees-a.TiltDegrees),
		Distance:    math.Exp(math.Log(a.Distance) + s*(math.Log(b.Distance)-math.Log(a.Distance))),
	}
}

func smoothstep(u float64) float64 {
	return u * u * (3 - 2*u)
}
