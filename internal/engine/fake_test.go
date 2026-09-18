package engine_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/stretchr/testify/require"
)

func Test_Fake_CopyFromTo_roundtrips_blob(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddVolume("proj_pgdata", []byte("hello"))
	f.AddContainer(engine.Container{
		ID:      "helper1",
		Running: true,
		Mounts: []engine.Mount{{
			Type: "volume",
			Name: "proj_pgdata",
			Dest: "/dosnap-vol",
		}},
	})

	rc, err := f.CopyFrom(ctx, "helper1", "/dosnap-vol")
	require.NoError(t, err)
	first, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.NotEmpty(t, first)

	err = f.CopyTo(ctx, "helper1", "/dosnap-vol", bytes.NewReader(first))
	require.NoError(t, err)

	rc, err = f.CopyFrom(ctx, "helper1", "/dosnap-vol")
	require.NoError(t, err)
	second, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, first, second)
}

func Test_Fake_PauseUnpause_records_order(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})

	require.NoError(t, f.Pause(ctx, "c1"))
	require.NoError(t, f.Unpause(ctx, "c1"))
	require.Equal(t, []string{"Pause:c1", "Unpause:c1"}, f.Calls())
}

func Test_Fake_Exec_wipe_clears_blob(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddVolume("proj_pgdata", []byte("hello"))
	f.AddContainer(engine.Container{
		ID:      "helper1",
		Running: true,
		Mounts: []engine.Mount{{
			Type: "volume",
			Name: "proj_pgdata",
			Dest: "/dosnap-vol",
		}},
	})

	err := f.Exec(ctx, "helper1", []string{"sh", "-c", "find /dosnap-vol -mindepth 1 -maxdepth 1 -exec rm -rf {} +"})
	require.NoError(t, err)
	require.Empty(t, f.Blob("proj_pgdata"))
}

func Test_Fake_CopyFrom_missing_volume_error_contains_name(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{
		ID:      "helper1",
		Running: true,
		Mounts: []engine.Mount{{
			Type: "volume",
			Name: "proj_pgdata",
			Dest: "/dosnap-vol",
		}},
	})

	_, err := f.CopyFrom(ctx, "helper1", "/dosnap-vol")
	require.Error(t, err)
	require.Contains(t, err.Error(), "proj_pgdata")
}
