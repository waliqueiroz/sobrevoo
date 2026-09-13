// Package helper provides shared test fixtures and builders used across the
// project's test suites (test/helper), so individual test files do not need
// to duplicate sample GPX content or TrackPoint construction.
package helper

// ValidGPXWithAltitudeAndTime returns a well-formed GPX document whose
// points all carry both altitude and time.
func ValidGPXWithAltitudeAndTime() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="sobrevoo-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="40.4168" lon="-3.7038">
        <ele>650.0</ele>
        <time>2026-01-01T08:00:00Z</time>
      </trkpt>
      <trkpt lat="40.4170" lon="-3.7030">
        <ele>652.5</ele>
        <time>2026-01-01T08:00:10Z</time>
      </trkpt>
      <trkpt lat="40.4175" lon="-3.7020">
        <ele>655.0</ele>
        <time>2026-01-01T08:00:20Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
}

// ValidGPXWithoutAltitude returns a well-formed GPX document whose points
// have time but no altitude at all.
func ValidGPXWithoutAltitude() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="sobrevoo-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="40.4168" lon="-3.7038">
        <time>2026-01-01T08:00:00Z</time>
      </trkpt>
      <trkpt lat="40.4170" lon="-3.7030">
        <time>2026-01-01T08:00:10Z</time>
      </trkpt>
      <trkpt lat="40.4175" lon="-3.7020">
        <time>2026-01-01T08:00:20Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
}

// ValidGPXWithoutTime returns a well-formed GPX document whose points have
// altitude but no time at all.
func ValidGPXWithoutTime() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="sobrevoo-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="40.4168" lon="-3.7038">
        <ele>650.0</ele>
      </trkpt>
      <trkpt lat="40.4170" lon="-3.7030">
        <ele>652.5</ele>
      </trkpt>
      <trkpt lat="40.4175" lon="-3.7020">
        <ele>655.0</ele>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
}

// GPXCrossingAntimeridian returns a well-formed GPX document whose points
// alternate between longitudes just below +180 and just above -180, at a
// walking/running pace (a few meters every 10 seconds) so the fixture stays
// valid once implausible-jump discarding is applied downstream.
func GPXCrossingAntimeridian() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="sobrevoo-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Antimeridian Track</name>
    <trkseg>
      <trkpt lat="0.0" lon="179.9998">
        <ele>10.0</ele>
        <time>2026-01-01T08:00:00Z</time>
      </trkpt>
      <trkpt lat="0.0" lon="179.9999">
        <ele>10.0</ele>
        <time>2026-01-01T08:00:10Z</time>
      </trkpt>
      <trkpt lat="0.0" lon="-179.9999">
        <ele>10.0</ele>
        <time>2026-01-01T08:00:20Z</time>
      </trkpt>
      <trkpt lat="0.0" lon="-179.9998">
        <ele>10.0</ele>
        <time>2026-01-01T08:00:30Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
}

// EmptyContent returns an empty file's content (FR-005).
func EmptyContent() string {
	return ""
}

// NonGPXContent returns content that is not any recognized track format
// (FR-004).
func NonGPXContent() string {
	return "this is not a GPS track file"
}

// MalformedGPX returns content whose root element is a GPX root but whose
// body is not well-formed XML.
func MalformedGPX() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="sobrevoo-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
      <trkpt lat="40.4168" lon="-3.7038">
</gpx>`
}

// GPXWithSinglePoint returns a well-formed GPX document with only one
// point — insufficient to compute a summary (FR-006).
func GPXWithSinglePoint() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="sobrevoo-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
      <trkpt lat="40.4168" lon="-3.7038">
        <ele>650.0</ele>
        <time>2026-01-01T08:00:00Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
}
