package builddomain

import (
	"sort"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// FrameDirectoryBuilder builds what a directory holds of frames: an empty
// directory unless told otherwise. The frames it adds are 1080 × 1920 and say
// they were drawn from the plan given to WithPlanID (none unless it is).
type FrameDirectoryBuilder struct {
	files      []domain.FrameFile
	planID     string
	resolution domain.Resolution
}

func NewFrameDirectoryBuilder() *FrameDirectoryBuilder {
	return &FrameDirectoryBuilder{resolution: domain.Resolution{Width: 1080, Height: 1920}}
}

// WithPlanID sets the plan the frames added from now on say they were drawn from.
func (b *FrameDirectoryBuilder) WithPlanID(planID string) *FrameDirectoryBuilder {
	b.planID = planID
	return b
}

// WithResolution sets the size of the frames added from now on.
func (b *FrameDirectoryBuilder) WithResolution(resolution domain.Resolution) *FrameDirectoryBuilder {
	b.resolution = resolution
	return b
}

// put adds a file, replacing what the directory had for that number.
func (b *FrameDirectoryBuilder) put(file domain.FrameFile) *FrameDirectoryBuilder {
	kept := b.files[:0]
	for _, existing := range b.files {
		if existing.Index != file.Index {
			kept = append(kept, existing)
		}
	}
	b.files = append(kept, file)
	return b
}

// ours is a whole frame this tool drew, of the given set and of the size and
// plan the builder has now.
func (b *FrameDirectoryBuilder) ours(index int, id domain.FrameSetID) domain.FrameFile {
	return domain.FrameFile{
		Index:    index,
		Ours:     true,
		SetID:    id,
		PlanID:   b.planID,
		Width:    b.resolution.Width,
		Height:   b.resolution.Height,
		Whole:    true,
		Complete: true,
	}
}

// WithOursFrame adds a whole frame this tool drew, of the given set.
func (b *FrameDirectoryBuilder) WithOursFrame(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	return b.put(b.ours(index, id))
}

// WithOursFrames adds whole frames this tool drew, of the given set, from
// first to last (both included).
func (b *FrameDirectoryBuilder) WithOursFrames(first, last int, id domain.FrameSetID) *FrameDirectoryBuilder {
	for i := first; i <= last; i++ {
		b.WithOursFrame(i, id)
	}
	return b
}

// WithIncompleteFrame adds a frame this tool drew, of the given set, that is
// not whole (truncated, or of another resolution).
func (b *FrameDirectoryBuilder) WithIncompleteFrame(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	file := b.ours(index, id)
	file.Whole, file.Complete = false, false
	return b.put(file)
}

// WithTruncatedFrame adds a frame this tool drew, of the given set, that was cut
// short: its header still says its size, but it does not end as a PNG does.
func (b *FrameDirectoryBuilder) WithTruncatedFrame(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	return b.WithIncompleteFrame(index, id)
}

// WithForeignFrame adds a file named like a frame that this tool did not draw.
func (b *FrameDirectoryBuilder) WithForeignFrame(index int) *FrameDirectoryBuilder {
	return b.put(domain.FrameFile{Index: index})
}

// WithFrameOfPlan adds a whole frame of the given set that says it was drawn
// from the given plan, whatever the builder's plan is.
func (b *FrameDirectoryBuilder) WithFrameOfPlan(index int, id domain.FrameSetID, planID string) *FrameDirectoryBuilder {
	file := b.ours(index, id)
	file.PlanID = planID
	return b.put(file)
}

// WithFrameWithoutPlan adds a whole frame of the given set that does not say
// which plan it was drawn from: one drawn before frames did.
func (b *FrameDirectoryBuilder) WithFrameWithoutPlan(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	file := b.ours(index, id)
	file.PlanID = ""
	return b.put(file)
}

// WithFrameOfResolution adds a whole frame of the given set and of the given
// size, whatever the builder's is.
func (b *FrameDirectoryBuilder) WithFrameOfResolution(index int, id domain.FrameSetID, resolution domain.Resolution) *FrameDirectoryBuilder {
	file := b.ours(index, id)
	file.Width, file.Height = resolution.Width, resolution.Height
	return b.put(file)
}

// WithRepeatedFrame adds a second file for a number, of the given set: a
// listing that repeats a number.
func (b *FrameDirectoryBuilder) WithRepeatedFrame(index int, id domain.FrameSetID) *FrameDirectoryBuilder {
	b.files = append(b.files, b.ours(index, id))
	return b
}

func (b *FrameDirectoryBuilder) Build() domain.FrameDirectory {
	files := append([]domain.FrameFile(nil), b.files...)
	sort.SliceStable(files, func(i, j int) bool { return files[i].Index < files[j].Index })
	return domain.FrameDirectory{Files: files}
}
