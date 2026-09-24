package freeze

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jjw/docksnap/internal/engine"
)

func PauseAll(ctx context.Context, eng engine.Engine, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idx, err := indexContainers(ctx, eng)
	if err != nil {
		return nil, err
	}
	var paused []string
	for _, id := range ids {
		if c, ok := idx[id]; ok && (!c.Running || c.Paused) {
			continue
		}
		if err := eng.Pause(ctx, id); err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			return paused, errors.Join(fmt.Errorf("pause %s: %w", id, err), UnpauseAll(cleanupCtx, eng, paused))
		}
		paused = append(paused, id)
	}
	return paused, nil
}

func UnpauseAll(ctx context.Context, eng engine.Engine, ids []string) error {
	var errs []error
	for _, id := range ids {
		if err := eng.Unpause(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("unpause %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func StopAll(ctx context.Context, eng engine.Engine, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idx, err := indexContainers(ctx, eng)
	if err != nil {
		return nil, err
	}
	var stopped []string
	for _, id := range ids {
		if c, ok := idx[id]; ok && !c.Running {
			continue
		}
		if err := eng.Stop(ctx, id); err != nil {
			var errs []error
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			for _, sid := range stopped {
				if serr := eng.Start(cleanupCtx, sid); serr != nil {
					errs = append(errs, fmt.Errorf("start %s: %w", sid, serr))
				}
			}
			cancel()
			return stopped, errors.Join(fmt.Errorf("stop %s: %w", id, err), errors.Join(errs...))
		}
		stopped = append(stopped, id)
	}
	return stopped, nil
}

func StartAll(ctx context.Context, eng engine.Engine, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idx, err := indexContainers(ctx, eng)
	if err != nil {
		return nil, err
	}
	var started []string
	for _, id := range ids {
		if c, ok := idx[id]; ok && c.Running {
			continue
		}
		if err := eng.Start(ctx, id); err != nil {
			var errs []error
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			for _, sid := range started {
				if serr := eng.Stop(cleanupCtx, sid); serr != nil {
					errs = append(errs, fmt.Errorf("stop %s: %w", sid, serr))
				}
			}
			cancel()
			return started, errors.Join(fmt.Errorf("start %s: %w", id, err), errors.Join(errs...))
		}
		started = append(started, id)
	}
	return started, nil
}

func indexContainers(ctx context.Context, eng engine.Engine) (map[string]engine.Container, error) {
	list, err := eng.ContainerList(ctx, engine.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	idx := make(map[string]engine.Container, len(list)*2)
	for _, c := range list {
		idx[c.ID] = c
		if c.Name != "" {
			idx[c.Name] = c
		}
	}
	return idx, nil
}
