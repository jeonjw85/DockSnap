package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jjw/docksnap/internal/copy"
	"github.com/jjw/docksnap/internal/engine"
	"github.com/jjw/docksnap/internal/freeze"
	"github.com/jjw/docksnap/internal/project"
	"github.com/jjw/docksnap/internal/store"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:           "restore <tag>",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRestore(cmd.Context(), args[0], cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

func runRestore(ctx context.Context, rawTag string, out io.Writer) (err error) {
	tag, err := store.NewTag(rawTag)
	if err != nil {
		return &usageError{err: err}
	}
	start := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	st := store.New(cwd)
	meta, err := st.Load(tag)
	if err != nil {
		return err
	}
	eng, err := newEngine()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, eng.Close())
	}()
	proj, err := project.Resolve(ctx, eng, cwd, os.Getenv)
	if err != nil {
		return err
	}
	if proj.Name != meta.Project.Name || proj.WorkingDir != meta.Project.WorkingDir {
		return fmt.Errorf("project mismatch: %s %s", meta.Project.Name, meta.Project.WorkingDir)
	}
	snap := filepath.Join(cwd, ".dosnap", "snapshots", tag.String())
	if err := verifySnapshot(snap, meta); err != nil {
		return err
	}
	running, err := runningIDs(ctx, eng, proj.Containers)
	if err != nil {
		return err
	}
	if _, err := freeze.StopAll(ctx, eng, running); err != nil {
		return err
	}
	restoreErr := inspectNamed(ctx, eng, meta)
	if restoreErr == nil {
		restoreErr = restoreVolumes(ctx, eng, snap, meta)
	}
	cleanupCtx, cancel := cleanupContext(ctx)
	_, startErr := freeze.StartAll(cleanupCtx, eng, running)
	cancel()
	if err := errors.Join(restoreErr, startErr); err != nil {
		return err
	}
	var total uint64
	for _, v := range meta.Volumes {
		total += v.Bytes
	}
	fmt.Fprintf(out, "%s\n", formatLine(tag.String(), len(meta.Volumes), total, time.Since(start)))
	return nil
}

func inspectNamed(ctx context.Context, eng engine.Engine, meta store.Meta) error {
	for _, vm := range meta.Volumes {
		switch vm.Type {
		case store.VolumeNamed, store.VolumeAnonymous:
			if _, err := eng.VolumeInspect(ctx, vm.Name); err != nil {
				return fmt.Errorf("%s: %w", vm.Name, err)
			}
		case store.VolumeBind:
		default:
			panic(fmt.Sprintf("unhandled volume type: %q", vm.Type))
		}
	}
	return nil
}

func restoreVolumes(ctx context.Context, eng engine.Engine, snap string, meta store.Meta) error {
	for id, vm := range meta.Volumes {
		switch vm.Type {
		case store.VolumeNamed, store.VolumeAnonymous:
			dest := filepath.Join(snap, "volumes", id, "data.tar")
			if err := copy.RestoreNamed(ctx, eng, vm.Name, dest); err != nil {
				return err
			}
		case store.VolumeBind:
			src := filepath.Join(snap, "volumes", id, "tree")
			if err := copy.RestoreBind(ctx, src, vm.Source); err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("unhandled volume type: %q", vm.Type))
		}
	}
	return nil
}

func runningIDs(ctx context.Context, eng engine.Engine, ids []string) ([]string, error) {
	list, err := eng.ContainerList(ctx, engine.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var running []string
	for _, c := range list {
		if _, ok := want[c.ID]; ok && c.Running {
			running = append(running, c.ID)
		}
	}
	return running, nil
}
