package domain

import "math"

// terrainLightDirection is the fixed, unit direction toward the light that
// shades the terrain, in the frame plane's basis (x grows to the east, y to
// the north, z is up) — the same basis framePlane/camera already use.
// Computed once, from TerrainLightAzimuthDegrees/TerrainLightAltitudeDegrees
// (render_tuning.go), never per pixel or per frame (research.md item 2).
var terrainLightDirection = newTerrainLightDirection()

// newTerrainLightDirection turns the fixed azimuth (clockwise from north,
// the same compass convention CameraFrame.Heading already uses) and
// altitude (above the horizon) into a unit direction.
func newTerrainLightDirection() [3]float64 {
	azimuth := degreesToRadians(TerrainLightAzimuthDegrees)
	altitude := degreesToRadians(TerrainLightAltitudeDegrees)
	sinAzimuth, cosAzimuth := math.Sin(azimuth), math.Cos(azimuth)
	sinAltitude, cosAltitude := math.Sin(altitude), math.Cos(altitude)

	return [3]float64{
		float64(cosAltitude * sinAzimuth),
		float64(cosAltitude * cosAzimuth),
		sinAltitude,
	}
}

// terrainLightFactor is how much a point of terrain whose surface has the
// given unit normal (nx, ny, nz) is lightened or darkened by the fixed
// directional light: the difference between the normal's and a flat
// surface's exposure to the light, dot(N, L) - dot(up, L) = dot(N - up, L),
// scaled by 1 and clamped to the fixed range. It is exactly 1 (unchanged)
// whenever the surface is flat (N = up = (0, 0, 1)), whatever the light's
// direction — never a special case, a consequence of N - up being zero
// there (research.md item 1).
func terrainLightFactor(nx, ny, nz float64) float64 {
	l := terrainLightDirection
	raw := float64(nx*l[0]) + float64(ny*l[1]) + float64((nz-1)*l[2])
	return clamp(1+raw, TerrainLightMinFactor, TerrainLightMaxFactor)
}
