package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

type TrackBuilder struct {
	track domain.Track
}

func NewTrackBuilder() *TrackBuilder {
	return &TrackBuilder{
		track: domain.Track{
			Format: domain.FormatGPX,
			Points: []domain.TrackPoint{
				NewTrackPointBuilder().Build(),
				NewTrackPointBuilder().WithLatitude(40.4170).WithLongitude(-3.7030).Build(),
			},
		},
	}
}

func (b *TrackBuilder) WithFormat(format domain.Format) *TrackBuilder {
	b.track.Format = format
	return b
}

func (b *TrackBuilder) WithPoints(points ...domain.TrackPoint) *TrackBuilder {
	b.track.Points = points
	return b
}

func (b *TrackBuilder) Build() domain.Track {
	return b.track
}
