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
