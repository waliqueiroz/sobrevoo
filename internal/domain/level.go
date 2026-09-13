package domain

// Level is a named intensity preset shared by the Simplifier and Smoother
// ports (FR-014, FR-015, FR-016).
type Level int

const (
	LevelLow Level = iota
	LevelMedium
	LevelHigh
)
