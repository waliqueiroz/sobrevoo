package builddomain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type CameraFrameBuilder struct {
	frame domain.CameraFrame
}

func NewCameraFrameBuilder() *CameraFrameBuilder {
	return &CameraFrameBuilder{
		frame: domain.CameraFrame{
			Index:                  0,
			Phase:                  domain.PhaseFollowing,
			CameraLatitude:         40.4168,
			CameraLongitude:        -3.7038,
			CameraAltitude:         100,
			Heading:                90,
			Tilt:                   45,
			MarkerLatitude:         40.4170,
			MarkerLongitude:        -3.7030,
			MarkerDistance:         0,
			CameraToMarkerDistance: 141.421,
		},
	}
}

func (b *CameraFrameBuilder) WithIndex(index int) *CameraFrameBuilder {
	b.frame.Index = index
	return b
}

func (b *CameraFrameBuilder) WithTime(t time.Duration) *CameraFrameBuilder {
	b.frame.Time = t
	return b
}

func (b *CameraFrameBuilder) WithPhase(phase domain.Phase) *CameraFrameBuilder {
	b.frame.Phase = phase
	return b
}

func (b *CameraFrameBuilder) WithCameraAltitude(altitude float64) *CameraFrameBuilder {
	b.frame.CameraAltitude = altitude
	return b
}

func (b *CameraFrameBuilder) WithHeading(heading float64) *CameraFrameBuilder {
	b.frame.Heading = heading
	return b
}

func (b *CameraFrameBuilder) WithTilt(tilt float64) *CameraFrameBuilder {
	b.frame.Tilt = tilt
	return b
}

func (b *CameraFrameBuilder) WithMarkerDistance(distance float64) *CameraFrameBuilder {
	b.frame.MarkerDistance = distance
	return b
}

func (b *CameraFrameBuilder) WithCameraToMarkerDistance(distance float64) *CameraFrameBuilder {
	b.frame.CameraToMarkerDistance = distance
	return b
}

func (b *CameraFrameBuilder) Build() domain.CameraFrame {
	return b.frame
}
