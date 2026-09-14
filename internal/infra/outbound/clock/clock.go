// Package clock implements the domain.Clock port using the real system
// clock.
package clock

import "time"

// Clock implements domain.Clock using time.Now().
type Clock struct{}

// New creates a Clock.
func New() Clock {
	return Clock{}
}

// Now returns the current time.
func (Clock) Now() time.Time {
	return time.Now()
}
