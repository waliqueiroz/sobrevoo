package application

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// RemoveGeoDataInput is the input for RemoveGeoDataService.Execute.
type RemoveGeoDataInput struct {
	// Name is the registered source to remove.
	Name string
}

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/remove_geo_data_service.go . RemoveGeoDataService

// RemoveGeoDataService removes a registered geo data source by name,
// without touching the underlying data file on disk (FR-011, FR-012). It
// depends only on ports declared in the domain, so it can be reused
// unchanged by any future entrypoint without duplicating any business rule
// (Constitution Principle III).
type RemoveGeoDataService interface {
	Execute(input RemoveGeoDataInput) error
}

type removeGeoDataService struct {
	registry domain.GeoDataRegistry
}

// NewRemoveGeoDataService creates a RemoveGeoDataService backed by the
// given port.
func NewRemoveGeoDataService(registry domain.GeoDataRegistry) RemoveGeoDataService {
	return &removeGeoDataService{registry: registry}
}

func (s *removeGeoDataService) Execute(input RemoveGeoDataInput) error {
	_, found, err := s.registry.FindByName(input.Name)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrDataSourceNotRegistered
	}

	return s.registry.Delete(input.Name)
}
