package engine

import (
	"context"
	"fmt"
)

func (f *Fake) ContainerList(ctx context.Context, opts ListOptions) ([]Container, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Container, 0, len(f.containers))
	for _, c := range f.containers {
		if !opts.All && !c.Running {
			continue
		}
		if !labelsMatch(c.Labels, opts.Labels) {
			continue
		}
		out = append(out, cloneContainer(*c))
	}
	return out, nil
}

func (f *Fake) Pause(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	_, fail := f.failPause[id]
	f.mu.Unlock()
	if fail {
		return fmt.Errorf("pause %s: injected failure", id)
	}
	return f.mutate(ctx, "Pause", id, func(c *Container) { c.Paused = true })
}

func (f *Fake) Unpause(ctx context.Context, id string) error {
	return f.mutate(ctx, "Unpause", id, func(c *Container) { c.Paused = false })
}

func (f *Fake) Stop(ctx context.Context, id string) error {
	return f.mutate(ctx, "Stop", id, func(c *Container) {
		c.Running = false
		c.Paused = false
	})
}

func (f *Fake) Start(ctx context.Context, id string) error {
	return f.mutate(ctx, "Start", id, func(c *Container) { c.Running = true })
}

func (f *Fake) Create(ctx context.Context, spec CreateSpec) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.images[spec.Image]; !ok {
		return "", fmt.Errorf("create %s: image %s not found", spec.Name, spec.Image)
	}
	f.seq++
	id := fmt.Sprintf("c%d", f.seq)
	c := Container{
		ID:     id,
		Name:   spec.Name,
		Labels: cloneMap(spec.Labels),
		Mounts: cloneMounts(spec.Mounts),
	}
	f.containers[id] = &c
	f.record("Create", id)
	return id, nil
}

func (f *Fake) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.lookup(id)
	if !ok {
		return fmt.Errorf("remove %s: not found", id)
	}
	delete(f.containers, c.ID)
	f.record("Remove", id)
	return nil
}

func (f *Fake) ImagePull(ctx context.Context, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.images[ref] = struct{}{}
	f.record("ImagePull", ref)
	return nil
}

func (f *Fake) ImageInspect(ctx context.Context, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.images[ref]; !ok {
		return fmt.Errorf("image %s: not found", ref)
	}
	return nil
}

func (f *Fake) VolumeInspect(ctx context.Context, name string) (Volume, error) {
	if err := ctx.Err(); err != nil {
		return Volume{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.volumes[name]; !ok {
		return Volume{}, fmt.Errorf("volume %s: not found", name)
	}
	return Volume{Name: name}, nil
}

func (f *Fake) Close() error {
	return nil
}

func (f *Fake) mutate(ctx context.Context, op, id string, fn func(*Container)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.lookup(id)
	if !ok {
		return fmt.Errorf("%s %s: not found", op, id)
	}
	fn(c)
	f.record(op, id)
	return nil
}

func labelsMatch(have, want map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}
