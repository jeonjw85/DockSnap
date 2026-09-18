package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jjw/docksnap/internal/engine"
)

const (
	labelProject    = "com.docker.compose.project"
	labelWorkingDir = "com.docker.compose.project.working_dir"
	labelService    = "com.docker.compose.service"
	labelHelper     = "dosnap.helper"
)

func Resolve(ctx context.Context, eng engine.Engine, cwd string, getenv func(string) string) (Project, error) {
	workingDir, err := canonicalize(cwd)
	if err != nil {
		return Project{}, fmt.Errorf("working dir %s: %w", cwd, err)
	}
	name := getenv("COMPOSE_PROJECT_NAME")
	var loaded composeProject
	var loadedOK bool
	if name == "" {
		cp, err := loadCompose(ctx, workingDir, "")
		if err == nil {
			loaded = cp
			loadedOK = true
			name = cp.Name
		}
	}
	var listed []engine.Container
	if name != "" {
		listed, err = eng.ContainerList(ctx, engine.ListOptions{
			All:    true,
			Labels: map[string]string{labelProject: name},
		})
		if err != nil {
			return Project{}, fmt.Errorf("list project %s: %w", name, err)
		}
	}
	var matched []engine.Container
	sameName := 0
	for _, c := range listed {
		if c.Labels[labelHelper] == "1" {
			continue
		}
		sameName++
		wd, err := canonicalize(c.Labels[labelWorkingDir])
		if err != nil {
			continue
		}
		if wd == workingDir {
			matched = append(matched, c)
		}
	}
	if sameName > 0 && len(matched) == 0 {
		return Project{}, fmt.Errorf("working_dir: no container matches %s", workingDir)
	}
	if len(matched) > 0 {
		ids := make([]string, 0, len(matched))
		for _, c := range matched {
			ids = append(ids, c.ID)
		}
		return Project{
			Name:       name,
			WorkingDir: workingDir,
			Containers: ids,
			Volumes:    volumesFromContainers(matched),
		}, nil
	}
	if !loadedOK {
		cp, err := loadCompose(ctx, workingDir, name)
		if err != nil {
			return Project{}, fmt.Errorf("no compose project: %w", err)
		}
		loaded = cp
	}
	return Project{
		Name:       loaded.Name,
		WorkingDir: workingDir,
		Volumes:    volumesFromCompose(loaded),
	}, nil
}

func canonicalize(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return filepath.Clean(abs), nil
	}
	return filepath.Clean(resolved), nil
}
