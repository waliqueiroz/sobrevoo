package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// AppearanceBuilder builds a domain.Appearance: the orange trail, the red
// marker with white ring, and the dark background the tool has always drawn,
// unless told otherwise.
type AppearanceBuilder struct {
	appearance domain.Appearance
}

func NewAppearanceBuilder() *AppearanceBuilder {
	return &AppearanceBuilder{
		appearance: domain.Appearance{
			TrailColor:        domain.RGB{R: 0xFF, G: 0xB0, B: 0x00},
			TrailWidthRatio:   0.005,
			MarkerColor:       domain.RGB{R: 0xE5, G: 0x25, B: 0x2A},
			MarkerRadiusRatio: 0.012,
			BackgroundColor:   domain.RGB{R: 0x20, G: 0x26, B: 0x2E},
		},
	}
}

func (b *AppearanceBuilder) WithTrailColor(color domain.RGB) *AppearanceBuilder {
	b.appearance.TrailColor = color
	return b
}

func (b *AppearanceBuilder) WithTrailWidthRatio(ratio float64) *AppearanceBuilder {
	b.appearance.TrailWidthRatio = ratio
	return b
}

func (b *AppearanceBuilder) WithMarkerColor(color domain.RGB) *AppearanceBuilder {
	b.appearance.MarkerColor = color
	return b
}

func (b *AppearanceBuilder) WithMarkerRadiusRatio(ratio float64) *AppearanceBuilder {
	b.appearance.MarkerRadiusRatio = ratio
	return b
}

func (b *AppearanceBuilder) WithBackgroundColor(color domain.RGB) *AppearanceBuilder {
	b.appearance.BackgroundColor = color
	return b
}

func (b *AppearanceBuilder) Build() domain.Appearance {
	return b.appearance
}
