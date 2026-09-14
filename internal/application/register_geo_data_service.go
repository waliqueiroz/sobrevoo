package application

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// RegisterGeoDataInput is the input for RegisterGeoDataService.Execute.
type RegisterGeoDataInput struct {
	// Name is the identifier chosen by the user for this registration.
	Name string

	// Path is the local geo data file to register.
	Path string
}

// RegisterGeoDataOutput is the registration produced by
// RegisterGeoDataService.Execute (FR-001 through FR-003), with type,
// format and geographic area already determined.
type RegisterGeoDataOutput struct {
	Source domain.GeoDataSource
}

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/register_geo_data_service.go . RegisterGeoDataService

// RegisterGeoDataService registers a local map or elevation data file
// (FR-001 through FR-008). It depends only on ports declared in the
// domain, so it can be reused unchanged by any future entrypoint without
// duplicating any business rule (Constitution Principle III).
type RegisterGeoDataService interface {
	Execute(input RegisterGeoDataInput) (RegisterGeoDataOutput, error)
}

type registerGeoDataService struct {
	registry  domain.GeoDataRegistry
	inspector domain.GeoDataInspector
	clock     domain.Clock
}

// NewRegisterGeoDataService creates a RegisterGeoDataService backed by the
// given ports.
func NewRegisterGeoDataService(registry domain.GeoDataRegistry, inspector domain.GeoDataInspector, clock domain.Clock) RegisterGeoDataService {
	return &registerGeoDataService{
		registry:  registry,
		inspector: inspector,
		clock:     clock,
	}
}

func (s *registerGeoDataService) Execute(input RegisterGeoDataInput) (RegisterGeoDataOutput, error) {
	_, found, err := s.registry.FindByName(input.Name)
	if err != nil {
		return RegisterGeoDataOutput{}, err
	}
	if found {
		return RegisterGeoDataOutput{}, domain.ErrDataSourceNameAlreadyUsed
	}

	inspected, err := s.inspector.Inspect(input.Path)
	if err != nil {
		return RegisterGeoDataOutput{}, err
	}

	source := domain.GeoDataSource{
		Name:         input.Name,
		Path:         input.Path,
		Type:         inspected.Type,
		Format:       inspected.Format,
		BoundingBox:  inspected.BoundingBox,
		RegisteredAt: s.clock.Now(),
	}

	if err := s.registry.Save(source); err != nil {
		return RegisterGeoDataOutput{}, err
	}

	return RegisterGeoDataOutput{Source: source}, nil
}
