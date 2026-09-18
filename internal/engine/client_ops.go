package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	moby "github.com/moby/moby/client"
)

func (c *Client) ContainerList(ctx context.Context, opts ListOptions) ([]Container, error) {
	listOpts := moby.ContainerListOptions{All: opts.All}
	if len(opts.Labels) > 0 {
		f := make(moby.Filters)
		for k, v := range opts.Labels {
			f.Add("label", k+"="+v)
		}
		listOpts.Filters = f
	}
	res, err := c.api.ContainerList(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("container list: %w", err)
	}
	out := make([]Container, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, fromSummary(item))
	}
	return out, nil
}

func (c *Client) Pause(ctx context.Context, id string) error {
	_, err := c.api.ContainerPause(ctx, id, moby.ContainerPauseOptions{})
	if err != nil {
		return fmt.Errorf("pause %s: %w", id, err)
	}
	return nil
}

func (c *Client) Unpause(ctx context.Context, id string) error {
	_, err := c.api.ContainerUnpause(ctx, id, moby.ContainerUnpauseOptions{})
	if err != nil {
		return fmt.Errorf("unpause %s: %w", id, err)
	}
	return nil
}

func (c *Client) Stop(ctx context.Context, id string) error {
	_, err := c.api.ContainerStop(ctx, id, moby.ContainerStopOptions{})
	if err != nil {
		return fmt.Errorf("stop %s: %w", id, err)
	}
	return nil
}

func (c *Client) Start(ctx context.Context, id string) error {
	_, err := c.api.ContainerStart(ctx, id, moby.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("start %s: %w", id, err)
	}
	return nil
}

func (c *Client) Create(ctx context.Context, spec CreateSpec) (string, error) {
	res, err := c.api.ContainerCreate(ctx, moby.ContainerCreateOptions{
		Name: spec.Name,
		Config: &container.Config{
			Image:  spec.Image,
			Cmd:    spec.Cmd,
			Labels: spec.Labels,
		},
		HostConfig: &container.HostConfig{
			AutoRemove: spec.AutoRemove,
			Mounts:     toAPIMounts(spec.Mounts),
		},
	})
	if err != nil {
		return "", fmt.Errorf("create %s: %w", spec.Name, err)
	}
	return res.ID, nil
}

func (c *Client) Remove(ctx context.Context, id string) error {
	_, err := c.api.ContainerRemove(ctx, id, moby.ContainerRemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("remove %s: %w", id, err)
	}
	return nil
}

func (c *Client) ImagePull(ctx context.Context, ref string) error {
	resp, err := c.api.ImagePull(ctx, ref, moby.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("image pull %s: %w", ref, err)
	}
	defer resp.Close()
	if err := resp.Wait(ctx); err != nil {
		return fmt.Errorf("image pull %s: %w", ref, err)
	}
	return nil
}

func (c *Client) ImageInspect(ctx context.Context, ref string) error {
	_, err := c.api.ImageInspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("image %s: %w", ref, err)
	}
	return nil
}

func (c *Client) VolumeInspect(ctx context.Context, name string) (Volume, error) {
	res, err := c.api.VolumeInspect(ctx, name, moby.VolumeInspectOptions{})
	if err != nil {
		return Volume{}, fmt.Errorf("volume %s: %w", name, err)
	}
	return Volume{Name: res.Volume.Name}, nil
}

func fromSummary(s container.Summary) Container {
	name := ""
	if len(s.Names) > 0 {
		name = strings.TrimPrefix(s.Names[0], "/")
	}
	mounts := make([]Mount, len(s.Mounts))
	for i, m := range s.Mounts {
		mounts[i] = Mount{
			Type:   string(m.Type),
			Name:   m.Name,
			Source: m.Source,
			Dest:   m.Destination,
		}
	}
	return Container{
		ID:      s.ID,
		Name:    name,
		Labels:  s.Labels,
		Running: s.State == container.StateRunning || s.State == container.StatePaused,
		Paused:  s.State == container.StatePaused,
		Mounts:  mounts,
	}
}

func toAPIMounts(in []Mount) []mount.Mount {
	out := make([]mount.Mount, 0, len(in))
	for _, m := range in {
		src := m.Source
		if src == "" {
			src = m.Name
		}
		out = append(out, mount.Mount{
			Type:   mount.Type(m.Type),
			Source: src,
			Target: m.Dest,
		})
	}
	return out
}
