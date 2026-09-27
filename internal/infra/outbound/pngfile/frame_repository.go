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
func (FrameRepository) Save(dir string, index int, id domain.FrameSetID, image domain.FrameImage) error {
	if err := ensureDirectory(dir); err != nil {
		return err
	}

	data, err := encodeFrame(image, id)
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
// of which set, and whether it is whole for a frame of the resolution.
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

	entries, err := os.ReadDir(dir)
	if err != nil {
		return domain.FrameDirectory{}, fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, dir, err)
	}

	var directory domain.FrameDirectory
	for _, entry := range entries {
		index, ok := domain.ParseFrameFileName(entry.Name())
		if !ok || entry.IsDir() {
			continue
		}
		directory.Files = append(directory.Files, inspectFrame(filepath.Join(dir, entry.Name()), index, resolution))
	}
	sort.Slice(directory.Files, func(i, j int) bool { return directory.Files[i].Index < directory.Files[j].Index })
	return directory, nil
}

// inspectFrame reads the start and the end of a frame file. A file that cannot
// be read is one that is not this tool's, and not whole.
func inspectFrame(path string, index int, resolution domain.Resolution) domain.FrameFile {
	file := domain.FrameFile{Index: index}

	f, err := os.Open(path)
	if err != nil {
		return file
	}
	defer f.Close()

	head := make([]byte, headBytes)
	n, _ := io.ReadFull(f, head)
	head = head[:n]

	if id, ok := readMark(head); ok {
		file.Ours, file.SetID = true, id
	}

	var tail []byte
	if info, err := f.Stat(); err == nil && info.Size() >= tailBytes {
		tail = make([]byte, tailBytes)
		if _, err := f.ReadAt(tail, info.Size()-tailBytes); err != nil {
			tail = nil
		}
	}
	width, height, complete := readInfo(head, tail)
	file.Complete = complete && width == resolution.Width && height == resolution.Height
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
