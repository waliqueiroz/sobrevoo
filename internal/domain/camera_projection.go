package domain

import "math"

// PlanePoint is a position on a LocalPlane, in meters east (X) and north (Y)
// of the plane's center.
type PlanePoint struct {
	X float64
	Y float64
}

// LocalPlane is a tangent plane centered on a track, using the azimuthal
// equidistant projection: distances and bearings measured from the center
// are exact. Working on it makes camera planning behave identically anywhere
// on Earth — across the antimeridian and at high latitudes — with no special
// cases (Constitution Principle IV).
type LocalPlane struct {
	latitude  float64 // radians
	longitude float64 // radians
}

// NewLocalPlane centers a plane on the centroid of the route's points,
// computed as the mean of their unit vectors, which is continuous across the
// antimeridian and near the poles (an arithmetic mean of longitudes is not).
func NewLocalPlane(route Route) LocalPlane {
	var sx, sy, sz float64
	for _, p := range route.Points {
		lat, lon := degreesToRadians(p.Latitude), degreesToRadians(p.Longitude)
		sx += float64(math.Cos(lat) * math.Cos(lon))
		sy += float64(math.Cos(lat) * math.Sin(lon))
		sz += math.Sin(lat)
	}

	norm := math.Sqrt(sx*sx + sy*sy + sz*sz)
	if norm < 1e-12 {
		// Degenerate (antipodal points cancel out): any center is as good.
		return LocalPlane{}
	}

	return LocalPlane{
		latitude:  math.Asin(sz / norm),
		longitude: math.Atan2(sy, sx),
	}
}

// Project converts a coordinate in degrees to a position on the plane.
func (l LocalPlane) Project(lat, lon float64) PlanePoint {
	phi, lambda := degreesToRadians(lat), degreesToRadians(lon)
	deltaLambda := lambda - l.longitude

	// Angular distance from the center, by the haversine formula, which is
	// accurate for very small distances (unlike acos of a cosine).
	sinHalfPhi := math.Sin((phi - l.latitude) / 2)
	sinHalfLambda := math.Sin(deltaLambda / 2)
	hav := float64(sinHalfPhi*sinHalfPhi) + float64(math.Cos(phi)*math.Cos(l.latitude)*sinHalfLambda*sinHalfLambda)
	angle := 2 * math.Asin(math.Min(1, math.Sqrt(hav)))

	// Direction from the center.
	east := math.Cos(phi) * math.Sin(deltaLambda)
	north := float64(math.Cos(l.latitude)*math.Sin(phi)) - float64(math.Sin(l.latitude)*math.Cos(phi)*math.Cos(deltaLambda))
	norm := math.Hypot(east, north)
	if norm < 1e-15 {
		return PlanePoint{}
	}

	distance := earthRadiusMeters * angle
	return PlanePoint{X: distance * east / norm, Y: distance * north / norm}
}

// Unproject converts a position on the plane back to a coordinate in
// degrees, with the longitude in [-180, 180).
func (l LocalPlane) Unproject(p PlanePoint) (lat, lon float64) {
	rho := math.Hypot(p.X, p.Y)
	if rho < 1e-12 {
		return radiansToDegrees(l.latitude), normalizeLongitude(radiansToDegrees(l.longitude))
	}

	angle := rho / earthRadiusMeters
	sinAngle, cosAngle := math.Sin(angle), math.Cos(angle)

	sinPhi := float64(cosAngle*math.Sin(l.latitude)) + float64(p.Y*sinAngle*math.Cos(l.latitude)/rho)
	phi := math.Asin(math.Max(-1, math.Min(1, sinPhi)))
	lambda := l.longitude + math.Atan2(
		p.X*sinAngle,
		float64(rho*math.Cos(l.latitude)*cosAngle)-float64(p.Y*math.Sin(l.latitude)*sinAngle),
	)

	return radiansToDegrees(phi), normalizeLongitude(radiansToDegrees(lambda))
}

func radiansToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

// ProjectRoute projects every point of the route onto the plane, keeping the
// distance travelled at each point.
func (l LocalPlane) ProjectRoute(route Route) PlanarRoute {
	points := make([]PlanePoint, len(route.Points))
	for i, p := range route.Points {
		points[i] = l.Project(p.Latitude, p.Longitude)
	}
	return PlanarRoute{Points: points, Distances: route.Distances()}
}
