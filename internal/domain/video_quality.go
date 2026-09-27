package domain

import "fmt"

// VideoQuality is a named quality of a video: it trades the size of the file and
// the time it takes to encode for how faithful the video is to the frames. What
// each one asks of the encoder is up to the encoder's adapter; the domain knows
// only the names and that they go from the least to the most faithful.
type VideoQuality int

const (
	VideoQualityLow VideoQuality = iota
	VideoQualityMedium
	VideoQualityHigh
)

// ParseVideoQuality reads the name of a quality: low, medium or high.
func ParseVideoQuality(text string) (VideoQuality, error) {
	switch text {
	case "low":
		return VideoQualityLow, nil
	case "medium":
		return VideoQualityMedium, nil
	case "high":
		return VideoQualityHigh, nil
	default:
		return 0, fmt.Errorf("%q: use one of low, medium, high", text)
	}
}

// String is the name of the quality.
func (q VideoQuality) String() string {
	switch q {
	case VideoQualityLow:
		return "low"
	case VideoQualityMedium:
		return "medium"
	case VideoQualityHigh:
		return "high"
	default:
		return fmt.Sprintf("VideoQuality(%d)", int(q))
	}
}
