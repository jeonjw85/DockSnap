package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_runRestore_restarts_running_containers_when_volume_copy_fails(t *testing.T) {
	ctx := context.Background()
	f, cwd := setupProj(t, true)
	require.NoError(t, runSave(ctx, "seed1", io.Discard))
	require.NoError(t, os.WriteFile(
		filepath.Join(cwd, ".dosnap", "snapshots", "seed1", "volumes", "proj_pgdata", "data.tar"),
		[]byte("invalid tar"),
		0o644,
	))

	err := runRestore(ctx, "seed1", io.Discard)

	require.Error(t, err)
	stop := callIndex(f.Calls(), "Stop:db1")
	start := callIndex(f.Calls(), "Start:db1")
	require.GreaterOrEqual(t, stop, 0)
	require.Greater(t, start, stop)
}
