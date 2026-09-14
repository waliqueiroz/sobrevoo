package domain

import "time"

//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/clock.go . Clock

// Clock provides the current time. Isolating it behind a port keeps
// RegisterGeoDataService's use of "now" (GeoDataSource.RegisteredAt)
// deterministic and testable (Constitution Principle VI), instead of
// calling time.Now() directly from the core.
type Clock interface {
	Now() time.Time
}
