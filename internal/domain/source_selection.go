package domain

import "fmt"

// SourceSelection is the user's explicit choice of which registered source
// to use for a base map and/or an elevation, in place of the automatic
// selection by area (SelectSource) — 010-geo-data-source-control, FR-004
// through FR-010.
type SourceSelection struct {
	// BaseMapName, when not nil, is the name of the registered base map
	// source to use exclusively. nil means the automatic selection decides,
	// exactly as it did before this feature (FR-009).
	BaseMapName *string

	// ElevationName is the same, for elevation. The two fields are resolved
	// independently (FR-008).
	ElevationName *string
}

// Resolve narrows baseMaps/elevations — the candidates already gathered and
// filtered to the ones still available on disk — to a single source for
// each type SourceSelection names, leaving the other type's list untouched.
// It does no I/O: it only inspects the two lists already in memory.
//
// Feeding the narrowed list of one candidate back into the very same
// automatic-selection functions (Route.Coverage/SelectSource) that already
// decide among many is what makes FR-007 (refuse on incomplete coverage)
// and FR-010 (never mix) hold without any change to those functions: a
// single-candidate list can either cover a point or not, and there is no
// second candidate left to mix in.
//
// A name not found in either list is ErrDataSourceNotRegistered; a name
// found in the list of the other type is ErrDataSourceTypeMismatch — both
// name the source and, for the mismatch, its actual type (FR-006).
func (s SourceSelection) Resolve(baseMaps, elevations []GeoDataSource) (resolvedBaseMaps, resolvedElevations []GeoDataSource, err error) {
	resolvedBaseMaps, err = resolveOne(s.BaseMapName, DataTypeBaseMap, baseMaps, elevations)
	if err != nil {
		return nil, nil, err
	}

	resolvedElevations, err = resolveOne(s.ElevationName, DataTypeElevation, elevations, baseMaps)
	if err != nil {
		return nil, nil, err
	}

	return resolvedBaseMaps, resolvedElevations, nil
}

// resolveOne resolves a single requested name against the candidates of its
// own type; sameType is left as-is (nil name) or narrowed to the one match
// (name found); otherType is consulted only to tell a type mismatch from an
// unregistered name.
func resolveOne(name *string, wantType DataType, sameType, otherType []GeoDataSource) ([]GeoDataSource, error) {
	if name == nil {
		return sameType, nil
	}

	if found, ok := findSourceByName(sameType, *name); ok {
		return []GeoDataSource{found}, nil
	}

	if found, ok := findSourceByName(otherType, *name); ok {
		return nil, fmt.Errorf("%w: %q is registered as %s, not %s", ErrDataSourceTypeMismatch, *name, found.Type, wantType)
	}

	return nil, fmt.Errorf("%w: %q (as %s)", ErrDataSourceNotRegistered, *name, wantType)
}

// findSourceByName looks up a candidate by name in sources.
func findSourceByName(sources []GeoDataSource, name string) (GeoDataSource, bool) {
	for _, source := range sources {
		if source.Name == name {
			return source, true
		}
	}
	return GeoDataSource{}, false
}
