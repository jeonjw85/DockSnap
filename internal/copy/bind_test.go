package copy_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jjw/docksnap/internal/copy"
	"github.com/stretchr/testify/require"
)

func Test_SaveBind_omits_dot_dosnap(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(src, ".dosnap", "snapshots"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, ".dosnap", "snapshots", "x"), []byte("x"), 0o644))

	require.NoError(t, copy.SaveBind(context.Background(), src, dest))

	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), got)
	_, err = os.Stat(filepath.Join(dest, ".dosnap"))
	require.True(t, os.IsNotExist(err))
}

func Test_SaveBind_uses_Go_copy_when_rsync_absent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := t.TempDir()
	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644))

	require.NoError(t, copy.SaveBind(context.Background(), src, dest))

	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), got)
}

func Test_SaveBind_missing_src_error_contains_path(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := filepath.Join(t.TempDir(), "missing")
	err := copy.SaveBind(context.Background(), src, t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), src)
}

func Test_RestoreBind_overwrites_and_deletes_extras_when_rsync_absent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := t.TempDir()
	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "a.txt"), []byte("dirty"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "extra.txt"), []byte("e"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dest, ".dosnap"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, ".dosnap", "keep"), []byte("k"), 0o644))

	require.NoError(t, copy.RestoreBind(context.Background(), src, dest))

	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), got)
	_, err = os.Stat(filepath.Join(dest, "extra.txt"))
	require.True(t, os.IsNotExist(err))
	keep, err := os.ReadFile(filepath.Join(dest, ".dosnap", "keep"))
	require.NoError(t, err)
	require.Equal(t, []byte("k"), keep)
}

func Test_Bind_roundtrips_files_when_rsync_absent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := filepath.Join(t.TempDir(), "config.yml")
	snapshot := filepath.Join(t.TempDir(), "tree")
	dest := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(src, []byte("saved"), 0o640))
	require.NoError(t, copy.SaveBind(context.Background(), src, snapshot))
	require.NoError(t, os.WriteFile(dest, []byte("dirty"), 0o600))

	require.NoError(t, copy.RestoreBind(context.Background(), snapshot, dest))

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, []byte("saved"), got)
	info, err := os.Stat(dest)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}

func Test_Bind_roundtrips_symlinks_when_rsync_absent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := t.TempDir()
	snapshot := t.TempDir()
	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "target-dir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "target-dir", "file"), []byte("saved"), 0o644))
	require.NoError(t, os.Symlink("target-dir", filepath.Join(src, "dir-link")))
	require.NoError(t, os.Symlink("missing-target", filepath.Join(src, "broken-link")))

	require.NoError(t, copy.SaveBind(context.Background(), src, snapshot))
	require.NoError(t, copy.RestoreBind(context.Background(), snapshot, dest))

	for name, target := range map[string]string{"dir-link": "target-dir", "broken-link": "missing-target"} {
		got, err := os.Readlink(filepath.Join(dest, name))
		require.NoError(t, err)
		require.Equal(t, target, got)
	}
	got, err := os.ReadFile(filepath.Join(dest, "dir-link", "file"))
	require.NoError(t, err)
	require.Equal(t, []byte("saved"), got)
}

func Test_SaveBind_copies_read_only_directories_when_rsync_absent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := t.TempDir()
	dest := t.TempDir()
	locked := filepath.Join(src, "locked")
	require.NoError(t, os.Mkdir(locked, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(locked, "data"), []byte("saved"), 0o644))
	require.NoError(t, os.Chmod(locked, 0o555))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	require.NoError(t, copy.SaveBind(context.Background(), src, dest))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dest, "locked"), 0o755) })

	info, err := os.Stat(filepath.Join(dest, "locked"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o555), info.Mode().Perm())
	fileInfo, err := os.Stat(filepath.Join(dest, "locked", "data"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fileInfo.Mode().Perm())
	got, err := os.ReadFile(filepath.Join(dest, "locked", "data"))
	require.NoError(t, err)
	require.Equal(t, []byte("saved"), got)
}
