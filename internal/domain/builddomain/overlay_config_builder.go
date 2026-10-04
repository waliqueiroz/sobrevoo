package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

type OverlayConfigBuilder struct {
	config domain.OverlayConfig
}

func NewOverlayConfigBuilder() *OverlayConfigBuilder {
	return &OverlayConfigBuilder{
		config: domain.OverlayConfig{Enabled: true, Distance: true, Elevation: true, Time: true, Profile: true},
	}
}

func (b *OverlayConfigBuilder) WithDisabled() *OverlayConfigBuilder {
	b.config = domain.OverlayConfig{}
	return b
}

func (b *OverlayConfigBuilder) WithoutDistance() *OverlayConfigBuilder {
	b.config.Distance = false
	return b
}

func (b *OverlayConfigBuilder) WithoutElevation() *OverlayConfigBuilder {
	b.config.Elevation = false
	return b
}

func (b *OverlayConfigBuilder) WithoutTime() *OverlayConfigBuilder {
	b.config.Time = false
	return b
}

func (b *OverlayConfigBuilder) WithoutProfile() *OverlayConfigBuilder {
	b.config.Profile = false
	return b
}

// WithSpeed turns on the speed block, which — unlike the other four — is
// off by default (014-speed-overlay-block).
func (b *OverlayConfigBuilder) WithSpeed() *OverlayConfigBuilder {
	b.config.Speed = true
	return b
}

// WithGain turns on the gain block, which — like WithSpeed — is off by
// default (015-overlay-redesign).
func (b *OverlayConfigBuilder) WithGain() *OverlayConfigBuilder {
	b.config.Gain = true
	return b
}

func (b *OverlayConfigBuilder) Build() domain.OverlayConfig {
	return b.config
}
