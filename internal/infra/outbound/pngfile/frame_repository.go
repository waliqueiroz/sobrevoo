package pngfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/atomicfile"
)

// FrameRepository implements domain.FrameRepository: the frames of a flight in a
// directory, one PNG each, named by their number in the plan.
type FrameRepository struct{}

// NewFrameRepository creates a FrameRepository.
func NewFrameRepository() FrameRepository {
	return FrameRepository{}
}

// Save publishes a frame in dir, creating dir when it does not exist (its parent
// has to). A frame of the same number that is there is replaced, all at once.
func (FrameRepository) Save(dir string, index int, mark domain.FrameMark, image domain.FrameImage) error {
	if err := ensureDirectory(dir); err != nil {
		return err
	}

	data, err := encodeFrame(image, mark)
	if err != nil {
		return fmt.Errorf("encoding the frame: %w", err)
	}

	path := filepath.Join(dir, domain.FrameFileName(index))
	err = atomicfile.Publish(path, true, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
	if err != nil {
		return fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, path, err)
	}
	return nil
}

// ensureDirectory makes sure dir is a directory, making it if it is not there.
func ensureDirectory(dir string) error {
	info, err := os.Stat(dir)
	switch {
	case err == nil && !info.IsDir():
		return fmt.Errorf("%w: %s is not a directory", domain.ErrFrameDestinationInvalid, dir)
	case err == nil:
		return nil
	case errors.Is(err, os.ErrNotExist):
		if err := os.Mkdir(dir, 0o755); err != nil {
			return fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, dir, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, dir, err)
	}
}

// Inspect lists the frame files of dir — the files named as a frame — by number,
// and reads off each, from its start and its end alone, whether this tool drew it,
// of which set and plan, its size, and whether it is whole for a frame of the
// resolution. A directory that does not exist has no frames.
func (FrameRepository) Inspect(dir string, resolution domain.Resolution) (domain.FrameDirectory, error) {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return domain.FrameDirectory{}, nil
	case err != nil:
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, dir, err)
	case !info.IsDir():
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s is not a directory", domain.ErrFrameDestinationInvalid, dir)
	}

	files, err := scanFrames(dir)
	if err != nil {
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, dir, err)
	}
	for i := range files {
		files[i].Complete = files[i].Whole && files[i].Width == resolution.Width && files[i].Height == resolution.Height
	}
	return domain.FrameDirectory{Files: files}, nil
}

// List lists the frame files of dir, by number, and reads off each what its image
// says of itself; nothing is asked of a resolution. A directory that is not
// there, is not a directory or cannot be read is an error, since there is nothing
// to join into a video.
func (FrameRepository) List(dir string) (domain.FrameDirectory, error) {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s does not exist", domain.ErrFrameDirectoryInvalid, dir)
	case err != nil:
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s: %w", domain.ErrFrameDirectoryInvalid, dir, err)
	case !info.IsDir():
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s is not a directory", domain.ErrFrameDirectoryInvalid, dir)
	}

	files, err := scanFrames(dir)
	if err != nil {
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s: %w", domain.ErrFrameDirectoryInvalid, dir, err)
	}
	return domain.FrameDirectory{Files: files}, nil
}

// scanFrames reads dir and each file in it that is named exactly as a frame — a
// file, not a directory —, sorted by number.
func scanFrames(dir string) ([]domain.FrameFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []domain.FrameFile
	for _, entry := range entries {
		index, ok := domain.ParseFrameFileName(entry.Name())
		if !ok || entry.IsDir() {
			continue
		}
		files = append(files, inspectFrame(filepath.Join(dir, entry.Name()), index))
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Index < files[j].Index })
	return files, nil
}

// inspectFrame reads the start and the end of a frame file. A file that cannot
// be read is one that is not this tool's, and not whole. Whole and the size say
// what the file is, whatever the resolution it should have.
func inspectFrame(path string, index int) domain.FrameFile {
	file := domain.FrameFile{Index: index}

	f, err := os.Open(path)
	if err != nil {
		return file
	}
	defer f.Close()

	head := make([]byte, headBytes)
	n, _ := io.ReadFull(f, head)
	head = head[:n]

	if mark, ok := readMark(head); ok {
		file.Ours, file.SetID, file.PlanID = true, mark.SetID, mark.PlanID
	}

	var tail []byte
	if info, err := f.Stat(); err == nil && info.Size() >= tailBytes {
		tail = make([]byte, tailBytes)
		if _, err := f.ReadAt(tail, info.Size()-tailBytes); err != nil {
			tail = nil
		}
	}
	file.Width, file.Height, file.Whole = readInfo(head, tail)
	return file
}

// Remove deletes the frame files with the given numbers, and only those files: a
// frame that is not there is not an error.
func (FrameRepository) Remove(dir string, indexes []int) error {
	for _, index := range indexes {
		path := filepath.Join(dir, domain.FrameFileName(index))
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, path, err)
		}
	}
	return nil
}
