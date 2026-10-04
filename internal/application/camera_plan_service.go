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
	// flight over it with the given parameters. treated is called once the
	// track is treated, right before the planning starts, so a caller can
	// tell the two apart as they happen (may be nil).
	Generate(reader io.Reader, parameters domain.PlanParameters, treated func()) (domain.CameraPlan, error)

	// Export writes plan to path; unless overwrite is true it refuses a path
	// that already holds a file.
	Export(plan domain.CameraPlan, path string, overwrite bool) error

	// Load reads the plan file at path — one Export wrote — and checks that
	// it is coherent with itself, so what follows can trust it.
	Load(path string) (domain.CameraPlan, error)
}

type cameraPlanService struct {
	trackService TrackService
	exporter     domain.CameraPlanExporter
	reader       domain.CameraPlanReader

	// tuning holds the planning constants, resolved by an outbound
	// configuration adapter and injected by whoever assembles the service
	// (Constitution Principle VIII). The simplification and smoothing level
	// applied to the track before planning is not resolved here: it always
	// comes already resolved in parameters.Simplification/.Smoothing
	// (013-treatment-level-flags), the same way Distance/Tilt already do.
	tuning domain.CameraTuning
}

// NewCameraPlanService creates a CameraPlanService backed by the given
// TrackService and exporter and reader ports.
func NewCameraPlanService(
	trackService TrackService,
	exporter domain.CameraPlanExporter,
	reader domain.CameraPlanReader,
	tuning domain.CameraTuning,
) CameraPlanService {
	return &cameraPlanService{
		trackService: trackService,
		exporter:     exporter,
		reader:       reader,
		tuning:       tuning,
	}
}

func (s *cameraPlanService) Generate(reader io.Reader, parameters domain.PlanParameters, treated func()) (domain.CameraPlan, error) {
	if err := parameters.Validate(); err != nil {
		return domain.CameraPlan{}, err
	}

	track, err := s.trackService.Treat(reader, parameters.Simplification, parameters.Smoothing)
	if err != nil {
		return domain.CameraPlan{}, err
	}
	if treated != nil {
		treated()
	}

	return track.PlanCamera(parameters, s.tuning)
}

func (s *cameraPlanService) Export(plan domain.CameraPlan, path string, overwrite bool) error {
	return s.exporter.Export(plan, path, overwrite)
}

func (s *cameraPlanService) Load(path string) (domain.CameraPlan, error) {
	plan, err := s.reader.Read(path)
	if err != nil {
		return domain.CameraPlan{}, err
	}

	if err := plan.Validate(); err != nil {
		return domain.CameraPlan{}, err
	}

	return plan, nil
}
