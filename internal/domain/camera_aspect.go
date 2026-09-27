package domain

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// MinAspectRatio and MaxAspectRatio bound the accepted width-to-height
	// ratios (inclusive): from 1:5 (very tall) to 5:1 (very wide).
	MinAspectRatio = 0.2
	MaxAspectRatio = 5.0

	// maxAspectTerm bounds each side of an aspect ratio, so it stays a
	// readable "W:H" and never overflows.
	maxAspectTerm = 1000
)

// AspectRatio is the shape of the video the camera plan is made for, as the
// ratio of its width to its height (9:16 is a vertical video). The camera's
// vertical field of view is fixed, so the horizontal one follows the aspect
// ratio: the plan uses it to frame the whole track, in the opening and the
// closing, in both directions.
type AspectRatio struct {
	Width  int
	Height int
}

// LandscapeAspectRatio is 16:9. It is what plans made before the aspect ratio
// existed assumed, since they framed the track by the vertical field of view
// only.
var LandscapeAspectRatio = AspectRatio{Width: 16, Height: 9}

// ParseAspectRatio reads "W:H" (for example "9:16"), with whole positive
// numbers. The error is ErrInvalidAspectRatio and quotes the received text.
func ParseAspectRatio(text string) (AspectRatio, error) {
	widthText, heightText, found := strings.Cut(strings.TrimSpace(text), ":")
	if !found {
		return AspectRatio{}, fmt.Errorf("%w: %q, expected WIDTH:HEIGHT, for example 9:16", ErrInvalidAspectRatio, text)
	}

	width, widthErr := strconv.Atoi(widthText)
	height, heightErr := strconv.Atoi(heightText)
	if widthErr != nil || heightErr != nil {
		return AspectRatio{}, fmt.Errorf("%w: %q, expected WIDTH:HEIGHT with whole numbers, for example 9:16", ErrInvalidAspectRatio, text)
	}

	aspect := AspectRatio{Width: width, Height: height}
	if err := aspect.Validate(); err != nil {
		return AspectRatio{}, err
	}

	return aspect, nil
}

// Validate checks that both sides are between 1 and 1000 and that the ratio is
// between 1:5 and 5:1.
func (a AspectRatio) Validate() error {
	if a.Width < 1 || a.Height < 1 || a.Width > maxAspectTerm || a.Height > maxAspectTerm {
		return fmt.Errorf("%w: %s, each side must be a whole number from 1 to %d", ErrInvalidAspectRatio, a, maxAspectTerm)
	}

	if ratio := a.Ratio(); ratio < MinAspectRatio || ratio > MaxAspectRatio {
		return fmt.Errorf("%w: %s, the width to height ratio must be between 1:5 and 5:1", ErrInvalidAspectRatio, a)
	}

	return nil
}

// Ratio is the width divided by the height.
func (a AspectRatio) Ratio() float64 {
	return float64(a.Width) / float64(a.Height)
}

// String is the "W:H" text.
func (a AspectRatio) String() string {
	return fmt.Sprintf("%d:%d", a.Width, a.Height)
}
