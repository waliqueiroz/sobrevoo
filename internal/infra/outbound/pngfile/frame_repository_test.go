package pngfile_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/pngfile"
	"github.com/waliqueiroz/sobrevoo/test/helper"
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

func Test_FrameRepository_Save(t *testing.T) {
	t.Run("should write the frame as a file named by its number, the same bytes the exporter writes", func(t *testing.T) {
		// given
		dir := t.TempDir()
		reference := filepath.Join(t.TempDir(), "reference.png")
		require.NoError(t, pngfile.NewFrameExporter().Export(drawnFrame(), frameMark, reference, false))

		// when
		err := pngfile.NewFrameRepository().Save(dir, 7, frameMark, drawnFrame())

		// then
		require.NoError(t, err)
		saved, readErr := os.ReadFile(filepath.Join(dir, "frame_000007.png"))
		require.NoError(t, readErr)
		want, _ := os.ReadFile(reference)
		assert.Equal(t, want, saved)
		decoded, decodeErr := png.Decode(bytes.NewReader(saved))
		require.NoError(t, decodeErr)
		assert.Equal(t, 8, decoded.Bounds().Dx())
	})

	t.Run("should create the directory when it does not exist, and its parent does", func(t *testing.T) {
		// given
		dir := filepath.Join(t.TempDir(), "frames")

		// when
		err := pngfile.NewFrameRepository().Save(dir, 0, frameMark, drawnFrame())

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"frame_000000.png"}, names(t, dir))
	})

	t.Run("should refuse a directory whose parent does not exist", func(t *testing.T) {
		// given
		dir := filepath.Join(t.TempDir(), "missing", "frames")

		// when
		err := pngfile.NewFrameRepository().Save(dir, 0, frameMark, drawnFrame())

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
	})

	t.Run("should refuse a path that exists and is not a directory, leaving it as it is", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "file.txt")
		require.NoError(t, os.WriteFile(path, []byte("mine"), 0o600))

		// when
		err := pngfile.NewFrameRepository().Save(path, 0, frameMark, drawnFrame())

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
		content, _ := os.ReadFile(path)
		assert.Equal(t, "mine", string(content))
	})

	t.Run("should refuse a directory it cannot write to", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		// when
		err := pngfile.NewFrameRepository().Save(dir, 0, frameMark, drawnFrame())

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
	})

	t.Run("should replace a frame that is already there, leaving no temporary file", func(t *testing.T) {
		// given
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000003.png"), []byte("old"), 0o600))

		// when
		err := pngfile.NewFrameRepository().Save(dir, 3, frameMark, drawnFrame())

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"frame_000003.png"}, names(t, dir))
		content, _ := os.ReadFile(filepath.Join(dir, "frame_000003.png"))
		assert.NotEqual(t, "old", string(content))
	})

	t.Run("should save frames in any order, all by their own numbers, in the order of the names", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()

		// when
		for _, index := range []int{2, 0, 10, 1} {
			require.NoError(t, repository.Save(dir, index, frameMark, drawnFrame()))
		}

		// then
		assert.Equal(t, []string{"frame_000000.png", "frame_000001.png", "frame_000002.png", "frame_000010.png"}, names(t, dir))
	})
}

func Test_FrameRepository_Inspect(t *testing.T) {
	resolution := domain.Resolution{Width: 8, Height: 6}
	otherSet := domain.FrameSetID("ff" + string(setID)[2:])

	t.Run("should find nothing in a directory that does not exist", func(t *testing.T) {
		// given
		dir := filepath.Join(t.TempDir(), "nope")

		// when
		directory, err := pngfile.NewFrameRepository().Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		assert.Empty(t, directory.Files)
	})

	t.Run("should refuse a path that is not a directory", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "file.txt")
		require.NoError(t, os.WriteFile(path, []byte("mine"), 0o600))

		// when
		_, err := pngfile.NewFrameRepository().Inspect(path, resolution)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
	})

	t.Run("should list only the files named as frames, by number, and none of the rest", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		for _, index := range []int{10, 2, 0} {
			require.NoError(t, repository.Save(dir, index, frameMark, drawnFrame()))
		}
		for _, name := range []string{"notes.txt", "frame_1.png", ".sobrevoo-x.tmp", "frame_000003.jpg", "frame_0000004.png"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
		}
		require.NoError(t, os.Mkdir(filepath.Join(dir, "frame_000005.png"), 0o755))

		// when
		directory, err := repository.Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		indexes := make([]int, len(directory.Files))
		for i, file := range directory.Files {
			indexes[i] = file.Index
		}
		assert.Equal(t, []int{0, 2, 10}, indexes)
	})

	t.Run("should say a frame this tool drew is its own, of which set, and whole", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 4, frameMark, drawnFrame()))

		// when
		directory, err := repository.Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 1)
		assert.Equal(t, domain.FrameFile{Index: 4, Ours: true, SetID: setID, PlanID: planID, Width: 8, Height: 6, Whole: true, Complete: true}, directory.Files[0])
	})

	t.Run("should say which plan a frame was drawn from, and its size, whatever the resolution asked", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))

		// when
		directory, err := repository.Inspect(dir, domain.Resolution{Width: 16, Height: 12})

		// then
		require.NoError(t, err)
		file := directory.Files[0]
		assert.Equal(t, planID, file.PlanID)
		assert.Equal(t, 8, file.Width)
		assert.Equal(t, 6, file.Height)
		assert.True(t, file.Whole)
		assert.False(t, file.Complete)
	})

	t.Run("should say a frame drawn before the plan was written is its own, with no plan", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))
		path := filepath.Join(dir, "frame_000000.png")
		data, _ := os.ReadFile(path)
		require.NoError(t, os.WriteFile(path, helper.WithoutPlanMark(data), 0o600))

		// when
		directory, err := repository.Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		file := directory.Files[0]
		assert.True(t, file.Ours)
		assert.Equal(t, setID, file.SetID)
		assert.Empty(t, file.PlanID)
		assert.True(t, file.Complete)
	})

	t.Run("should tell the set of each frame apart", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))
		require.NoError(t, repository.Save(dir, 1, domain.FrameMark{SetID: otherSet, PlanID: planID}, drawnFrame()))

		// when
		directory, err := repository.Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 2)
		assert.Equal(t, setID, directory.Files[0].SetID)
		assert.Equal(t, otherSet, directory.Files[1].SetID)
	})

	t.Run("should say a frame that was cut short is its own but not whole", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))
		path := filepath.Join(dir, "frame_000000.png")
		data, _ := os.ReadFile(path)
		require.NoError(t, os.WriteFile(path, data[:len(data)-5], 0o600))

		// when
		directory, err := repository.Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 1)
		assert.True(t, directory.Files[0].Ours)
		assert.Equal(t, setID, directory.Files[0].SetID)
		assert.False(t, directory.Files[0].Complete)
		assert.False(t, directory.Files[0].Whole)
		assert.Equal(t, 8, directory.Files[0].Width, "the header is still readable")
	})

	t.Run("should say a frame of another size is not whole", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))

		// when
		directory, err := repository.Inspect(dir, domain.Resolution{Width: 16, Height: 12})

		// then
		require.NoError(t, err)
		assert.False(t, directory.Files[0].Complete)
		assert.True(t, directory.Files[0].Ours)
	})

	t.Run("should say a PNG of another tool, and a file with no content, are not its own", func(t *testing.T) {
		// given
		dir := t.TempDir()
		var foreign bytes.Buffer
		require.NoError(t, png.Encode(&foreign, image.NewNRGBA(image.Rect(0, 0, 8, 6))))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000000.png"), foreign.Bytes(), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000001.png"), nil, 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000002.png"), []byte("not a png"), 0o600))

		// when
		directory, err := pngfile.NewFrameRepository().Inspect(dir, resolution)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 3)
		for _, file := range directory.Files {
			assert.False(t, file.Ours, "frame %d", file.Index)
			assert.Empty(t, file.SetID)
		}
		assert.False(t, directory.Files[1].Complete)
		assert.False(t, directory.Files[2].Complete)
	})
}

func Test_FrameRepository_Remove(t *testing.T) {
	t.Run("should delete the frames with the numbers given, and nothing else", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		for _, index := range []int{3, 4, 9} {
			require.NoError(t, repository.Save(dir, index, frameMark, drawnFrame()))
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_1.png"), []byte("mine"), 0o600))

		// when
		err := repository.Remove(dir, []int{3, 9})

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"frame_000004.png", "frame_1.png", "notes.txt"}, names(t, dir))
	})

	t.Run("should not mind a frame that is not there", func(t *testing.T) {
		// given
		dir := t.TempDir()
		require.NoError(t, pngfile.NewFrameRepository().Save(dir, 1, frameMark, drawnFrame()))

		// when
		err := pngfile.NewFrameRepository().Remove(dir, []int{7, 8})

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"frame_000001.png"}, names(t, dir))
	})

	t.Run("should do nothing for no numbers", func(t *testing.T) {
		// given
		dir := t.TempDir()
		require.NoError(t, pngfile.NewFrameRepository().Save(dir, 1, frameMark, drawnFrame()))

		// when
		err := pngfile.NewFrameRepository().Remove(dir, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"frame_000001.png"}, names(t, dir))
	})

	t.Run("should refuse to delete from a directory it cannot write to", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		require.NoError(t, pngfile.NewFrameRepository().Save(dir, 2, frameMark, drawnFrame()))
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		// when
		err := pngfile.NewFrameRepository().Remove(dir, []int{2})

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
	})
}

func Test_FrameRepository_Save_Failure(t *testing.T) {
	t.Run("should keep the frame that was there, whole, when the new one cannot be written", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 5, frameMark, drawnFrame()))
		before, _ := os.ReadFile(filepath.Join(dir, "frame_000005.png"))
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		// when
		err := repository.Save(dir, 5, domain.FrameMark{SetID: "another-set", PlanID: planID}, drawnFrame())

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationInvalid)
		require.NoError(t, os.Chmod(dir, 0o700))
		after, _ := os.ReadFile(filepath.Join(dir, "frame_000005.png"))
		assert.Equal(t, before, after)
		assert.Equal(t, []string{"frame_000005.png"}, names(t, dir))
	})
}

func Test_FrameRepository_List(t *testing.T) {
	t.Run("should list only the files named exactly as frames, by number, and none of the rest", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		for _, index := range []int{3, 0, 12} {
			require.NoError(t, repository.Save(dir, index, frameMark, drawnFrame()))
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover.png"), []byte("cover"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_3.png"), []byte("short number"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_0000003.png"), []byte("long number"), 0o600))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "frame_000005.png"), 0o755))

		// when
		directory, err := repository.List(dir)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 3)
		assert.Equal(t, []int{0, 3, 12}, []int{directory.Files[0].Index, directory.Files[1].Index, directory.Files[2].Index})
	})

	t.Run("should say for each frame this tool drew which set and plan it says it is of, its size and that it is whole", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 4, frameMark, drawnFrame()))

		// when
		directory, err := repository.List(dir)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 1)
		assert.Equal(t, domain.FrameFile{Index: 4, Ours: true, SetID: setID, PlanID: planID, Width: 8, Height: 6, Whole: true}, directory.Files[0])
	})

	t.Run("should not say whether a frame is of a resolution, which is not asked", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))

		// when
		directory, err := repository.List(dir)

		// then
		require.NoError(t, err)
		assert.False(t, directory.Files[0].Complete)
	})

	t.Run("should say a frame that was cut short is its own, still with its size, but not whole", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))
		path := filepath.Join(dir, "frame_000000.png")
		data, _ := os.ReadFile(path)
		require.NoError(t, os.WriteFile(path, helper.TruncatedAt(data, len(data)-5), 0o600))

		// when
		directory, err := repository.List(dir)

		// then
		require.NoError(t, err)
		file := directory.Files[0]
		assert.True(t, file.Ours)
		assert.False(t, file.Whole)
		assert.Equal(t, 8, file.Width)
		assert.Equal(t, 6, file.Height)
	})

	t.Run("should say a frame drawn before frames said the plan is its own, with no plan", func(t *testing.T) {
		// given
		dir := t.TempDir()
		repository := pngfile.NewFrameRepository()
		require.NoError(t, repository.Save(dir, 0, frameMark, drawnFrame()))
		path := filepath.Join(dir, "frame_000000.png")
		data, _ := os.ReadFile(path)
		require.NoError(t, os.WriteFile(path, helper.WithoutPlanMark(data), 0o600))

		// when
		directory, err := repository.List(dir)

		// then
		require.NoError(t, err)
		assert.True(t, directory.Files[0].Ours)
		assert.Equal(t, setID, directory.Files[0].SetID)
		assert.Empty(t, directory.Files[0].PlanID)
		assert.True(t, directory.Files[0].Whole)
	})

	t.Run("should say a file named like a frame that is not a PNG of this tool is not its own", func(t *testing.T) {
		// given
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000001.png"), []byte("this is not an image"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000002.png"), nil, 0o600))
		var other bytes.Buffer
		require.NoError(t, png.Encode(&other, image.NewNRGBA(image.Rect(0, 0, 4, 4))))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "frame_000003.png"), other.Bytes(), 0o600))

		// when
		directory, err := pngfile.NewFrameRepository().List(dir)

		// then
		require.NoError(t, err)
		require.Len(t, directory.Files, 3)
		for _, file := range directory.Files {
			assert.False(t, file.Ours, "frame %d", file.Index)
		}
		assert.True(t, directory.Files[2].Whole, "a whole PNG of another tool")
	})

	t.Run("should find nothing in a directory that has no frames, without failing", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		directory, err := pngfile.NewFrameRepository().List(dir)

		// then
		require.NoError(t, err)
		assert.Empty(t, directory.Files)
	})

	t.Run("should refuse a directory that does not exist, saying so", func(t *testing.T) {
		// given
		dir := filepath.Join(t.TempDir(), "missing")

		// when
		_, err := pngfile.NewFrameRepository().List(dir)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.ErrorContains(t, err, dir+" does not exist")
		assert.NoDirExists(t, dir, "listing creates nothing")
	})

	t.Run("should refuse a path that is a file, saying it is not a directory", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))

		// when
		_, err := pngfile.NewFrameRepository().List(path)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.ErrorContains(t, err, "is not a directory")
	})

	t.Run("should refuse a directory it cannot read", func(t *testing.T) {
		// given
		skipWithoutPermissions(t)
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		// when
		_, err := pngfile.NewFrameRepository().List(dir)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
	})
}
