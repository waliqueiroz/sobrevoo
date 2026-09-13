// Package domain holds Sobrevoo's core: entities, ports and pure business
// functions for reading and treating a GPS track. It never imports an
// infrastructure library (no XML/GPX parsing, no filesystem, no external
// process) — every such dependency is accessed through a port declared here
// and implemented in internal/infra/outbound (Constitution Principles I and
// II).
package domain

import "time"

// TrackPoint is a single recorded point of a track: a geographic position
// plus optional elevation and time, exactly as it may or may not have been
// provided by the source file.
type TrackPoint struct {
	Latitude  float64
	Longitude float64
	Elevation *float64
	Time      *time.Time
}

// HasElevation reports whether this point carries altitude data.
func (p TrackPoint) HasElevation() bool {
	return p.Elevation != nil
}

// HasTime reports whether this point carries an associated timestamp.
func (p TrackPoint) HasTime() bool {
	return p.Time != nil
}
