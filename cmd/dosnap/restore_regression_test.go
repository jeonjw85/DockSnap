package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_runRestore_rejects_corrupt_snapshot_before_stopping_containers(t *testing.T) {
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
	require.Equal(t, -1, callIndex(f.Calls(), "Stop:db1"))
	require.Equal(t, []byte("hello"), f.Blob("proj_pgdata"))
}
