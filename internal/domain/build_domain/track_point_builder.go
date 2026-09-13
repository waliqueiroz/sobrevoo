package build_domain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type TrackPointBuilder struct {
	trackPoint domain.TrackPoint
}

func NewTrackPointBuilder() *TrackPointBuilder {
	return &TrackPointBuilder{
		trackPoint: domain.TrackPoint{
			Latitude:  40.4168,
			Longitude: -3.7038,
		},
	}
}

func (b *TrackPointBuilder) WithLatitude(latitude float64) *TrackPointBuilder {
	b.trackPoint.Latitude = latitude
	return b
}

func (b *TrackPointBuilder) WithLongitude(longitude float64) *TrackPointBuilder {
	b.trackPoint.Longitude = longitude
	return b
}

func (b *TrackPointBuilder) WithElevation(elevation float64) *TrackPointBuilder {
	b.trackPoint.Elevation = &elevation
	return b
}

func (b *TrackPointBuilder) WithoutElevation() *TrackPointBuilder {
	b.trackPoint.Elevation = nil
	return b
}

func (b *TrackPointBuilder) WithTime(t time.Time) *TrackPointBuilder {
	b.trackPoint.Time = &t
	return b
}

func (b *TrackPointBuilder) WithoutTime() *TrackPointBuilder {
	b.trackPoint.Time = nil
	return b
}

func (b *TrackPointBuilder) Build() domain.TrackPoint {
	return b.trackPoint
}
