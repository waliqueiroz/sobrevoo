package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/simplifier.go -package mockdomain . Simplifier

// Simplifier reduces the number of points in a route while preserving its
// overall shape (FR-012, FR-014). It does not belong to a single entity —
// unlike TrackParser, which produces a Track — so it gets its own file
// instead of being declared alongside one.
type Simplifier interface {
	Simplify(points []TrackPoint, level Level) []TrackPoint
}
