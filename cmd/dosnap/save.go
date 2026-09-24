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

var saveCmd = &cobra.Command{
	Use:           "save <tag>",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSave(cmd.Context(), args[0], cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
}

func runSave(ctx context.Context, rawTag string, out io.Writer) (err error) {
	tag, err := store.NewTag(rawTag)
	if err != nil {
		return &usageError{err: err}
	}
	eng, err := newEngine()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, eng.Close())
	}()
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	proj, err := project.Resolve(ctx, eng, cwd, os.Getenv)
	if err != nil {
		return err
	}
	if len(proj.Volumes) == 0 {
		return fmt.Errorf("no volumes")
	}
	paused, err := freeze.PauseAll(ctx, eng, proj.Containers)
	if err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := cleanupContext(ctx)
		defer cancel()
		err = errors.Join(err, freeze.UnpauseAll(cleanupCtx, eng, paused))
	}()
	start := time.Now()
	st := store.New(cwd)
	tmp, err := st.Prepare(tag)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, os.RemoveAll(tmp))
		}
	}()
	volumes := make(map[string]store.VolumeMeta, len(proj.Volumes))
	var total uint64
	for _, vol := range proj.Volumes {
		vm, err := saveVolume(ctx, eng, tmp, vol)
		if err != nil {
			return err
		}
		id := store.VolumeID(vm.Type, vm.Name, vm.Source)
		volumes[id] = vm
		total += vm.Bytes
	}
	meta := store.Meta{
		Tag:       tag,
		CreatedAt: time.Now().UTC(),
		Project: store.ProjectMeta{
			Name:       proj.Name,
			WorkingDir: proj.WorkingDir,
		},
		Volumes: volumes,
	}
	if err := st.Commit(meta); err != nil {
		return err
	}
	committed = true
	fmt.Fprintf(out, "%s\n", formatLine(tag.String(), len(volumes), total, time.Since(start)))
	return nil
}

func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
}

func saveVolume(ctx context.Context, eng engine.Engine, tmp string, vol project.Volume) (store.VolumeMeta, error) {
	typ := mapVolType(vol.Type)
	id := store.VolumeID(typ, vol.Name, vol.Source)
	dir := filepath.Join(tmp, "volumes", id)
	switch vol.Type {
	case project.VolumeNamed, project.VolumeAnonymous:
		dest := filepath.Join(dir, "data.tar")
		if err := copy.SaveNamed(ctx, eng, vol.Name, dest); err != nil {
			return store.VolumeMeta{}, err
		}
		sum, n, err := checksumFile(dest)
		if err != nil {
			return store.VolumeMeta{}, err
		}
		return store.VolumeMeta{
			Name:     vol.Name,
			Type:     typ,
			Source:   vol.Source,
			Bytes:    n,
			Checksum: sum,
			Format:   store.FormatTar,
		}, nil
	case project.VolumeBind:
		dest := filepath.Join(dir, "tree")
		if err := copy.SaveBind(ctx, vol.Source, dest); err != nil {
			return store.VolumeMeta{}, err
		}
		sum, n, err := checksumTree(dest)
		if err != nil {
			return store.VolumeMeta{}, err
		}
		return store.VolumeMeta{
			Name:     vol.Name,
			Type:     typ,
			Source:   vol.Source,
			Bytes:    n,
			Checksum: sum,
			Format:   store.FormatTree,
		}, nil
	default:
		panic(fmt.Sprintf("unhandled volume type: %q", vol.Type))
	}
}

func mapVolType(t project.VolumeType) store.VolumeType {
	switch t {
	case project.VolumeNamed:
		return store.VolumeNamed
	case project.VolumeAnonymous:
		return store.VolumeAnonymous
	case project.VolumeBind:
		return store.VolumeBind
	default:
		panic(fmt.Sprintf("unhandled volume type: %q", t))
	}
}
