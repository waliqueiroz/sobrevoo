package domain

// DiscardStats counts how many points were discarded during treatment,
// broken down by reason (FR-011).
type DiscardStats struct {
	ImpossibleCoordinates int
	ConsecutiveDuplicates int
	ImplausibleJumps      int
}

// Total returns the sum of all discarded points, regardless of reason.
func (d DiscardStats) Total() int {
	return d.ImpossibleCoordinates + d.ConsecutiveDuplicates + d.ImplausibleJumps
}
