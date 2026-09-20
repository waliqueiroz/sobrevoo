package builddomain

import (
	"math"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

const syntheticEarthRadiusMeters = 6371000.0

// SyntheticRouteBuilder builds routes of known shape for geometric tests. The
// geometry uses spherical geodesics (a destination from an origin, a bearing
// and a distance), never a flat projection, so a shape has the same size
// anywhere on Earth, across the antimeridian and near the poles.
type SyntheticRouteBuilder struct {
	originLat, originLon float64
	length               float64
	position             func(distance float64) (lat, lon float64)
	pointCount           int
	speed                float64
	laps                 int
}

func NewSyntheticRouteBuilder() *SyntheticRouteBuilder {
	b := &SyntheticRouteBuilder{originLat: 0, originLon: 0}
	return b.WithLine(1000, 90)
}

func (b *SyntheticRouteBuilder) WithOrigin(lat, lon float64) *SyntheticRouteBuilder {
	b.originLat, b.originLon = lat, lon
	return b
}

// WithLine makes a straight route of lengthMeters along bearing (degrees
// clockwise from north).
func (b *SyntheticRouteBuilder) WithLine(lengthMeters, bearing float64) *SyntheticRouteBuilder {
	b.length, b.laps = lengthMeters, 0
	b.position = func(d float64) (float64, float64) {
		return destination(b.originLat, b.originLon, bearing, d)
	}
	return b
}

// WithOutAndBack makes a route that goes lengthMeters east and returns along
// the same path.
func (b *SyntheticRouteBuilder) WithOutAndBack(lengthMeters float64) *SyntheticRouteBuilder {
	b.length, b.laps = 2*lengthMeters, 0
	b.position = func(d float64) (float64, float64) {
		if d > lengthMeters {
			d = 2*lengthMeters - d
		}
		return destination(b.originLat, b.originLon, 90, d)
	}
	return b
}

// WithUTurn makes a route that goes lengthMeters east, turns 180° on a small
// semicircle (radius a tenth of the length) and comes back on a parallel
// path.
func (b *SyntheticRouteBuilder) WithUTurn(lengthMeters float64) *SyntheticRouteBuilder {
	radius := lengthMeters / 10
	arc := math.Pi * radius
	b.length, b.laps = 2*lengthMeters+arc, 0
	b.position = func(d float64) (float64, float64) {
		outLat, outLon := destination(b.originLat, b.originLon, 90, lengthMeters)
		centerLat, centerLon := destination(outLat, outLon, 180, radius)
		switch {
		case d <= lengthMeters:
			return destination(b.originLat, b.originLon, 90, d)
		case d <= lengthMeters+arc:
			angle := 0 + 180*(d-lengthMeters)/arc // bearing from the center: north (0°) → east (90°) → south (180°)
			return destination(centerLat, centerLon, angle, radius)
		default:
			startLat, startLon := destination(centerLat, centerLon, 180, radius)
			return destination(startLat, startLon, 270, d-lengthMeters-arc)
		}
	}
	return b
}

// WithCircle makes a route of laps circles of radiusMeters, starting at the
// origin.
func (b *SyntheticRouteBuilder) WithCircle(radiusMeters float64, laps int) *SyntheticRouteBuilder {
	b.length, b.laps = 2*math.Pi*radiusMeters*float64(laps), laps
	b.position = func(d float64) (float64, float64) {
		centerLat, centerLon := destination(b.originLat, b.originLon, 0, radiusMeters)
		angle := 180 + (d/radiusMeters)*180/math.Pi
		return destination(centerLat, centerLon, angle, radiusMeters)
	}
	return b
}

func (b *SyntheticRouteBuilder) WithPointCount(count int) *SyntheticRouteBuilder {
	b.pointCount = count
	return b
}

// WithConstantSpeed gives every point a timestamp, as if the route were
// travelled at metersPerSecond.
func (b *SyntheticRouteBuilder) WithConstantSpeed(metersPerSecond float64) *SyntheticRouteBuilder {
	b.speed = metersPerSecond
	return b
}

func (b *SyntheticRouteBuilder) WithoutTime() *SyntheticRouteBuilder {
	b.speed = 0
	return b
}

func (b *SyntheticRouteBuilder) Build() []domain.TrackPoint {
	count := b.pointCount
	if count == 0 {
		count = 201 // odd, so an out-and-back route samples its turning point
		if b.laps > 0 {
			count = 100 * b.laps
		}
	}

	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	points := make([]domain.TrackPoint, count)
	for i := range points {
		distance := b.length * float64(i) / float64(count-1)
		lat, lon := b.position(distance)

		point := NewTrackPointBuilder().WithLatitude(lat).WithLongitude(lon)
		if b.speed > 0 {
			point.WithTime(start.Add(time.Duration(distance / b.speed * float64(time.Second))))
		}
		points[i] = point.Build()
	}
	return points
}

// destination returns the coordinate reached from (lat, lon) after travelling
// distance meters along bearing (degrees clockwise from north).
func destination(lat, lon, bearing, distance float64) (float64, float64) {
	phi, lambda := lat*math.Pi/180, lon*math.Pi/180
	theta := bearing * math.Pi / 180
	delta := distance / syntheticEarthRadiusMeters

	phi2 := math.Asin(math.Sin(phi)*math.Cos(delta) + math.Cos(phi)*math.Sin(delta)*math.Cos(theta))
	lambda2 := lambda + math.Atan2(math.Sin(theta)*math.Sin(delta)*math.Cos(phi), math.Cos(delta)-math.Sin(phi)*math.Sin(phi2))

	lon2 := math.Mod(lambda2*180/math.Pi+540, 360) - 180
	return phi2 * 180 / math.Pi, lon2
}
