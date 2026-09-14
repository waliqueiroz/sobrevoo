// Package jsonfile implements the domain.GeoDataRegistry port as a single
// JSON file on disk (research.md item 5).
package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// Store implements domain.GeoDataRegistry, persisting registered sources as
// a single JSON file at Path.
type Store struct {
	path string
}

// New creates a Store that persists the registry at path.
func New(path string) *Store {
	return &Store{path: path}
}

// Save adds or replaces the registered source with the same Name (FR-001,
// FR-007).
func (s *Store) Save(source domain.GeoDataSource) error {
	sources, err := s.read()
	if err != nil {
		return err
	}

	replaced := false
	for i, existing := range sources {
		if existing.Name == source.Name {
			sources[i] = source
			replaced = true
			break
		}
	}
	if !replaced {
		sources = append(sources, source)
	}

	return s.write(sources)
}

// FindByName looks up a registered source by name. It reports
// (GeoDataSource{}, false, nil) — not an error — when no source with that
// name is registered.
func (s *Store) FindByName(name string) (domain.GeoDataSource, bool, error) {
	sources, err := s.read()
	if err != nil {
		return domain.GeoDataSource{}, false, err
	}

	for _, existing := range sources {
		if existing.Name == name {
			return existing, true, nil
		}
	}

	return domain.GeoDataSource{}, false, nil
}

// List returns every registered source (FR-009).
func (s *Store) List() ([]domain.GeoDataSource, error) {
	return s.read()
}

// Delete removes the registered source with the given name. It never
// touches the underlying data file on disk (FR-011). Deleting a name that
// is not registered is a no-op — callers (the application service) are
// responsible for reporting domain.ErrDataSourceNotRegistered beforehand.
func (s *Store) Delete(name string) error {
	sources, err := s.read()
	if err != nil {
		return err
	}

	filtered := make([]domain.GeoDataSource, 0, len(sources))
	for _, existing := range sources {
		if existing.Name != name {
			filtered = append(filtered, existing)
		}
	}

	return s.write(filtered)
}

// read loads the registry file's content. A missing file (e.g. the very
// first run) is treated as an empty registry, not an error.
func (s *Store) read() ([]domain.GeoDataSource, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading geo data registry: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var sources []domain.GeoDataSource
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, fmt.Errorf("parsing geo data registry: %w", err)
	}

	return sources, nil
}

// write persists sources atomically: it writes to a temporary file in the
// same directory and then renames it over the registry file, so a process
// interrupted mid-write never leaves a corrupted registry behind
// (research.md item 5).
func (s *Store) write(sources []domain.GeoDataSource) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding geo data registry: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".registry-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary registry file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temporary registry file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temporary registry file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("promoting temporary registry file: %w", err)
	}

	return nil
}
