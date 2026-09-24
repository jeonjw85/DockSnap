package copy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jjw/docksnap/internal/engine"
)

const (
	alpineImage = "docker.io/library/alpine:3.21"
	volumeMount = "/dosnap-vol"
)

var wipeCmd = []string{"sh", "-c", "find /dosnap-vol -mindepth 1 -maxdepth 1 -exec rm -rf {} +"}

func SaveNamed(ctx context.Context, eng engine.Engine, volumeName, destFile string) error {
	return withHelper(ctx, eng, volumeName, func(id string) error {
		rc, err := eng.CopyFrom(ctx, id, volumeMount)
		if err != nil {
			return fmt.Errorf("copy from %s: %w", volumeName, err)
		}
		defer rc.Close()
		if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", volumeName, err)
		}
		f, err := os.Create(destFile)
		if err != nil {
			return fmt.Errorf("create %s: %w", destFile, err)
		}
		defer f.Close()
		if _, err := io.Copy(f, rc); err != nil {
			return fmt.Errorf("write %s: %w", volumeName, err)
		}
		return nil
	})
}

func RestoreNamed(ctx context.Context, eng engine.Engine, volumeName, destFile string) error {
	f, err := os.Open(destFile)
	if err != nil {
		return fmt.Errorf("open %s: %w", destFile, err)
	}
	defer f.Close()
	return withHelper(ctx, eng, volumeName, func(id string) error {
		if err := eng.Exec(ctx, id, wipeCmd); err != nil {
			return fmt.Errorf("wipe %s: %w", volumeName, err)
		}
		if err := eng.CopyTo(ctx, id, volumeMount, f); err != nil {
			return fmt.Errorf("copy to %s: %w", volumeName, err)
		}
		return nil
	})
}

func withHelper(ctx context.Context, eng engine.Engine, volumeName string, fn func(id string) error) (err error) {
	if err := ensureImage(ctx, eng); err != nil {
		return fmt.Errorf("%s: %w", volumeName, err)
	}
	id, err := eng.Create(ctx, engine.CreateSpec{
		Name:  "dosnap-" + volumeName,
		Image: alpineImage,
		Cmd:   []string{"sleep", "infinity"},
		Labels: map[string]string{
			"dosnap.helper": "1",
		},
		Mounts: []engine.Mount{{
			Type:   "volume",
			Name:   volumeName,
			Source: volumeName,
			Dest:   volumeMount,
		}},
		AutoRemove: true,
	})
	if err != nil {
		return fmt.Errorf("create helper %s: %w", volumeName, err)
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		rerr := eng.Remove(cleanupCtx, id)
		if rerr != nil {
			err = errors.Join(err, fmt.Errorf("remove helper %s: %w", volumeName, rerr))
		}
	}()
	if err := eng.Start(ctx, id); err != nil {
		return fmt.Errorf("start helper %s: %w", volumeName, err)
	}
	return fn(id)
}

func ensureImage(ctx context.Context, eng engine.Engine) error {
	if err := eng.ImageInspect(ctx, alpineImage); err == nil {
		return nil
	}
	if err := eng.ImagePull(ctx, alpineImage); err != nil {
		return fmt.Errorf("pull %s: %w", alpineImage, err)
	}
	return nil
}
