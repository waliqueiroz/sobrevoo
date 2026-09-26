package domain

import (
	"fmt"
	"math"
)

// Coordinate is a valid position on Earth, in decimal degrees.
type Coordinate struct {
	// Latitude is in [-90, 90]; Longitude in [-180, 180), where 180 and -180
	// — the same meridian — are both -180.
	Latitude, Longitude float64
}

// NewCoordinate checks a latitude and a longitude and builds the coordinate.
// A value that is not a finite number in range fails with
// ErrInvalidCoordinate, saying the value and the range.
func NewCoordinate(latitude, longitude float64) (Coordinate, error) {
	if !validLatitude(latitude) {
		return Coordinate{}, fmt.Errorf("%w: latitude %g is out of range, must be between -90 and 90", ErrInvalidCoordinate, latitude)
	}
	if !validLongitude(longitude) || math.IsInf(longitude, 0) {
		return Coordinate{}, fmt.Errorf("%w: longitude %g is out of range, must be between -180 and 180", ErrInvalidCoordinate, longitude)
	}

	if longitude == 180 {
		longitude = -180
	}
	return Coordinate{Latitude: latitude, Longitude: longitude}, nil
}
