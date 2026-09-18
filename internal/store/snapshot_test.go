package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testMeta(t *testing.T, tag string, bytes uint64) Meta {
	t.Helper()
	tg, err := NewTag(tag)
	require.NoError(t, err)
	return Meta{
		Tag:       tg,
		CreatedAt: time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
		Project: ProjectMeta{
			Name:       "proj",
			WorkingDir: "/abs/path",
		},
		Volumes: map[string]VolumeMeta{
			"proj_pgdata": {
				Name:     "proj_pgdata",
				Type:     VolumeNamed,
				Source:   "proj_pgdata",
				Bytes:    bytes,
				Checksum: "sha256:abcd",
				Format:   FormatTar,
			},
		},
	}
}

func Test_Store_Save_then_List_one_when_seed1(t *testing.T) {
	cwd := t.TempDir()
	s := New(cwd)
	meta := testMeta(t, "seed1", 12345)

	err := s.Save(meta)
	require.NoError(t, err)
	got, err := s.List()

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "seed1", got[0].Tag.String())
	require.Equal(t, uint64(12345), got[0].Volumes["proj_pgdata"].Bytes)
	_, err = os.Stat(filepath.Join(cwd, ".dosnap", "snapshots", "seed1", "meta.json"))
	require.NoError(t, err)
}

func Test_Store_Save_replaces_same_tag(t *testing.T) {
	cwd := t.TempDir()
	s := New(cwd)
	require.NoError(t, s.Save(testMeta(t, "seed1", 1)))

	err := s.Save(testMeta(t, "seed1", 99))

	require.NoError(t, err)
	got, err := s.Load(mustTag(t, "seed1"))
	require.NoError(t, err)
	require.Equal(t, uint64(99), got.Volumes["proj_pgdata"].Bytes)
	listed, err := s.List()
	require.NoError(t, err)
	require.Len(t, listed, 1)
}

func Test_Store_List_skips_leftover_tmp(t *testing.T) {
	cwd := t.TempDir()
	s := New(cwd)
	require.NoError(t, s.Save(testMeta(t, "seed1", 5)))
	tmp := filepath.Join(cwd, ".dosnap", "snapshots", "seed1.tmp")
	require.NoError(t, os.MkdirAll(tmp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "meta.json"), []byte(`{"tag":"seed1"}`), 0o644))

	got, err := s.List()

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "seed1", got[0].Tag.String())
}

func Test_Store_Load_missing_error_contains_missing(t *testing.T) {
	s := New(t.TempDir())
	tag := mustTag(t, "missing")

	_, err := s.Load(tag)

	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")
}

func Test_Store_List_skips_dir_without_meta(t *testing.T) {
	cwd := t.TempDir()
	s := New(cwd)
	require.NoError(t, s.Save(testMeta(t, "seed1", 5)))
	empty := filepath.Join(cwd, ".dosnap", "snapshots", "empty")
	require.NoError(t, os.MkdirAll(empty, 0o755))

	got, err := s.List()

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "seed1", got[0].Tag.String())
}

func Test_Store_List_skips_corrupt_meta(t *testing.T) {
	cwd := t.TempDir()
	s := New(cwd)
	require.NoError(t, s.Save(testMeta(t, "seed1", 5)))
	bad := filepath.Join(cwd, ".dosnap", "snapshots", "bad")
	require.NoError(t, os.MkdirAll(bad, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bad, "meta.json"), []byte(`{not json`), 0o644))

	got, err := s.List()

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "seed1", got[0].Tag.String())
}

func mustTag(t *testing.T, raw string) Tag {
	t.Helper()
	tag, err := NewTag(raw)
	require.NoError(t, err)
	return tag
}
