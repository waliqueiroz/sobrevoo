package application

import (
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/camera_plan_service.go -package mockapplication . CameraPlanService

// CameraPlanService plans the camera flight of a GPS track (the third
// stage): it produces a CameraPlan and can export it. Like every service, it
// groups all the operations on one resource and only orchestrates — the
// planning rules live in internal/domain (camera_planning.go and friends),
// the track treatment in TrackService, and the file writing behind the
// domain.CameraPlanExporter port.
type CameraPlanService interface {
	// Generate reads and treats the track from reader and plans the camera
	// flight over it with the given parameters.
	Generate(reader io.Reader, parameters domain.PlanParameters) (domain.CameraPlan, error)

	// Export writes plan to path; unless overwrite is true it refuses a path
	// that already holds a file.
	Export(plan domain.CameraPlan, path string, overwrite bool) error
}

type cameraPlanService struct {
	trackService TrackService
	exporter     domain.CameraPlanExporter

	// defaultLevel is the simplification and smoothing level applied to the
	// track before planning, and tuning holds the planning constants. Both
	// are resolved by an outbound configuration adapter and injected by
	// whoever assembles the service (Constitution Principle VIII).
	defaultLevel domain.Level
	tuning       domain.CameraTuning
}

// NewCameraPlanService creates a CameraPlanService backed by the given
// TrackService and exporter port.
func NewCameraPlanService(
	trackService TrackService,
	exporter domain.CameraPlanExporter,
	defaultLevel domain.Level,
	tuning domain.CameraTuning,
) CameraPlanService {
	return &cameraPlanService{
		trackService: trackService,
		exporter:     exporter,
		defaultLevel: defaultLevel,
		tuning:       tuning,
	}
}

func (s *cameraPlanService) Generate(reader io.Reader, parameters domain.PlanParameters) (domain.CameraPlan, error) {
	if err := parameters.Validate(); err != nil {
		return domain.CameraPlan{}, err
	}

	treated, err := s.trackService.Treat(reader, s.defaultLevel, s.defaultLevel)
	if err != nil {
		return domain.CameraPlan{}, err
	}

	return domain.PlanCamera(treated, parameters, s.tuning)
}

func (s *cameraPlanService) Export(plan domain.CameraPlan, path string, overwrite bool) error {
	return s.exporter.Export(plan, path, overwrite)
}
