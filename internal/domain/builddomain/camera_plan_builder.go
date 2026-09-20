package builddomain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type CameraPlanBuilder struct {
	parameters    domain.PlanParameters
	durationMode  domain.DurationMode
	timeReference domain.TimeReference
	reason        string
	frames        []domain.CameraFrame
	spans         []domain.SmoothedSpan
}

func NewCameraPlanBuilder() *CameraPlanBuilder {
	return &CameraPlanBuilder{
		parameters:    NewPlanParametersBuilder().WithDuration(100 * time.Millisecond).WithFrameRate(30).Build(),
		durationMode:  domain.DurationModeExplicit,
		timeReference: domain.TimeReferenceClock,
		frames: []domain.CameraFrame{
			NewCameraFrameBuilder().WithIndex(0).WithPhase(domain.PhaseOpening).Build(),
			NewCameraFrameBuilder().WithIndex(1).WithPhase(domain.PhaseFollowing).Build(),
			NewCameraFrameBuilder().WithIndex(2).WithPhase(domain.PhaseClosing).Build(),
		},
	}
}

func (b *CameraPlanBuilder) WithParameters(parameters domain.PlanParameters) *CameraPlanBuilder {
	b.parameters = parameters
	return b
}

func (b *CameraPlanBuilder) WithDurationMode(mode domain.DurationMode) *CameraPlanBuilder {
	b.durationMode = mode
	return b
}

func (b *CameraPlanBuilder) WithTimeReference(reference domain.TimeReference, reason string) *CameraPlanBuilder {
	b.timeReference = reference
	b.reason = reason
	return b
}

func (b *CameraPlanBuilder) WithFrames(frames ...domain.CameraFrame) *CameraPlanBuilder {
	b.frames = frames
	return b
}

func (b *CameraPlanBuilder) WithSmoothedSpans(spans ...domain.SmoothedSpan) *CameraPlanBuilder {
	b.spans = spans
	return b
}

func (b *CameraPlanBuilder) Build() domain.CameraPlan {
	return domain.NewCameraPlan(b.parameters, b.durationMode, b.timeReference, b.reason, b.frames, b.spans)
}
