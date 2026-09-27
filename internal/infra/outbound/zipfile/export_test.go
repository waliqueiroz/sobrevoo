package zipfile

// SetMaxSliceBytes lets a test lower the largest slice file the reader accepts,
// and returns what puts it back.
func SetMaxSliceBytes(limit int64) (restore func()) {
	previous := maxSliceBytes
	maxSliceBytes = limit
	return func() { maxSliceBytes = previous }
}
