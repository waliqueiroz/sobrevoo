package domain

import "math"

// CameraPose is where a camera is, as derived from what it looks at.
type CameraPose struct {
	// Position is the camera's horizontal position on the local plane.
	Position PlanePoint

	// Altitude is the camera's height above the observed point, in meters.
	Altitude float64
}

// CameraView is what a camera looks at and how: the point it observes, the
// horizontal direction it points (degrees clockwise from north), its tilt
// below the horizon (degrees) and its straight-line distance to the point.
type CameraView struct {
	Target      PlanePoint
	Heading     float64
	TiltDegrees float64
	Distance    float64
}

// Pose places the camera behind the target, along the heading: its
// horizontal offset is Distance·cos(tilt) and its altitude Distance·sin(tilt).
func (v CameraView) Pose() CameraPose {
	heading := degreesToRadians(v.Heading)
	tilt := degreesToRadians(v.TiltDegrees)
	horizontal := v.Distance * math.Cos(tilt)

	return CameraPose{
		Position: PlanePoint{
			X: v.Target.X - horizontal*math.Sin(heading),
			Y: v.Target.Y - horizontal*math.Cos(heading),
		},
		Altitude: v.Distance * math.Sin(tilt),
	}
}

// Blend interpolates from v to other for u in [0, 1] with a smoothstep
// (3u² - 2u³), whose derivative is zero at both ends, so the motion starts
// and stops gently. Heading follows the shortest arc, and the distance
// interpolates in log space so zooming is uniform.
func (v CameraView) Blend(other CameraView, u float64) CameraView {
	s := smoothstep(clamp(u, 0, 1))

	delta := math.Mod(other.Heading-v.Heading+180, 360)
	if delta < 0 {
		delta += 360
	}
	delta -= 180

	return CameraView{
		Target: PlanePoint{
			X: v.Target.X + s*(other.Target.X-v.Target.X),
			Y: v.Target.Y + s*(other.Target.Y-v.Target.Y),
		},
		Heading:     normalizeDegrees(v.Heading + s*delta),
		TiltDegrees: v.TiltDegrees + s*(other.TiltDegrees-v.TiltDegrees),
		Distance:    math.Exp(math.Log(v.Distance) + s*(math.Log(other.Distance)-math.Log(v.Distance))),
	}
}

func smoothstep(u float64) float64 {
	return u * u * (3 - 2*u)
}
