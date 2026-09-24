package main

import (
	"context"
	"io"
	"testing"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/stretchr/testify/require"
)

type cancelOnCopyFrom struct {
	engine.Engine
	cancel context.CancelFunc
}

func (e cancelOnCopyFrom) CopyFrom(ctx context.Context, id, path string) (io.ReadCloser, error) {
	e.cancel()
	return e.Engine.CopyFrom(ctx, id, path)
}

type cancelOnCopyTo struct {
	engine.Engine
	cancel context.CancelFunc
}

func (e cancelOnCopyTo) CopyTo(ctx context.Context, id, path string, r io.Reader) error {
	e.cancel()
	return e.Engine.CopyTo(ctx, id, path, r)
}

func Test_runSave_unpauses_and_removes_helper_after_context_cancellation(t *testing.T) {
	f, _ := setupProj(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	orig := newEngine
	newEngine = func() (engine.Engine, error) {
		return cancelOnCopyFrom{Engine: f, cancel: cancel}, nil
	}
	t.Cleanup(func() { newEngine = orig })

	err := runSave(ctx, "seed1", io.Discard)

	require.Error(t, err)
	containers, listErr := f.ContainerList(context.Background(), engine.ListOptions{All: true})
	require.NoError(t, listErr)
	for _, c := range containers {
		require.False(t, c.Paused, c.ID)
	}
	require.Contains(t, f.Calls(), "Remove:c1")
}

func Test_runRestore_restarts_containers_after_context_cancellation(t *testing.T) {
	f, _ := setupProj(t, true)
	require.NoError(t, runSave(context.Background(), "seed1", io.Discard))
	f.AddVolume("proj_pgdata", []byte("dirty"))
	ctx, cancel := context.WithCancel(context.Background())
	orig := newEngine
	newEngine = func() (engine.Engine, error) {
		return cancelOnCopyTo{Engine: f, cancel: cancel}, nil
	}
	t.Cleanup(func() { newEngine = orig })

	err := runRestore(ctx, "seed1", io.Discard)

	require.Error(t, err)
	stop := callIndex(f.Calls(), "Stop:db1")
	start := callIndex(f.Calls(), "Start:db1")
	require.GreaterOrEqual(t, stop, 0)
	require.Greater(t, start, stop)
}
