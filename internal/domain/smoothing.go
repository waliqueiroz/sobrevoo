package domain

//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/smoother.go . Smoother

// Smoother reduces the point-to-point jitter typical of GPS readings
// (FR-013, FR-015). Like Simplifier, it does not belong to a single entity,
// so it gets its own file instead of being declared alongside one.
type Smoother interface {
	Smooth(points []TrackPoint, level Level) []TrackPoint
}
