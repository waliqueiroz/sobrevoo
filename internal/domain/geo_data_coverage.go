package domain

import "sort"

// CoverageStatus is ComputeCoverage's overall verdict for a track
// (SC-003). See CoverageReport for the exact rule that decides between the
// three values.
type CoverageStatus string

const (
	CoverageStatusFull    CoverageStatus = "full"
	CoverageStatusPartial CoverageStatus = "partial"
	CoverageStatusNone    CoverageStatus = "none"
)

// MissingDataType names what kind of geo data is missing for an
// UncoveredSegment (FR-015).
type MissingDataType string

const (
	MissingBaseMap   MissingDataType = "base map"
	MissingElevation MissingDataType = "elevation"
	MissingBoth      MissingDataType = "base map and elevation"
)

// UncoveredSegment is one contiguous run of route points not fully covered
// (FR-015, Clarification — spec.md).
type UncoveredSegment struct {
	StartLatitude, StartLongitude float64
	EndLatitude, EndLongitude     float64
	Missing                       MissingDataType
}

// CoverageReport is the verdict produced by ComputeCoverage.
//
// Status decision rule (resolves the ambiguity flagged by /speckit-analyze —
// see data-model.md):
//   - CoverageStatusFull: every route point is fully covered (UncoveredSegments is empty).
//   - CoverageStatusNone: no route point has coverage of either type anywhere —
//     equivalently, BaseMapSourcesUsed and ElevationSourcesUsed are both empty.
//   - CoverageStatusPartial: everything else — some real coverage exists
//     (at least one fully covered point, or at least one of the two source
//     sets is non-empty), just not for the whole route.
type CoverageReport struct {
	Status               CoverageStatus
	UncoveredSegments    []UncoveredSegment
	BaseMapSourcesUsed   []GeoDataSource
	ElevationSourcesUsed []GeoDataSource
}

// ComputeCoverage checks route against the registered base map and
// elevation candidates that are still available on disk (FR-013 through
// FR-018) — candidates whose file is no longer found are the caller's
// responsibility to exclude before calling this function (it is a pure
// function: no port, no I/O).
//
// For each point, the candidate of each type that wins is the most
// specific one (smallest BoundingBox.AreaDegrees), ties broken by the
// oldest RegisteredAt (FR-016, Clarification — spec.md). Points are then
// grouped into contiguous UncoveredSegment runs by their missing-data
// status (FR-015, Clarification — spec.md).
func ComputeCoverage(route []TrackPoint, baseMaps, elevations []GeoDataSource) CoverageReport {
	baseMapUsed := map[string]GeoDataSource{}
	elevationUsed := map[string]GeoDataSource{}

	var segments []UncoveredSegment
	segmentOpen := false

	for _, point := range route {
		baseMap, hasBaseMap := pickCoverageWinner(baseMaps, point.Latitude, point.Longitude)
		elevation, hasElevation := pickCoverageWinner(elevations, point.Latitude, point.Longitude)

		if hasBaseMap {
			baseMapUsed[baseMap.Name] = baseMap
		}
		if hasElevation {
			elevationUsed[elevation.Name] = elevation
		}

		missing, fullyCovered := missingDataType(hasBaseMap, hasElevation)
		if fullyCovered {
			segmentOpen = false
			continue
		}

		if segmentOpen && segments[len(segments)-1].Missing == missing {
			segments[len(segments)-1].EndLatitude = point.Latitude
			segments[len(segments)-1].EndLongitude = point.Longitude
			continue
		}

		segments = append(segments, UncoveredSegment{
			StartLatitude:  point.Latitude,
			StartLongitude: point.Longitude,
			EndLatitude:    point.Latitude,
			EndLongitude:   point.Longitude,
			Missing:        missing,
		})
		segmentOpen = true
	}

	baseMapSourcesUsed := sortedSources(baseMapUsed)
	elevationSourcesUsed := sortedSources(elevationUsed)

	status := CoverageStatusPartial
	switch {
	case len(segments) == 0:
		status = CoverageStatusFull
	case len(baseMapSourcesUsed) == 0 && len(elevationSourcesUsed) == 0:
		status = CoverageStatusNone
	}

	return CoverageReport{
		Status:               status,
		UncoveredSegments:    segments,
		BaseMapSourcesUsed:   baseMapSourcesUsed,
		ElevationSourcesUsed: elevationSourcesUsed,
	}
}

// missingDataType reports what is missing for a point, given whether a
// base map and an elevation source were found to cover it, and whether
// that makes the point fully covered.
func missingDataType(hasBaseMap, hasElevation bool) (missing MissingDataType, fullyCovered bool) {
	switch {
	case hasBaseMap && hasElevation:
		return "", true
	case hasBaseMap:
		return MissingElevation, false
	case hasElevation:
		return MissingBaseMap, false
	default:
		return MissingBoth, false
	}
}

// pickCoverageWinner returns the candidate covering (lat, lon) that wins
// the determinism rule (FR-016, Clarification — spec.md): the smallest
// BoundingBox.AreaDegrees (most specific); ties broken by the oldest
// RegisteredAt.
func pickCoverageWinner(candidates []GeoDataSource, lat, lon float64) (GeoDataSource, bool) {
	var winner GeoDataSource
	found := false

	for _, candidate := range candidates {
		if !candidate.BoundingBox.Contains(lat, lon) {
			continue
		}
		if !found || isMoreSpecific(candidate, winner) {
			winner = candidate
			found = true
		}
	}

	return winner, found
}

// isMoreSpecific reports whether a should win over b under FR-016's
// desempate rule.
func isMoreSpecific(a, b GeoDataSource) bool {
	areaA, areaB := a.BoundingBox.AreaDegrees(), b.BoundingBox.AreaDegrees()
	if areaA != areaB {
		return areaA < areaB
	}
	return a.RegisteredAt.Before(b.RegisteredAt)
}

// sortedSources returns used's values sorted by Name, so CoverageReport is
// deterministic regardless of map iteration order (SC-005).
func sortedSources(used map[string]GeoDataSource) []GeoDataSource {
	sources := make([]GeoDataSource, 0, len(used))
	for _, source := range used {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
	return sources
}
