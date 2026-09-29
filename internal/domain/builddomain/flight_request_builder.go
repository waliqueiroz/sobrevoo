package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// FlightRequestBuilder builds a domain.FlightRequest: valid plan parameters, a
// 1080 × 1920 resolution, medium quality, a "/tmp/flight.mp4" output, no kept
// directory and no overwrite, unless told otherwise.
type FlightRequestBuilder struct {
	request domain.FlightRequest
}

func NewFlightRequestBuilder() *FlightRequestBuilder {
	return &FlightRequestBuilder{
		request: domain.FlightRequest{
			Parameters: NewPlanParametersBuilder().Build(),
			Resolution: domain.Resolution{Width: 1080, Height: 1920},
			Appearance: NewAppearanceBuilder().Build(),
			Overlay:    NewOverlayConfigBuilder().Build(),
			Quality:    domain.VideoQualityMedium,
			Output:     "/tmp/flight.mp4",
		},
	}
}

func (b *FlightRequestBuilder) WithParameters(parameters domain.PlanParameters) *FlightRequestBuilder {
	b.request.Parameters = parameters
	return b
}

func (b *FlightRequestBuilder) WithResolution(resolution domain.Resolution) *FlightRequestBuilder {
	b.request.Resolution = resolution
	return b
}

func (b *FlightRequestBuilder) WithAppearance(appearance domain.Appearance) *FlightRequestBuilder {
	b.request.Appearance = appearance
	return b
}

func (b *FlightRequestBuilder) WithOverlay(overlay domain.OverlayConfig) *FlightRequestBuilder {
	b.request.Overlay = overlay
	return b
}

func (b *FlightRequestBuilder) WithQuality(quality domain.VideoQuality) *FlightRequestBuilder {
	b.request.Quality = quality
	return b
}

func (b *FlightRequestBuilder) WithOutput(output string) *FlightRequestBuilder {
	b.request.Output = output
	return b
}

func (b *FlightRequestBuilder) WithKeep(directory string) *FlightRequestBuilder {
	b.request.Keep = directory
	return b
}

func (b *FlightRequestBuilder) WithOverwrite() *FlightRequestBuilder {
	b.request.Overwrite = true
	return b
}

func (b *FlightRequestBuilder) Build() domain.FlightRequest {
	return b.request
}
