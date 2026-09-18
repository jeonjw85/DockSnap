package copy_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jjw/docksnap/internal/copy"
	"github.com/jjw/docksnap/internal/engine"
	"github.com/stretchr/testify/require"
)

func Test_SaveNamed_RestoreNamed_roundtrips_hello(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddVolume("proj_pgdata", []byte("hello"))
	dest := filepath.Join(t.TempDir(), "data.tar")

	require.NoError(t, copy.SaveNamed(ctx, f, "proj_pgdata", dest))
	f.AddVolume("proj_pgdata", []byte("dirty"))
	require.NoError(t, copy.RestoreNamed(ctx, f, "proj_pgdata", dest))

	require.Equal(t, []byte("hello"), f.Blob("proj_pgdata"))
}

func Test_SaveNamed_CopyFrom_fail_still_Remove(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	dest := filepath.Join(t.TempDir(), "data.tar")

	err := copy.SaveNamed(ctx, f, "proj_pgdata", dest)

	require.Error(t, err)
	require.Contains(t, err.Error(), "proj_pgdata")
	list, lerr := f.ContainerList(ctx, engine.ListOptions{All: true})
	require.NoError(t, lerr)
	require.Empty(t, list)
	removed := false
	for _, c := range f.Calls() {
		if strings.HasPrefix(c, "Remove:") {
			removed = true
		}
	}
	require.True(t, removed)
}

func Test_SaveNamed_helper_gone_after_roundtrip(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()
	f.AddVolume("proj_pgdata", []byte("hello"))
	dest := filepath.Join(t.TempDir(), "data.tar")

	require.NoError(t, copy.SaveNamed(ctx, f, "proj_pgdata", dest))

	list, err := f.ContainerList(ctx, engine.ListOptions{All: true})
	require.NoError(t, err)
	require.Empty(t, list)
	_, err = os.Stat(dest)
	require.NoError(t, err)
}
