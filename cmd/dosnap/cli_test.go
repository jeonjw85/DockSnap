package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/jjw/docksnap/internal/store"
	"github.com/stretchr/testify/require"
)

const (
	labelProject    = "com.docker.compose.project"
	labelWorkingDir = "com.docker.compose.project.working_dir"
	labelService    = "com.docker.compose.service"
)

func setupProj(t *testing.T, running bool) (*engine.Fake, string) {
	t.Helper()
	return setupProjBlob(t, running, true)
}

func setupProjBlob(t *testing.T, running, withBlob bool) (*engine.Fake, string) {
	t.Helper()
	cwd := t.TempDir()
	chdir(t, cwd)
	wd, err := os.Getwd()
	require.NoError(t, err)
	f := engine.NewFake()
	require.NoError(t, f.ImagePull(context.Background(), engine.HelperImage))
	if withBlob {
		f.AddVolume("proj_pgdata", []byte("hello"))
	}
	f.AddContainer(engine.Container{
		ID:      "db1",
		Running: running,
		Labels: map[string]string{
			labelProject:    "proj",
			labelWorkingDir: wd,
			labelService:    "db",
		},
		Mounts: []engine.Mount{
			{Type: "volume", Name: "proj_pgdata", Source: "proj_pgdata", Dest: "/var/lib/postgresql/data"},
		},
	})
	orig := newEngine
	newEngine = func() (engine.Engine, error) { return f, nil }
	t.Cleanup(func() { newEngine = orig })
	t.Setenv("COMPOSE_PROJECT_NAME", "proj")
	return f, wd
}

func callIndex(calls []string, prefix string) int {
	for i, c := range calls {
		if strings.HasPrefix(c, prefix) {
			return i
		}
	}
	return -1
}

func Test_runSave_seed1_then_List(t *testing.T) {
	ctx := context.Background()
	f, cwd := setupProj(t, true)

	err := runSave(ctx, "seed1", io.Discard)

	require.NoError(t, err)
	got, err := runList()
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "seed1", got[0].Tag.String())
	_, err = os.Stat(filepath.Join(cwd, ".dosnap", "snapshots", "seed1", "volumes", "proj_pgdata", "data.tar"))
	require.NoError(t, err)
	calls := f.Calls()
	pi := callIndex(calls, "Pause:")
	ci := callIndex(calls, "CopyFrom:")
	ui := callIndex(calls, "Unpause:")
	require.GreaterOrEqual(t, pi, 0)
	require.Greater(t, ci, pi)
	require.Greater(t, ui, ci)
}

func Test_runSave_copy_fail_still_Unpause_no_live_tag(t *testing.T) {
	ctx := context.Background()
	f, cwd := setupProjBlob(t, true, false)

	err := runSave(ctx, "seed1", io.Discard)

	require.Error(t, err)
	require.Greater(t, callIndex(f.Calls(), "Unpause:"), callIndex(f.Calls(), "Pause:"))
	_, statErr := os.Stat(filepath.Join(cwd, ".dosnap", "snapshots", "seed1"))
	require.True(t, os.IsNotExist(statErr))
	_, tmpErr := os.Stat(filepath.Join(cwd, ".dosnap", "snapshots", "seed1.tmp"))
	require.True(t, os.IsNotExist(tmpErr))
}

func Test_runSave_invalid_tag_does_not_Pause(t *testing.T) {
	ctx := context.Background()
	f, _ := setupProj(t, true)

	err := runSave(ctx, "bad tag", io.Discard)

	require.Error(t, err)
	require.Contains(t, err.Error(), "tag")
	require.Equal(t, 2, exitStatus(err))
	require.Equal(t, -1, callIndex(f.Calls(), "Pause:"))
}

func Test_runRestore_roundtrip_blob(t *testing.T) {
	ctx := context.Background()
	f, _ := setupProj(t, true)
	require.NoError(t, runSave(ctx, "seed1", io.Discard))
	f.AddVolume("proj_pgdata", []byte("dirty"))

	err := runRestore(ctx, "seed1", io.Discard)

	require.NoError(t, err)
	require.Equal(t, []byte("hello"), f.Blob("proj_pgdata"))
}

func Test_runRestore_workingDir_mismatch_no_Stop(t *testing.T) {
	ctx := context.Background()
	f, cwd := setupProj(t, true)
	tag, err := store.NewTag("seed1")
	require.NoError(t, err)
	st := store.New(cwd)
	require.NoError(t, st.Save(store.Meta{
		Tag:       tag,
		CreatedAt: time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
		Project:   store.ProjectMeta{Name: "proj", WorkingDir: "/foreign"},
		Volumes: map[string]store.VolumeMeta{
			"proj_pgdata": {
				Name:   "proj_pgdata",
				Type:   store.VolumeNamed,
				Source: "proj_pgdata",
				Format: store.FormatTar,
			},
		},
	}))

	err = runRestore(ctx, "seed1", io.Discard)

	require.Error(t, err)
	require.Equal(t, -1, callIndex(f.Calls(), "Stop:"))
}

func Test_runRestore_already_stopped_not_Start(t *testing.T) {
	ctx := context.Background()
	f, _ := setupProj(t, false)
	require.NoError(t, runSave(ctx, "seed1", io.Discard))

	err := runRestore(ctx, "seed1", io.Discard)

	require.NoError(t, err)
	for _, c := range f.Calls() {
		require.NotEqual(t, "Start:db1", c)
	}
}

func Test_runList_sorts_createdAt_desc(t *testing.T) {
	cwd := t.TempDir()
	chdir(t, cwd)
	st := store.New(cwd)
	require.NoError(t, st.Save(listMeta(t, "old", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))))
	require.NoError(t, st.Save(listMeta(t, "new", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))))

	got, err := runList()

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "new", got[0].Tag.String())
	require.Equal(t, "old", got[1].Tag.String())
}

func Test_runList_empty_dir_empty_slice(t *testing.T) {
	chdir(t, t.TempDir())

	got, err := runList()

	require.NoError(t, err)
	require.Empty(t, got)
}

func Test_cobra_save_restore_ls(t *testing.T) {
	ctx := context.Background()
	f, _ := setupProj(t, true)

	_, _, err := execRoot(ctx, "save", "seed1")
	require.NoError(t, err)
	_, _, err = execRoot(ctx, "ls")
	require.NoError(t, err)
	f.AddVolume("proj_pgdata", []byte("dirty"))
	_, _, err = execRoot(ctx, "restore", "seed1")
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), f.Blob("proj_pgdata"))
}

func Test_cobra_save_bad_tag_stderr_contains_tag(t *testing.T) {
	ctx := context.Background()
	setupProj(t, true)

	_, errOut, err := execRoot(ctx, "save", "bad tag")

	require.Error(t, err)
	require.Contains(t, err.Error(), "tag")
	require.Equal(t, 2, exitStatus(err))
	_ = errOut
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

func execRoot(ctx context.Context, args ...string) (string, string, error) {
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(errb)
	rootCmd.SetArgs(args)
	err := rootCmd.ExecuteContext(ctx)
	return out.String(), errb.String(), err
}

func listMeta(t *testing.T, tag string, at time.Time) store.Meta {
	t.Helper()
	tg, err := store.NewTag(tag)
	require.NoError(t, err)
	return store.Meta{
		Tag:       tg,
		CreatedAt: at,
		Project:   store.ProjectMeta{Name: "proj", WorkingDir: "/abs/path"},
		Volumes: map[string]store.VolumeMeta{
			"proj_pgdata": {
				Name:   "proj_pgdata",
				Type:   store.VolumeNamed,
				Source: "proj_pgdata",
				Bytes:  1,
				Format: store.FormatTar,
			},
		},
	}
}
