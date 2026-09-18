package freeze_test

import (
	"context"
	"testing"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/jjw/docksnap/internal/freeze"
	"github.com/stretchr/testify/require"
)

func Test_PauseAll_pauses_three_running_ids(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})
	f.AddContainer(engine.Container{ID: "c2", Running: true})
	f.AddContainer(engine.Container{ID: "c3", Running: true})
	ids := []string{"c1", "c2", "c3"}

	paused, err := freeze.PauseAll(ctx, f, ids)

	require.NoError(t, err)
	require.Equal(t, ids, paused)
	require.Equal(t, []string{"Pause:c1", "Pause:c2", "Pause:c3"}, f.Calls())
}

func Test_PauseAll_unpauses_prior_when_second_fails(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})
	f.AddContainer(engine.Container{ID: "c2", Running: true})
	f.AddContainer(engine.Container{ID: "c3", Running: true})
	f.FailPauseOn("c2")

	_, err := freeze.PauseAll(ctx, f, []string{"c1", "c2", "c3"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "c2")
	require.Equal(t, []string{"Pause:c1", "Unpause:c1"}, f.Calls())
}

func Test_PauseAll_skips_already_paused(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})
	f.AddContainer(engine.Container{ID: "c2", Running: true, Paused: true})
	f.AddContainer(engine.Container{ID: "c3", Running: true})

	paused, err := freeze.PauseAll(ctx, f, []string{"c1", "c2", "c3"})

	require.NoError(t, err)
	require.Equal(t, []string{"c1", "c3"}, paused)
	require.Equal(t, []string{"Pause:c1", "Pause:c3"}, f.Calls())
}

func Test_PauseAll_skips_stopped(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})
	f.AddContainer(engine.Container{ID: "c2", Running: false})
	f.AddContainer(engine.Container{ID: "c3", Running: true})

	paused, err := freeze.PauseAll(ctx, f, []string{"c1", "c2", "c3"})

	require.NoError(t, err)
	require.Equal(t, []string{"c1", "c3"}, paused)
	require.Equal(t, []string{"Pause:c1", "Pause:c3"}, f.Calls())
}

func Test_PauseAll_empty_ids_noop(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})

	paused, err := freeze.PauseAll(ctx, f, nil)

	require.NoError(t, err)
	require.Empty(t, paused)
	require.Empty(t, f.Calls())
}

func Test_UnpauseAll_joins_errors(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()

	err := freeze.UnpauseAll(ctx, f, []string{"gone1", "gone2"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "gone1")
	require.Contains(t, err.Error(), "gone2")
}

func Test_StopAll_stops_running_and_skips_stopped(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})
	f.AddContainer(engine.Container{ID: "c2", Running: false})
	f.AddContainer(engine.Container{ID: "c3", Running: true})

	stopped, err := freeze.StopAll(ctx, f, []string{"c1", "c2", "c3"})

	require.NoError(t, err)
	require.Equal(t, []string{"c1", "c3"}, stopped)
	require.Equal(t, []string{"Stop:c1", "Stop:c3"}, f.Calls())
}

func Test_StopAll_starts_prior_when_second_missing(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: true})

	_, err := freeze.StopAll(ctx, f, []string{"c1", "c2"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "c2")
	require.Equal(t, []string{"Stop:c1", "Start:c1"}, f.Calls())
}

func Test_StartAll_starts_stopped_and_skips_running(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddContainer(engine.Container{ID: "c1", Running: false})
	f.AddContainer(engine.Container{ID: "c2", Running: true})
	f.AddContainer(engine.Container{ID: "c3", Running: false})

	started, err := freeze.StartAll(ctx, f, []string{"c1", "c2", "c3"})

	require.NoError(t, err)
	require.Equal(t, []string{"c1", "c3"}, started)
	require.Equal(t, []string{"Start:c1", "Start:c3"}, f.Calls())
}

func Test_StartAll_empty_ids_noop(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()

	started, err := freeze.StartAll(ctx, f, nil)

	require.NoError(t, err)
	require.Empty(t, started)
	require.Empty(t, f.Calls())
}

func Test_StopAll_empty_ids_noop(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()

	stopped, err := freeze.StopAll(ctx, f, nil)

	require.NoError(t, err)
	require.Empty(t, stopped)
	require.Empty(t, f.Calls())
}
