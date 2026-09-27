package videofile_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/videofile"
)

// skipWithoutPermissions skips a test that needs a directory that cannot be
// written, which a superuser (or Windows) always can.
func skipWithoutPermissions(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permissions do not restrict this user")
	}
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	list := make([]string, len(entries))
	for i, entry := range entries {
		list[i] = entry.Name()
	}
	return list
}

// writing is what an encoder does: write the video where it is told.
func writing(content string) func(string) error {
	return func(temporary string) error { return os.WriteFile(temporary, []byte(content), 0o600) }
}

func Test_VideoExporter_Export(t *testing.T) {
	t.Run("should publish what was written as the destination and say its size", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "flight.mp4")

		// when
		size, err := videofile.NewVideoExporter().Export(path, false, writing("twelve bytes"))

		// then
		require.NoError(t, err)
		assert.Equal(t, int64(12), size)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "twelve bytes", string(content))
		assert.Equal(t, []string{"flight.mp4"}, names(t, dir))
	})

	t.Run("should give the encoder the absolute path of a file in the folder of the destination, other than the destination", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "flight.mp4")
		var temporary string

		// when
		_, err := videofile.NewVideoExporter().Export(path, false, func(given string) error {
			temporary = given
			return os.WriteFile(given, []byte("video"), 0o600)
		})

		// then
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(temporary))
		assert.Equal(t, dir, filepath.Dir(temporary))
		assert.NotEqual(t, path, temporary)
	})

	t.Run("should give the encoder an absolute path even when the destination is relative", func(t *testing.T) {
		// given
		dir := t.TempDir()
		t.Chdir(dir)
		var temporary string

		// when
		_, err := videofile.NewVideoExporter().Export("flight.mp4", false, func(given string) error {
			temporary = given
			return os.WriteFile(given, []byte("video"), 0o600)
		})

		// then
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(temporary))
		assert.FileExists(t, filepath.Join(dir, "flight.mp4"))
	})

	t.Run("should refuse a destination that exists without overwrite, saying how to replace it, and leave it as it is", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "flight.mp4")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		_, err := videofile.NewVideoExporter().Export(path, false, writing("new"))

		// then
		require.ErrorIs(t, err, domain.ErrVideoDestinationExists)
		assert.EqualError(t, err, "video destination already exists: "+path+"; use --overwrite to replace it")
		content, _ := os.ReadFile(path)
		assert.Equal(t, "precious", string(content))
		assert.Equal(t, []string{"flight.mp4"}, names(t, dir))
	})

	t.Run("should replace a destination that exists all at once with overwrite", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "flight.mp4")
		require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

		// when
		size, err := videofile.NewVideoExporter().Export(path, true, writing("newer"))

		// then
		require.NoError(t, err)
		assert.Equal(t, int64(5), size)
		content, _ := os.ReadFile(path)
		assert.Equal(t, "newer", string(content))
		assert.Equal(t, []string{"flight.mp4"}, names(t, dir))
	})

	t.Run("should return the error of the encoder as it is, and leave nothing behind", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "flight.mp4")
		failure := errors.New("the encoder failed")

		// when
		_, err := videofile.NewVideoExporter().Export(path, false, func(temporary string) error {
			require.NoError(t, os.WriteFile(temporary, []byte("half"), 0o600))
			return failure
		})

		// then
		assert.Same(t, failure, err)
		assert.NotErrorIs(t, err, domain.ErrVideoDestinationInvalid)
		assert.Empty(t, names(t, dir))
	})

	t.Run("should keep the previous video when the encoder fails with overwrite", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "flight.mp4")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		_, err := videofile.NewVideoExporter().Export(path, true, func(temporary string) error {
			require.NoError(t, os.WriteFile(temporary, []byte("half"), 0o600))
			return errors.New("the encoder failed")
		})

		// then
		require.Error(t, err)
		content, _ := os.ReadFile(path)
		assert.Equal(t, "precious", string(content))
		assert.Equal(t, []string{"flight.mp4"}, names(t, dir))
	})

	t.Run("should refuse a destination whose folder does not exist", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "missing", "flight.mp4")
		called := false

		// when
		_, err := videofile.NewVideoExporter().Export(path, false, func(string) error {
			called = true
			return nil
		})

		// then
		require.ErrorIs(t, err, domain.ErrVideoDestinationInvalid)
		assert.False(t, called)
	})

	t.Run("should publish the video readable by everyone", func(t *testing.T) {
		// given
		if runtime.GOOS == "windows" {
			t.Skip("permission bits are not the same on Windows")
		}
		path := filepath.Join(t.TempDir(), "flight.mp4")

		// when
		_, err := videofile.NewVideoExporter().Export(path, false, writing("video"))

		// then
		require.NoError(t, err)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})
}

func Test_VideoExporter_Check(t *testing.T) {
	t.Run("should accept a file that does not exist in a folder that does", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		err := videofile.NewVideoExporter().Check(filepath.Join(dir, "flight.mp4"), false)

		// then
		assert.NoError(t, err)
	})

	t.Run("should refuse a file that exists without overwrite, saying how to replace it", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "flight.mp4")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := videofile.NewVideoExporter().Check(path, false)

		// then
		require.ErrorIs(t, err, domain.ErrVideoDestinationExists)
		assert.EqualError(t, err, "video destination already exists: "+path+"; use --overwrite to replace it")
	})

	t.Run("should accept a file that exists with overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "flight.mp4")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o600))

		// when
		err := videofile.NewVideoExporter().Check(path, true)

		// then
		assert.NoError(t, err)
	})

	t.Run("should refuse a path that is a directory, with or without overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "flight.mp4")
		require.NoError(t, os.Mkdir(path, 0o755))

		// when
		without := videofile.NewVideoExporter().Check(path, false)
		with := videofile.NewVideoExporter().Check(path, true)

		// then
		require.ErrorIs(t, without, domain.ErrVideoDestinationInvalid)
		assert.ErrorContains(t, without, "is a directory")
		assert.ErrorIs(t, with, domain.ErrVideoDestinationInvalid)
	})

	t.Run("should refuse a file whose folder does not exist, saying so", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "missing", "flight.mp4")

		// when
		err := videofile.NewVideoExporter().Check(path, false)

		// then
		require.ErrorIs(t, err, domain.ErrVideoDestinationInvalid)
		assert.ErrorContains(t, err, "the folder does not exist")
	})

	t.Run("should refuse a file whose folder is a file", func(t *testing.T) {
		// given
		folder := filepath.Join(t.TempDir(), "folder")
		require.NoError(t, os.WriteFile(folder, []byte("x"), 0o600))

		// when
		err := videofile.NewVideoExporter().Check(filepath.Join(folder, "flight.mp4"), false)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoDestinationInvalid)
	})

	t.Run("should refuse a folder it cannot write to", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		// when
		err := videofile.NewVideoExporter().Check(filepath.Join(dir, "flight.mp4"), false)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoDestinationInvalid)
	})

	t.Run("should leave nothing behind, not even a temporary file", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		err := videofile.NewVideoExporter().Check(filepath.Join(dir, "flight.mp4"), false)

		// then
		require.NoError(t, err)
		assert.Empty(t, names(t, dir))
	})

	t.Run("should judge a relative path against the working directory", func(t *testing.T) {
		// given
		dir := t.TempDir()
		t.Chdir(dir)
		require.NoError(t, os.WriteFile("flight.mp4", []byte("precious"), 0o600))

		// when
		err := videofile.NewVideoExporter().Check("flight.mp4", false)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoDestinationExists)
	})
}
