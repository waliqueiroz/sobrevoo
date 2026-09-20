package jsonfile_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/jsonfile"
)

func examplePlan() domain.CameraPlan {
	frames := []domain.CameraFrame{
		{Index: 0, Time: 0, Phase: domain.PhaseOpening, CameraLatitude: -23.5505199, CameraLongitude: -46.6333094, CameraAltitude: 912.804, Heading: 0, Tilt: 60, MarkerLatitude: -23.5505199, MarkerLongitude: -46.6333094, MarkerDistance: 0, CameraToMarkerDistance: 1054.02},
		{Index: 1, Time: 33333333 * time.Nanosecond, Phase: domain.PhaseFollowing, CameraLatitude: -23.55, CameraLongitude: 179.9999999, CameraAltitude: 100.5, Heading: 359.999, Tilt: 45, MarkerLatitude: -23.55, MarkerLongitude: -179.9999999, MarkerDistance: 12.5, CameraToMarkerDistance: 141.421},
	}
	parameters := builddomain.NewPlanParametersBuilder().WithDuration(42 * time.Second).WithFrameRate(29.97).WithDistance(domain.LevelHigh).WithTilt(domain.LevelLow).Build()
	return builddomain.NewCameraPlanBuilder().
		WithParameters(parameters).
		WithDurationMode(domain.DurationModeAutomatic).
		WithTimeReference(domain.TimeReferenceDistance, "no time data").
		WithFrames(frames...).
		WithSmoothedSpans(domain.SmoothedSpan{Start: 0, End: 1200 * time.Millisecond, Quantity: domain.QuantityHeading}).
		Build()
}

func exportToTemp(t *testing.T, plan domain.CameraPlan) (path string, content []byte) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "plan.json")
	require.NoError(t, jsonfile.NewCameraPlanExporter().Export(plan, path, false))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return path, content
}

func Test_CameraPlanExporter_Export(t *testing.T) {
	t.Run("should write the format version first, then parameters, summary and frames", func(t *testing.T) {
		// when
		_, content := exportToTemp(t, examplePlan())

		// then
		text := string(content)
		assert.True(t, strings.HasPrefix(text, "{\n  \"format_version\": 1,\n  \"parameters\": {"))
		assert.Less(t, strings.Index(text, `"parameters"`), strings.Index(text, `"summary"`))
		assert.Less(t, strings.Index(text, `"summary"`), strings.Index(text, `"frames"`))
		assert.True(t, strings.HasSuffix(text, "]\n}\n"))
	})

	t.Run("should write each frame compactly on a single line", func(t *testing.T) {
		// when
		_, content := exportToTemp(t, examplePlan())

		// then
		lines := strings.Split(string(content), "\n")
		var frameLines []string
		for _, l := range lines {
			if strings.HasPrefix(l, `    {"index":`) {
				frameLines = append(frameLines, l)
			}
		}
		require.Len(t, frameLines, 2)
		assert.Equal(t, `    {"index":0,"time_s":0,"phase":"opening","camera":{"lat":-23.5505199,"lon":-46.6333094,"altitude_m":912.804},"heading_deg":0,"tilt_deg":60,"marker":{"lat":-23.5505199,"lon":-46.6333094,"distance_m":0},"camera_to_marker_m":1054.02},`, frameLines[0])
		assert.True(t, strings.HasSuffix(frameLines[1], "}"), "the last frame has no trailing comma")
	})

	t.Run("should be valid JSON that reads back with every value of the plan", func(t *testing.T) {
		// given
		plan := examplePlan()

		// when
		_, content := exportToTemp(t, plan)

		// then
		var decoded struct {
			FormatVersion int `json:"format_version"`
			Parameters    struct {
				DurationS float64 `json:"duration_s"`
				FrameRate float64 `json:"frame_rate"`
				Distance  string  `json:"distance"`
				Tilt      string  `json:"tilt"`
			} `json:"parameters"`
			Summary struct {
				DurationS      float64 `json:"duration_s"`
				DurationMode   string  `json:"duration_mode"`
				FrameCount     int     `json:"frame_count"`
				CameraAltitude struct {
					Min float64 `json:"min"`
					Max float64 `json:"max"`
				} `json:"camera_altitude_m"`
				CameraDistance struct {
					Min float64 `json:"min"`
					Max float64 `json:"max"`
				} `json:"camera_distance_m"`
				TimeReference      string `json:"time_reference"`
				TimeFallbackReason string `json:"time_fallback_reason"`
				SmoothedSpans      []struct {
					StartS   float64 `json:"start_s"`
					EndS     float64 `json:"end_s"`
					Quantity string  `json:"quantity"`
				} `json:"smoothed_spans"`
			} `json:"summary"`
			Frames []struct {
				Index  int     `json:"index"`
				TimeS  float64 `json:"time_s"`
				Phase  string  `json:"phase"`
				Camera struct {
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
					AltitudeM float64 `json:"altitude_m"`
				} `json:"camera"`
				HeadingDeg float64 `json:"heading_deg"`
				TiltDeg    float64 `json:"tilt_deg"`
				Marker     struct {
					Lat       float64 `json:"lat"`
					Lon       float64 `json:"lon"`
					DistanceM float64 `json:"distance_m"`
				} `json:"marker"`
				CameraToMkr float64 `json:"camera_to_marker_m"`
			} `json:"frames"`
		}
		require.NoError(t, json.Unmarshal(content, &decoded))

		assert.Equal(t, 1, decoded.FormatVersion)
		assert.Equal(t, 42.0, decoded.Parameters.DurationS)
		assert.Equal(t, 29.97, decoded.Parameters.FrameRate)
		assert.Equal(t, "high", decoded.Parameters.Distance)
		assert.Equal(t, "low", decoded.Parameters.Tilt)
		assert.Equal(t, "automatic", decoded.Summary.DurationMode)
		assert.Equal(t, plan.Summary.FrameCount, decoded.Summary.FrameCount)
		assert.Equal(t, plan.Summary.MinCameraAltitude, decoded.Summary.CameraAltitude.Min)
		assert.Equal(t, plan.Summary.MaxCameraDistance, decoded.Summary.CameraDistance.Max)
		assert.Equal(t, "distance", decoded.Summary.TimeReference)
		assert.Equal(t, "no time data", decoded.Summary.TimeFallbackReason)
		require.Len(t, decoded.Summary.SmoothedSpans, 1)
		assert.Equal(t, 1.2, decoded.Summary.SmoothedSpans[0].EndS)
		assert.Equal(t, "heading", decoded.Summary.SmoothedSpans[0].Quantity)

		require.Len(t, decoded.Frames, len(plan.Frames))
		for i, want := range plan.Frames {
			got := decoded.Frames[i]
			assert.Equal(t, want.Index, got.Index)
			assert.InDelta(t, want.Time.Seconds(), got.TimeS, 1e-9)
			assert.Equal(t, string(want.Phase), got.Phase)
			assert.Equal(t, want.CameraLatitude, got.Camera.Lat)
			assert.Equal(t, want.CameraLongitude, got.Camera.Lon)
			assert.Equal(t, want.CameraAltitude, got.Camera.AltitudeM)
			assert.Equal(t, want.Heading, got.HeadingDeg)
			assert.Equal(t, want.Tilt, got.TiltDeg)
			assert.Equal(t, want.MarkerLatitude, got.Marker.Lat)
			assert.Equal(t, want.MarkerLongitude, got.Marker.Lon)
			assert.Equal(t, want.MarkerDistance, got.Marker.DistanceM)
			assert.Equal(t, want.CameraToMarkerDistance, got.CameraToMkr)
		}
	})

	t.Run("should mark the duration as explicit for a requested duration", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().WithDurationMode(domain.DurationModeExplicit).Build()

		// when
		_, content := exportToTemp(t, plan)

		// then
		assert.Contains(t, string(content), `"duration_mode": "explicit"`)
	})

	t.Run("should always write the fallback reason and the smoothed spans, even when empty", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().Build()

		// when
		_, content := exportToTemp(t, plan)

		// then
		assert.Contains(t, string(content), `"time_fallback_reason": ""`)
		assert.Contains(t, string(content), `"smoothed_spans": []`)
	})

	t.Run("should write a plan without frames", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().WithFrames().Build()

		// when
		_, content := exportToTemp(t, plan)

		// then
		assert.Contains(t, string(content), `"frames": []`)
		assert.True(t, json.Valid(content))
	})

	t.Run("should print numbers with fixed places and no trailing zeros, never negative zero", func(t *testing.T) {
		// given
		frame := builddomain.NewCameraFrameBuilder().WithHeading(90).Build()
		frame.MarkerLatitude = -0.00000001 // rounds to -0 at seven places
		frame.CameraLatitude = 10.5
		plan := builddomain.NewCameraPlanBuilder().WithFrames(frame).Build()

		// when
		_, content := exportToTemp(t, plan)

		// then
		text := string(content)
		assert.Contains(t, text, `"heading_deg":90,`)
		assert.Contains(t, text, `"lat":10.5,`)
		assert.Contains(t, text, `"marker":{"lat":0,`)
		assert.NotContains(t, text, "-0,")
	})

	t.Run("should produce identical bytes for a hundred exports of the same plan", func(t *testing.T) {
		// given
		plan := examplePlan()
		_, first := exportToTemp(t, plan)

		for i := 0; i < 100; i++ {
			// when
			_, again := exportToTemp(t, plan)

			// then
			require.True(t, bytes.Equal(first, again))
		}
	})

	t.Run("should refuse an existing destination without overwrite, leaving it intact", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "plan.json")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := jsonfile.NewCameraPlanExporter().Export(examplePlan(), path, false)

		// then
		require.ErrorIs(t, err, domain.ErrPlanDestinationExists)
		assert.Contains(t, err.Error(), "--overwrite")
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
	})

	t.Run("should replace an existing destination with overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "plan.json")
		require.NoError(t, os.WriteFile(path, []byte("old content that is longer than nothing"), 0o600))

		// when
		err := jsonfile.NewCameraPlanExporter().Export(examplePlan(), path, true)

		// then
		require.NoError(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.True(t, json.Valid(content))
		assert.NotContains(t, string(content), "old content")
	})

	t.Run("should create the file when it does not exist, even with overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "plan.json")

		// when
		err := jsonfile.NewCameraPlanExporter().Export(examplePlan(), path, true)

		// then
		require.NoError(t, err)
		assert.FileExists(t, path)
	})

	t.Run("should report a destination in a missing directory as invalid and leave nothing behind", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "missing", "plan.json")

		// when
		err := jsonfile.NewCameraPlanExporter().Export(examplePlan(), path, false)

		// then
		require.ErrorIs(t, err, domain.ErrPlanDestinationInvalid)
		assert.Contains(t, err.Error(), path)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Empty(t, namesOf(entries))
	})

	t.Run("should report a destination that is a directory as invalid when overwriting", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "plan.json")
		require.NoError(t, os.Mkdir(path, 0o755))

		// when
		err := jsonfile.NewCameraPlanExporter().Export(examplePlan(), path, true)

		// then
		require.ErrorIs(t, err, domain.ErrPlanDestinationInvalid)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Equal(t, []string{"plan.json"}, namesOf(entries), "no temporary file is left")
	})

	t.Run("should report an unwritable directory as invalid and leave nothing behind", func(t *testing.T) {
		// given
		if os.Geteuid() == 0 {
			t.Skip("permissions are not enforced for root")
		}
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { os.Chmod(dir, 0o700) })

		// when
		err := jsonfile.NewCameraPlanExporter().Export(examplePlan(), filepath.Join(dir, "plan.json"), false)

		// then
		require.ErrorIs(t, err, domain.ErrPlanDestinationInvalid)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Empty(t, entries)
	})

	t.Run("should leave only the plan file in the directory after a successful export", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		require.NoError(t, jsonfile.NewCameraPlanExporter().Export(examplePlan(), filepath.Join(dir, "plan.json"), false))

		// then
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Equal(t, []string{"plan.json"}, namesOf(entries))
	})

	t.Run("should create a readable file", func(t *testing.T) {
		// when
		path, _ := exportToTemp(t, examplePlan())

		// then
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})
}

func namesOf(entries []os.DirEntry) []string {
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
