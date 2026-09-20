package domain

// Level is a named intensity preset shared by the Simplifier and Smoother
// ports (FR-014, FR-015, FR-016).
type Level int

const (
	LevelLow Level = iota
	LevelMedium
	LevelHigh
)

// index is the position of the level in the per-level tables of CameraTuning,
// clamped so an out-of-range level can never index outside them.
func (l Level) index() int {
	return min(max(int(l), 0), levelCount-1)
}
