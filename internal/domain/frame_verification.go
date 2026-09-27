package domain

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// maxNumberRanges is how many ranges of frame numbers a message lists, and
	// maxListedFiles how many frame files: what is left out is counted.
	maxNumberRanges = 10
	maxListedFiles  = 5

	// shortIDLength is how many characters of an identification a message shows.
	shortIDLength = 12
)

// Verify checks that the frames the directory lists can be joined into the video
// of plan, and gives their resolution. Only the frames this tool drew count: what
// else is named like a frame is somebody else's, and is neither missing, nor
// repeated, nor left over.
//
// It stops at the first kind of problem, from the one that says most to the one
// that only makes sense once the others are out of the way: whether there are
// frames of this tool at all (ErrFrameDirectoryInvalid); whether they say which
// plan they were drawn from (ErrFramesWithoutPlanID) and whether it is the
// plan's (ErrFramesDoNotMatchPlan); whether they have the same resolution, an
// even one, which the video needs (ErrFrameResolutionInvalid) — before whether
// they belong to one set, because the set includes the resolution, and frames of
// a drawing that was interrupted at another resolution should be told apart by
// that —; whether they belong to one set (ErrFramesDoNotMatchPlan); whether
// there is one, and only one, for each number of the plan, from 0 on
// (ErrFrameSequenceInvalid); and whether each is a whole image
// (ErrFrameFileInvalid). Each message says exactly what is wrong.
func (d FrameDirectory) Verify(plan CameraPlan) (Resolution, error) {
	var ours, foreign []FrameFile
	for _, file := range d.Files {
		if file.Ours {
			ours = append(ours, file)
		} else {
			foreign = append(foreign, file)
		}
	}
	if len(ours) == 0 {
		return Resolution{}, noFramesError(len(foreign))
	}

	if err := verifyPlanIdentification(ours, plan); err != nil {
		return Resolution{}, err
	}
	resolution, err := verifyResolution(ours)
	if err != nil {
		return Resolution{}, err
	}
	if err := verifySingleSet(ours); err != nil {
		return Resolution{}, err
	}
	if err := verifyNumbering(ours, foreign, len(plan.Frames)); err != nil {
		return Resolution{}, err
	}
	if err := verifyWhole(ours); err != nil {
		return Resolution{}, err
	}

	return resolution, nil
}

func noFramesError(ignored int) error {
	var note string
	switch ignored {
	case 0:
	case 1:
		note = " (1 file named like a frame was ignored: it does not carry this tool's identification)"
	default:
		note = fmt.Sprintf(" (%d files named like frames were ignored: they do not carry this tool's identification)", ignored)
	}
	return fmt.Errorf("%w: holds no frames drawn by this tool%s; draw them with \"render all\"", ErrFrameDirectoryInvalid, note)
}

func verifyPlanIdentification(ours []FrameFile, plan CameraPlan) error {
	planID := plan.ID()

	var without, another int
	var anotherID string
	for _, file := range ours {
		switch {
		case file.PlanID == "":
			without++
		case file.PlanID != planID:
			another++
			if anotherID == "" {
				anotherID = file.PlanID
			}
		}
	}

	if without > 0 {
		return fmt.Errorf("%w: %d of %d frames were drawn by an earlier version of Sobrevoo; draw them again with \"render all --overwrite\"",
			ErrFramesWithoutPlanID, without, len(ours))
	}
	if another > 0 {
		return fmt.Errorf("%w: %d of %d frames carry another plan identification (plan %s, frames %s)",
			ErrFramesDoNotMatchPlan, another, len(ours), shortID(planID), shortID(anotherID))
	}
	return nil
}

// shortID is the start of an identification, enough to tell two apart.
func shortID(id string) string {
	if len(id) > shortIDLength {
		return id[:shortIDLength]
	}
	return id
}

// verifyResolution checks that the frames are of one resolution, and an even one,
// and gives it. When they are not, the resolution to expect is the one of most of
// them, or, when two have as many frames, the one of the lowest numbered.
func verifyResolution(ours []FrameFile) (Resolution, error) {
	type tally struct {
		resolution Resolution
		frames     []FrameFile
	}

	var tallies []*tally
	for _, file := range ours {
		resolution := Resolution{Width: file.Width, Height: file.Height}
		var found *tally
		for _, candidate := range tallies {
			if candidate.resolution == resolution {
				found = candidate
				break
			}
		}
		if found == nil {
			found = &tally{resolution: resolution}
			tallies = append(tallies, found)
		}
		found.frames = append(found.frames, file)
	}

	expected := tallies[0]
	for _, candidate := range tallies[1:] {
		if len(candidate.frames) > len(expected.frames) {
			expected = candidate
		}
	}

	if len(tallies) > 1 {
		var differing []string
		for _, candidate := range tallies {
			if candidate == expected {
				continue
			}
			for _, file := range candidate.frames {
				differing = append(differing, fmt.Sprintf("%s is %dx%d", FrameFileName(file.Index), file.Width, file.Height))
			}
		}
		// The frames are told in the order of their numbers, whatever the
		// resolution they differ in.
		sort.Slice(differing, func(i, j int) bool { return differing[i] < differing[j] })

		verb := "differ"
		if len(differing) == 1 {
			verb = "differs"
		}
		return Resolution{}, fmt.Errorf("%w: %dx%d is the resolution of %d frames, but %d %s (%s)",
			ErrFrameResolutionInvalid, expected.resolution.Width, expected.resolution.Height, len(expected.frames), len(differing), verb, listed(differing, maxListedFiles))
	}

	if expected.resolution.Width%2 != 0 || expected.resolution.Height%2 != 0 {
		return Resolution{}, fmt.Errorf("%w: the frames are %dx%d, but width and height must be even for the video",
			ErrFrameResolutionInvalid, expected.resolution.Width, expected.resolution.Height)
	}
	return expected.resolution, nil
}

func verifySingleSet(ours []FrameFile) error {
	var sets []FrameSetID
	for _, file := range ours {
		known := false
		for _, set := range sets {
			if set == file.SetID {
				known = true
				break
			}
		}
		if !known {
			sets = append(sets, file.SetID)
		}
	}

	if len(sets) > 1 {
		return fmt.Errorf("%w: the directory mixes frames of %d different sets (another slice, resolution or drawing version); draw them again into an empty directory",
			ErrFramesDoNotMatchPlan, len(sets))
	}
	return nil
}

// verifyNumbering checks that there is one frame, and only one, for each number
// from 0 to frameCount-1.
func verifyNumbering(ours, foreign []FrameFile, frameCount int) error {
	counts := make(map[int]int, len(ours))
	for _, file := range ours {
		counts[file.Index]++
	}

	var missing, beyond, repeated []int
	for index := 0; index < frameCount; index++ {
		if counts[index] == 0 {
			missing = append(missing, index)
		}
	}
	for index, count := range counts {
		if index >= frameCount {
			beyond = append(beyond, index)
		}
		if count > 1 {
			repeated = append(repeated, index)
		}
	}
	if len(missing) == 0 && len(beyond) == 0 && len(repeated) == 0 {
		return nil
	}
	sort.Ints(beyond)
	sort.Ints(repeated)

	var parts []string
	if len(missing) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing (%s)", len(missing), numberRanges(missing, maxNumberRanges)))
	}
	if len(beyond) > 0 {
		parts = append(parts, fmt.Sprintf("%d not in the plan (%s; the plan has frames 0 to %d)", len(beyond), numberRanges(beyond, maxNumberRanges), frameCount-1))
	}
	switch len(repeated) {
	case 0:
	case 1:
		parts = append(parts, fmt.Sprintf("number %d appears more than once", repeated[0]))
	default:
		parts = append(parts, fmt.Sprintf("%d numbers appear more than once (%s)", len(repeated), numberRanges(repeated, maxNumberRanges)))
	}
	message := strings.Join(parts, ", ")

	if ignored := ignoredInPlaceOf(missing, foreign); len(ignored) > 0 {
		if len(ignored) == 1 {
			message += fmt.Sprintf("; %s was ignored: it does not carry this tool's identification", ignored[0])
		} else {
			message += fmt.Sprintf("; %s were ignored: they do not carry this tool's identification", listed(ignored, maxListedFiles))
		}
	}
	if len(missing) > 0 {
		message += `; draw the missing frames with "render all"`
	}
	return fmt.Errorf("%w: %s", ErrFrameSequenceInvalid, message)
}

// ignoredInPlaceOf is the names of the frame files that are not this tool's and
// are where a frame that is missing should be.
func ignoredInPlaceOf(missing []int, foreign []FrameFile) []string {
	lacking := make(map[int]bool, len(missing))
	for _, index := range missing {
		lacking[index] = true
	}

	var names []string
	for _, file := range foreign {
		if lacking[file.Index] {
			names = append(names, FrameFileName(file.Index))
		}
	}
	return names
}

func verifyWhole(ours []FrameFile) error {
	var broken []string
	for _, file := range ours {
		if !file.Whole {
			broken = append(broken, FrameFileName(file.Index))
		}
	}

	switch len(broken) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("%w: %s is not a whole PNG image (truncated?); draw it again with \"render all\"", ErrFrameFileInvalid, broken[0])
	default:
		return fmt.Errorf("%w: %d frames are not whole PNG images (truncated?): %s; draw them again with \"render all\"", ErrFrameFileInvalid, len(broken), listed(broken, maxListedFiles))
	}
}

// numberRanges writes ascending numbers as ranges — "0-3, 7, 10-12" —, at most
// maxRanges of them, and counts the numbers of the ranges left out.
func numberRanges(numbers []int, maxRanges int) string {
	type span struct{ from, to int }

	var spans []span
	for _, number := range numbers {
		if last := len(spans) - 1; last >= 0 && spans[last].to == number-1 {
			spans[last].to = number
			continue
		}
		spans = append(spans, span{from: number, to: number})
	}

	shown, hidden := spans, 0
	if len(spans) > maxRanges {
		for _, left := range spans[maxRanges:] {
			hidden += left.to - left.from + 1
		}
		shown = spans[:maxRanges]
	}

	parts := make([]string, 0, len(shown)+1)
	for _, one := range shown {
		if one.from == one.to {
			parts = append(parts, fmt.Sprintf("%d", one.from))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", one.from, one.to))
		}
	}
	if hidden > 0 {
		parts = append(parts, fmt.Sprintf("and %d more", hidden))
	}
	return strings.Join(parts, ", ")
}

// listed joins at most limit of the items, and counts the ones left out.
func listed(items []string, limit int) string {
	if len(items) <= limit {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:limit], ", ") + fmt.Sprintf(", and %d more", len(items)-limit)
}
