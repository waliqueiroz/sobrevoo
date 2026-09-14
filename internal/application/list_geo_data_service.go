package application

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// GeoDataSummary is one registered source as reported by
// ListGeoDataService, with its live file availability (FR-010).
type GeoDataSummary struct {
	Source domain.GeoDataSource

	// Available is false when the file is no longer found at Source.Path
	// (FR-010).
	Available bool
}

// ListGeoDataOutput is the full registry as reported by
// ListGeoDataService.Execute (FR-009).
type ListGeoDataOutput struct {
	Sources []GeoDataSummary
}

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/list_geo_data_service.go . ListGeoDataService

// ListGeoDataService lists every registered geo data source (FR-009,
// FR-010). It depends only on ports declared in the domain, so it can be
// reused unchanged by any future entrypoint without duplicating any
// business rule (Constitution Principle III).
type ListGeoDataService interface {
	Execute() (ListGeoDataOutput, error)
}

type listGeoDataService struct {
	registry    domain.GeoDataRegistry
	fileChecker domain.FileChecker
}

// NewListGeoDataService creates a ListGeoDataService backed by the given
// ports.
func NewListGeoDataService(registry domain.GeoDataRegistry, fileChecker domain.FileChecker) ListGeoDataService {
	return &listGeoDataService{
		registry:    registry,
		fileChecker: fileChecker,
	}
}

func (s *listGeoDataService) Execute() (ListGeoDataOutput, error) {
	sources, err := s.registry.List()
	if err != nil {
		return ListGeoDataOutput{}, err
	}

	summaries := make([]GeoDataSummary, len(sources))
	for i, source := range sources {
		summaries[i] = GeoDataSummary{
			Source:    source,
			Available: s.fileChecker.Exists(source.Path),
		}
	}

	return ListGeoDataOutput{Sources: summaries}, nil
}
