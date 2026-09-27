package domain

import "math"

// The way a frame is drawn is described in specs/005-frame-rendering/
// research.md. Every product that goes into a sum is wrapped in float64(...),
// so the compiler cannot fuse it into a multiply-add (which some processors
// have and others do not, and which would change the last bits): a frame must be
// the same wherever it is drawn.

// minimumCosLatitudeOfFrame keeps the width of a degree of longitude finite
// near the poles, as minimumCosLatitude does for the area of a plan.
const minimumCosLatitudeOfFrame = 1e-6

// nearPlane is how far in front of the camera, in meters, something has to be
// to be drawn.
const nearPlane = 0.1

// framePlane is the local plane of one frame: its origin is where the camera
// is, x grows to the east and y to the north, in meters. A degree of latitude
// is π/180 · R and a degree of longitude the same times the cosine of the
// latitude of the camera, so the plane is an affine image of the graticule:
// finding the cell of a grid or a tile at (x, y) takes no trigonometry. It is
// the same anywhere on Earth, and it has no seam at the antimeridian, since
// longitudes are taken relative to the camera's.
type framePlane struct {
	latitude, longitude float64
	metersPerDegreeLat  float64
	metersPerDegreeLon  float64
}

func newFramePlane(latitude, longitude float64) framePlane {
	perDegree := math.Pi / 180 * earthRadiusMeters
	cosLatitude := math.Max(math.Cos(degreesToRadians(latitude)), minimumCosLatitudeOfFrame)

	return framePlane{
		latitude:           latitude,
		longitude:          longitude,
		metersPerDegreeLat: perDegree,
		metersPerDegreeLon: float64(perDegree * cosLatitude),
	}
}

// eastwards is how many degrees of longitude lon is to the east of the
// camera's, in [-180, 180).
func (p framePlane) eastwards(lon float64) float64 {
	delta := math.Mod(lon-p.longitude, 360)
	switch {
	case delta >= 180:
		delta -= 360
	case delta < -180:
		delta += 360
	}
	return delta
}

// toPlane is the position, in meters, of a coordinate in degrees.
func (p framePlane) toPlane(lat, lon float64) (x, y float64) {
	return float64(p.eastwards(lon) * p.metersPerDegreeLon), float64((lat - p.latitude) * p.metersPerDegreeLat)
}

// toGeo is the coordinate, in degrees, of a position in meters; the longitude
// is in [-180, 180).
func (p framePlane) toGeo(x, y float64) (lat, lon float64) {
	return p.latitude + y/p.metersPerDegreeLat, normalizeLongitude(p.longitude + x/p.metersPerDegreeLon)
}

// camera is the point of view of a frame: where it is, the three directions of
// its base — forward, to the right of the image and up on it — and the focal
// length, in pixels, of a pinhole with square pixels.
type camera struct {
	position           [3]float64
	forward, right, up [3]float64
	focal              float64
	width, height      float64
}

// newCamera looks along a heading (degrees clockwise from north), tilted down
// by tilt degrees, from a height above the plane's z = 0. The vertical field of
// view is fovDegrees whatever the resolution, so a wider image only sees more
// to the sides. The horizon is always level: there is no roll.
func newCamera(heading, tilt, height float64, resolution Resolution, fovDegrees float64) camera {
	h, t := degreesToRadians(heading), degreesToRadians(tilt)
	sinH, cosH, sinT, cosT := math.Sin(h), math.Cos(h), math.Sin(t), math.Cos(t)

	return camera{
		position: [3]float64{0, 0, height},
		forward:  [3]float64{float64(sinH * cosT), float64(cosH * cosT), -sinT},
		right:    [3]float64{cosH, -sinH, 0},
		up:       [3]float64{float64(sinH * sinT), float64(cosH * sinT), cosT},
		focal:    float64(resolution.Height) / 2 / math.Tan(degreesToRadians(fovDegrees)/2),
		width:    float64(resolution.Width),
		height:   float64(resolution.Height),
	}
}

// ray is the direction, of length 1, of the ray through the center of the pixel
// at column i and row j, counted from the top left corner.
func (c camera) ray(i, j int) (dx, dy, dz float64) {
	right := (float64(i) + 0.5 - c.width/2) / c.focal
	up := -(float64(j) + 0.5 - c.height/2) / c.focal

	dx = c.forward[0] + float64(right*c.right[0]) + float64(up*c.up[0])
	dy = c.forward[1] + float64(right*c.right[1]) + float64(up*c.up[1])
	dz = c.forward[2] + float64(right*c.right[2]) + float64(up*c.up[2])

	length := math.Sqrt(float64(dx*dx) + float64(dy*dy) + float64(dz*dz))
	return dx / length, dy / length, dz / length
}

// toCamera is the position of a point in the base of the camera: to the right,
// up and forward of it.
func (c camera) toCamera(x, y, z float64) (right, up, forward float64) {
	dx, dy, dz := x-c.position[0], y-c.position[1], z-c.position[2]

	right = float64(dx*c.right[0]) + float64(dy*c.right[1]) + float64(dz*c.right[2])
	up = float64(dx*c.up[0]) + float64(dy*c.up[1]) + float64(dz*c.up[2])
	forward = float64(dx*c.forward[0]) + float64(dy*c.forward[1]) + float64(dz*c.forward[2])
	return right, up, forward
}

// screen is where a point seen at (right, up, forward) falls on the image, in
// pixels from the top left corner.
func (c camera) screen(right, up, forward float64) (px, py float64) {
	return c.width/2 + float64(right/forward*c.focal), c.height/2 - float64(up/forward*c.focal)
}

// project is where a point falls on the image, and how far it is from the
// camera; ok is false for a point behind the camera or too close to it.
func (c camera) project(x, y, z float64) (px, py, distance float64, ok bool) {
	right, up, forward := c.toCamera(x, y, z)
	if forward < nearPlane {
		return 0, 0, 0, false
	}

	px, py = c.screen(right, up, forward)
	distance = math.Sqrt(float64(right*right) + float64(up*up) + float64(forward*forward))
	return px, py, distance, true
}

// frameTarget is the point the camera of a frame observes, on the plane of the
// frame. The camera plan puts the camera behind the point it observes, along
// the heading, at a height above it that the frame records: the point is
// recovered from the frame as altitude / tan(tilt) ahead of the camera. Under
// MinTiltForTargetDegrees that cannot be done and the marker stands for it.
func frameTarget(plane framePlane, frame CameraFrame, tuning RenderTuning) (x, y float64) {
	if frame.Tilt < tuning.MinTiltForTargetDegrees {
		return plane.toPlane(frame.MarkerLatitude, frame.MarkerLongitude)
	}

	horizontal := frame.CameraAltitude / math.Tan(degreesToRadians(frame.Tilt))
	heading := degreesToRadians(frame.Heading)
	return float64(horizontal * math.Sin(heading)), float64(horizontal * math.Cos(heading))
}

// groundSource says how high the terrain is at a position of the plane.
type groundSource interface {
	groundAt(x, y float64) float64
}

// cameraHeight is the height of the camera of a frame above the datum of the
// elevation data. The plan records the altitude above the terrain under the
// observed point, so it is that terrain plus the altitude — but never less than
// MinCameraClearanceMeters above the terrain under the camera itself, which
// would otherwise put it inside a hill it stands behind.
func cameraHeight(ground groundSource, plane framePlane, frame CameraFrame, tuning RenderTuning) float64 {
	targetX, targetY := frameTarget(plane, frame, tuning)
	height := ground.groundAt(targetX, targetY) + frame.CameraAltitude

	return math.Max(height, ground.groundAt(0, 0)+tuning.MinCameraClearanceMeters)
}
