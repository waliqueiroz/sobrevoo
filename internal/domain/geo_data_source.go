package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/geo_data_inspector.go -package mockdomain . GeoDataInspector
//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/geo_data_repository.go -package mockdomain . GeoDataRepository

import "time"

// GeoDataInspector examines a local map or elevation data file and
// determines its type and the geographic area it covers (FR-002, FR-003).
// Concrete implementations (one per recognized format, plus the
// format-detection glue) live in internal/infra/outbound/geodatainspector.
//
// Unlike TrackParser, Inspect takes a file path rather than an io.Reader:
// the underlying formats (a SQLite database for MBTiles, TIFF tags for
// GeoTIFF) need random access to the file, which an io.Reader alone does
// not provide without loading a potentially large file entirely into
// memory (research.md item 8). The concrete adapter — not the core — is
// what actually opens the file, translating filesystem errors into
// ErrDataFileNotFound/ErrDataFileUnreadable.
type GeoDataInspector interface {
	Inspect(path string) (InspectedGeoData, error)
}

// GeoDataRepository persists and retrieves the set of registered
// GeoDataSource entries (FR-008). Implemented by
// internal/infra/outbound/jsonfile.
type GeoDataRepository interface {
	// Save adds or replaces the registered source with the same Name.
	Save(source GeoDataSource) error

	// FindByName looks up a registered source by name. The second return
	// value is false (with a zero GeoDataSource and a nil error) when no
	// source with that name is registered — this is not an error case.
	FindByName(name string) (GeoDataSource, bool, error)

	// List returns every registered source.
	List() ([]GeoDataSource, error)

	// Delete removes the registered source with the given name. It never
	// touches the underlying data file on disk (FR-011).
	Delete(name string) error
}

// DataType identifies what a GeoDataSource represents (FR-002).
type DataType string

const (
	DataTypeBaseMap   DataType = "base map"
	DataTypeElevation DataType = "elevation"
)

// DataFormat identifies the concrete file format a GeoDataSource was
// recognized from (research.md items 1 and 3). Kept separate from DataType
// so that a second format for the same type (e.g. a second base map
// format) can be added later without changing this entity, mirroring why
// Track has both a Format and no separate "kind" field.
type DataFormat string

const (
	DataFormatMBTiles DataFormat = "MBTiles"
	DataFormatGeoTIFF DataFormat = "GeoTIFF"
)

// GeoDataSource is a local map or elevation data file the user has
// registered with the tool (FR-001 through FR-003; data-model.md).
type GeoDataSource struct {
	// Name is the identifier chosen by the user, unique among all
	// registered sources (FR-007).
	Name string

	// Path is the file's location on the filesystem, exactly as given at
	// registration time.
	Path string

	// Type and Format are determined automatically from the file's content
	// at registration time (FR-002) — never supplied by the user.
	Type   DataType
	Format DataFormat

	// BoundingBox is the geographic area the source covers, determined
	// automatically from the file's content at registration time (FR-003).
	BoundingBox BoundingBox

	// RegisteredAt is when this source was registered (time.Now(), stamped
	// by NewGeoDataSource — not worth a Clock port, since time.Now() is a
	// stdlib call, not an external dependency in the Constitution's sense,
	// and RegisteredAt is only ever compared to itself). Used only to break
	// ties deterministically between overlapping sources of the same type
	// (FR-016) — not business data by itself.
	RegisteredAt time.Time
}

// NewGeoDataSource builds a GeoDataSource from what a GeoDataInspector
// discovered about the file at path, stamping RegisteredAt with the
// current time — the same way domain.NewGroup stamps CreatedAt in
// waliqueiroz/mystery-gifter-api: time.Now() is a stdlib call, not an
// external dependency in the Constitution's sense (Princípio II), so it is
// called directly here rather than through a Clock port (research.md item
// 10.1).
func NewGeoDataSource(name, path string, inspected InspectedGeoData) GeoDataSource {
	return GeoDataSource{
		Name:         name,
		Path:         path,
		Type:         inspected.Type,
		Format:       inspected.Format,
		BoundingBox:  inspected.BoundingBox,
		RegisteredAt: time.Now(),
	}
}

// InspectedGeoData is what a GeoDataInspector discovers by examining a
// geographic data file's content.
type InspectedGeoData struct {
	Format      DataFormat
	Type        DataType
	BoundingBox BoundingBox
}

// GeoDataSummary is one registered source together with its live file
// availability (FR-010), as reported by GeoDataService.List.
type GeoDataSummary struct {
	Source GeoDataSource

	// Available is false when the file is no longer found at Source.Path
	// (FR-010).
	Available bool
}
