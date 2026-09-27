package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/frame_repository.go -package mockdomain . FrameRepository
//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/frame_exporter.go -package mockdomain . FrameExporter

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// FrameRepository keeps the frames of a flight in a directory, one image each.
// Concrete implementations live in internal/infra/outbound.
type FrameRepository interface {
	// Inspect lists the frame files of dir — the files named FrameFileName(n)
	// — and says, for each, whether this tool drew it, of which set, and
	// whether it is whole for a frame of resolution. A dir that does not exist
	// is an empty FrameDirectory; a path that is not a directory fails with
	// ErrFrameDestinationInvalid.
	Inspect(dir string, resolution Resolution) (FrameDirectory, error)

	// Save publishes the frame as dir/FrameFileName(index), marked as belonging
	// to the set id, creating dir when it does not exist (its parent must). A
	// file that is already there is replaced — deciding that is FrameDirectory's
	// job — and a failure never leaves a partial file: the previous file, if
	// any, stays as it was. Failures are ErrFrameDestinationInvalid.
	Save(dir string, index int, id FrameSetID, image FrameImage) error

	// Remove deletes the frame files with the given numbers; one that is not
	// there is not an error.
	Remove(dir string, indexes []int) error
}

// FrameExporter writes one frame to a file the user chose. Concrete
// implementations live in internal/infra/outbound.
type FrameExporter interface {
	// Export writes image to path, marked as belonging to the set id. Unless
	// overwrite is true it refuses a path that already exists
	// (ErrFrameDestinationExists), and it never leaves a partial file: on
	// failure the previous file, if any, stays as it was. A path that cannot be
	// written fails with ErrFrameDestinationInvalid.
	Export(image FrameImage, id FrameSetID, path string, overwrite bool) error
}

// FrameSetID identifies a set of frames: those drawn from the same plan, the
// same slice file, at the same resolution, by the same version of the drawing
// and with the same tuning. It goes inside every image, so frames of another
// set are told from the ones a directory may keep.
type FrameSetID string

// NewFrameSetID is the SHA-256, in lowercase hexadecimal, of the plan's ID, the
// slice's ContentID, the resolution, the tuning's fingerprint and RenderVersion.
func NewFrameSetID(plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning) FrameSetID {
	return newFrameSetID(RenderVersion, plan, slice, resolution, tuning)
}

func newFrameSetID(version int, plan CameraPlan, slice GeoSlice, resolution Resolution, tuning RenderTuning) FrameSetID {
	hash := sha256.New()
	writeText := func(text string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		hash.Write(size[:])
		hash.Write([]byte(text))
	}

	writeText("sobrevoo-frames")
	writeText(strconv.Itoa(version))
	writeText(plan.ID())
	writeText(slice.ContentID)
	writeText(strconv.Itoa(resolution.Width))
	writeText(strconv.Itoa(resolution.Height))
	writeText(tuning.Fingerprint())

	return FrameSetID(hex.EncodeToString(hash.Sum(nil)))
}

const (
	frameFilePrefix = "frame_"
	frameFileSuffix = ".png"
	frameFileDigits = 6
)

// FrameFileName is the name of the file of a frame: its number in the plan, in
// six digits — the plan has at most 432 000 frames — so the names sort in the
// order of the frames.
func FrameFileName(index int) string {
	return fmt.Sprintf("%s%0*d%s", frameFilePrefix, frameFileDigits, index, frameFileSuffix)
}

// ParseFrameFileName is the inverse of FrameFileName; ok is false for a name
// that is not exactly frame_ and six digits and .png.
func ParseFrameFileName(name string) (index int, ok bool) {
	if !strings.HasPrefix(name, frameFilePrefix) || !strings.HasSuffix(name, frameFileSuffix) {
		return 0, false
	}
	digits := name[len(frameFilePrefix) : len(name)-len(frameFileSuffix)]
	if len(digits) != frameFileDigits || !allDigits(digits) {
		return 0, false
	}

	index, err := strconv.Atoi(digits)
	return index, err == nil
}

// ParseFrameNumber reads the number of a frame of a plan of frameCount frames.
// The error is ErrFrameOutOfRange, saying what was wrong: text that is not a
// whole number, or a number outside 0 to frameCount-1.
func ParseFrameNumber(text string, frameCount int) (int, error) {
	digits := strings.TrimPrefix(text, "-")
	if !allDigits(digits) {
		return 0, fmt.Errorf("%w: %q is not a whole number", ErrFrameOutOfRange, text)
	}

	number, err := strconv.Atoi(text)
	if err != nil || number < 0 || number >= frameCount {
		return 0, fmt.Errorf("%w: %s is not a frame of the plan, which has frames 0 to %d", ErrFrameOutOfRange, text, frameCount-1)
	}
	return number, nil
}

// SingleFrameRequest asks for one frame of a plan: Number is its index in the
// plan.
type SingleFrameRequest struct {
	Number     int
	Path       string
	Resolution Resolution
	Overwrite  bool
}

// FrameSetRequest asks for the frames of a plan in a directory.
type FrameSetRequest struct {
	Directory  string
	Resolution Resolution
	Overwrite  bool
}

// FrameFile is a file of a directory that is named like a frame.
type FrameFile struct {
	// Index is the number in the file name.
	Index int

	// Ours is true when the file is a PNG this tool drew, which says so inside
	// (SetID is then the set it belongs to); a file that is not is somebody
	// else's, whatever its name.
	Ours  bool
	SetID FrameSetID

	// Complete is true when the file is whole: a PNG whose header has the
	// resolution asked for and that ends where a PNG ends.
	Complete bool
}

// FrameDirectory is what a destination directory holds of frames, sorted by
// number.
type FrameDirectory struct {
	Files []FrameFile
}

// FrameWork is what to do about the frames of a plan in a directory: the
// numbers to keep as they are, to draw, and — of a previous set — to remove.
type FrameWork struct {
	Keep, Draw, Remove []int
}

// Plan says what to do about the frames of a plan of frameCount frames in the
// directory, for the set id.
//
// Without overwrite, it keeps the frames that are this tool's, of this set, whole
// and numbered as the plan numbers them, and draws all the others, in order: that
// is what lets a drawing that was interrupted carry on where it stopped. It
// refuses (ErrFrameSetConflict) a directory that holds a frame of another set —
// whatever its number: the stage that joins the frames into a video would take
// it for one of this flight — or a file named like a frame the plan has that is
// not this tool's, since it can neither be kept nor written over. What is not
// named like a frame, and what is named like one the plan has no number for and
// is not this tool's, is not the plan's to touch.
//
// With overwrite, it draws every frame, keeping none, and removes the frames of
// another set that have a number the plan does not: only what this tool drew,
// which says so inside, and never anything else.
func (d FrameDirectory) Plan(id FrameSetID, frameCount int, overwrite bool) (FrameWork, error) {
	if overwrite {
		return d.overwrite(id, frameCount), nil
	}

	var otherSet, foreign int
	kept := make(map[int]bool, len(d.Files))
	for _, file := range d.Files {
		switch {
		case file.Ours && file.SetID != id:
			otherSet++
		case !file.Ours && file.Index < frameCount:
			foreign++
		case file.Ours && file.Complete && file.Index < frameCount:
			kept[file.Index] = true
		}
	}
	if otherSet > 0 || foreign > 0 {
		return FrameWork{}, conflict(otherSet, foreign)
	}

	var work FrameWork
	for index := 0; index < frameCount; index++ {
		if kept[index] {
			work.Keep = append(work.Keep, index)
		} else {
			work.Draw = append(work.Draw, index)
		}
	}
	return work, nil
}

func (d FrameDirectory) overwrite(id FrameSetID, frameCount int) FrameWork {
	var work FrameWork
	for index := 0; index < frameCount; index++ {
		work.Draw = append(work.Draw, index)
	}
	for _, file := range d.Files {
		if file.Ours && file.SetID != id && file.Index >= frameCount {
			work.Remove = append(work.Remove, file.Index)
		}
	}
	return work
}

// conflict is the error for a directory that cannot be written to without
// overwrite, saying what is in the way and how to go on.
func conflict(otherSet, foreign int) error {
	var causes []string
	if otherSet > 0 {
		causes = append(causes, fmt.Sprintf("%s of another set", countOf(otherSet, "frame is", "frames are")))
	}
	if foreign > 0 {
		causes = append(causes, fmt.Sprintf("%s not this tool's", countOf(foreign, "file named like a frame is", "files named like frames are")))
	}
	return fmt.Errorf("%w: %s; use --overwrite to replace them, or another --output", ErrFrameSetConflict, strings.Join(causes, ", "))
}

func countOf(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
