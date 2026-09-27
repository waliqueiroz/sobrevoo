package builddomain

import (
	"sort"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// FrameDirectoryBuilder builds what a destination directory holds of frames:
// an empty directory unless told otherwise.
type FrameDirectoryBuilder struct {
	files map[int]domain.FrameFile
}

func NewFrameDirectoryBuilder() *FrameDirectoryBuilder {
	return &FrameDirectoryBuilder{files: map[int]domain.FrameFile{}}
}

// WithOursFrame adds a whole frame this tool drew, of the given set.
func (b *FrameDirectoryBuilder) WithOursFrame(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	b.files[index] = domain.FrameFile{Index: index, Ours: true, SetID: id, Complete: true}
	return b
}

// WithIncompleteFrame adds a frame this tool drew, of the given set, that is
// not whole (truncated, or of another resolution).
func (b *FrameDirectoryBuilder) WithIncompleteFrame(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	b.files[index] = domain.FrameFile{Index: index, Ours: true, SetID: id}
	return b
}

// WithForeignFrame adds a file named like a frame that this tool did not draw.
func (b *FrameDirectoryBuilder) WithForeignFrame(index int) *FrameDirectoryBuilder {
	b.files[index] = domain.FrameFile{Index: index}
	return b
}

// WithOursFrames adds whole frames this tool drew, of the given set, from
// first to last (both included).
func (b *FrameDirectoryBuilder) WithOursFrames(first, last int, id domain.FrameSetID) *FrameDirectoryBuilder {
	for i := first; i <= last; i++ {
		b.WithOursFrame(i, id)
	}
	return b
}

func (b *FrameDirectoryBuilder) Build() domain.FrameDirectory {
	files := make([]domain.FrameFile, 0, len(b.files))
	for _, file := range b.files {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Index < files[j].Index })
	return domain.FrameDirectory{Files: files}
}
