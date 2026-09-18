package project_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/jjw/docksnap/internal/project"
	"github.com/stretchr/testify/require"
)

const (
	labelProject    = "com.docker.compose.project"
	labelWorkingDir = "com.docker.compose.project.working_dir"
	labelService    = "com.docker.compose.service"
	labelHelper     = "dosnap.helper"
)

func envProj(key string) string {
	if key == "COMPOSE_PROJECT_NAME" {
		return "proj"
	}
	return ""
}

func envNone(string) string { return "" }

func cleanPath(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(resolved)
}

func volumeByType(vols []project.Volume) map[project.VolumeType][]project.Volume {
	out := make(map[project.VolumeType][]project.Volume)
	for _, v := range vols {
		out[v.Type] = append(out[v.Type], v)
	}
	return out
}

func Test_Resolve_returns_named_and_bind_from_engine_labels(t *testing.T) {
	ctx := context.Background()
	cwd := t.TempDir()
	bindSrc := filepath.Join(cwd, "data")
	fake := engine.NewFake()
	fake.AddContainer(engine.Container{
		ID:      "db1",
		Running: true,
		Labels: map[string]string{
			labelProject:    "proj",
			labelWorkingDir: cwd,
			labelService:    "db",
		},
		Mounts: []engine.Mount{
			{Type: "volume", Name: "proj_pgdata", Source: "proj_pgdata", Dest: "/var/lib/postgresql/data"},
			{Type: "bind", Source: bindSrc, Dest: "/data"},
			{Type: "tmpfs", Dest: "/tmp"},
		},
	})

	got, err := project.Resolve(ctx, fake, cwd, envProj)

	require.NoError(t, err)
	require.Equal(t, "proj", got.Name)
	require.Equal(t, cleanPath(t, cwd), got.WorkingDir)
	require.Equal(t, []string{"db1"}, got.Containers)
	byType := volumeByType(got.Volumes)
	require.Len(t, byType[project.VolumeNamed], 1)
	require.Equal(t, "proj_pgdata", byType[project.VolumeNamed][0].Name)
	require.Len(t, byType[project.VolumeBind], 1)
	require.Equal(t, bindSrc, byType[project.VolumeBind][0].Source)
	require.Empty(t, byType[project.VolumeAnonymous])
	for _, v := range got.Volumes {
		require.NotEqual(t, "tmpfs", string(v.Type))
	}
}

func Test_Resolve_errors_when_working_dir_mismatches(t *testing.T) {
	ctx := context.Background()
	cwd := t.TempDir()
	fake := engine.NewFake()
	fake.AddContainer(engine.Container{
		ID:      "db1",
		Running: true,
		Labels: map[string]string{
			labelProject:    "proj",
			labelWorkingDir: "/other",
			labelService:    "db",
		},
		Mounts: []engine.Mount{
			{Type: "volume", Name: "proj_pgdata", Source: "proj_pgdata", Dest: "/var/lib/postgresql/data"},
		},
	})

	_, err := project.Resolve(ctx, fake, cwd, envProj)

	require.Error(t, err)
}

func Test_Resolve_ignores_helper_container(t *testing.T) {
	ctx := context.Background()
	cwd := t.TempDir()
	fake := engine.NewFake()
	fake.AddContainer(engine.Container{
		ID:      "db1",
		Running: true,
		Labels: map[string]string{
			labelProject:    "proj",
			labelWorkingDir: cwd,
			labelService:    "db",
		},
		Mounts: []engine.Mount{
			{Type: "volume", Name: "proj_pgdata", Source: "proj_pgdata", Dest: "/var/lib/postgresql/data"},
		},
	})
	fake.AddContainer(engine.Container{
		ID:      "helper1",
		Running: true,
		Labels: map[string]string{
			labelProject:    "proj",
			labelWorkingDir: cwd,
			labelHelper:     "1",
		},
		Mounts: []engine.Mount{
			{Type: "volume", Name: "helper_vol", Source: "helper_vol", Dest: "/dosnap-vol"},
		},
	})

	got, err := project.Resolve(ctx, fake, cwd, envProj)

	require.NoError(t, err)
	require.Equal(t, []string{"db1"}, got.Containers)
	byType := volumeByType(got.Volumes)
	require.Len(t, byType[project.VolumeNamed], 1)
	require.Equal(t, "proj_pgdata", byType[project.VolumeNamed][0].Name)
}

func Test_Resolve_compose_fallback_named_and_bind_skips_tmpfs(t *testing.T) {
	ctx := context.Background()
	cwd := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(cwd, "data"), 0o755))
	yaml := []byte(`name: proj
services:
  postgres:
    image: postgres
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./data:/data
      - type: tmpfs
        target: /tmp
volumes:
  pgdata:
`)
	require.NoError(t, os.WriteFile(filepath.Join(cwd, "compose.yaml"), yaml, 0o644))
	fake := engine.NewFake()

	got, err := project.Resolve(ctx, fake, cwd, envNone)

	require.NoError(t, err)
	require.Equal(t, "proj", got.Name)
	require.Empty(t, got.Containers)
	byType := volumeByType(got.Volumes)
	require.NotEmpty(t, byType[project.VolumeNamed])
	require.NotEmpty(t, byType[project.VolumeBind])
	require.Empty(t, byType[project.VolumeAnonymous])
	namedNames := make([]string, 0, len(byType[project.VolumeNamed]))
	for _, v := range byType[project.VolumeNamed] {
		namedNames = append(namedNames, v.Name)
	}
	require.Contains(t, namedNames, "proj_pgdata")
	bindSrc := filepath.Join(cleanPath(t, cwd), "data")
	require.Equal(t, bindSrc, byType[project.VolumeBind][0].Source)
	for _, v := range got.Volumes {
		require.NotEqual(t, "tmpfs", string(v.Type))
	}
}

func Test_Resolve_errors_when_no_compose_project(t *testing.T) {
	ctx := context.Background()
	cwd := t.TempDir()
	fake := engine.NewFake()

	_, err := project.Resolve(ctx, fake, cwd, envNone)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no compose project")
}
