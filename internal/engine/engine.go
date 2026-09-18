package engine

import (
	"context"
	"io"
)

type Engine interface {
	ContainerList(ctx context.Context, opts ListOptions) ([]Container, error)
	Pause(ctx context.Context, id string) error
	Unpause(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Start(ctx context.Context, id string) error
	Create(ctx context.Context, spec CreateSpec) (string, error)
	Remove(ctx context.Context, id string) error
	Exec(ctx context.Context, id string, cmd []string) error
	CopyFrom(ctx context.Context, id, path string) (io.ReadCloser, error)
	CopyTo(ctx context.Context, id, path string, r io.Reader) error
	ImagePull(ctx context.Context, ref string) error
	ImageInspect(ctx context.Context, ref string) error
	VolumeInspect(ctx context.Context, name string) (Volume, error)
	Close() error
}

type ListOptions struct {
	All    bool
	Labels map[string]string
}

type Container struct {
	ID      string
	Name    string
	Labels  map[string]string
	Running bool
	Paused  bool
	Mounts  []Mount
}

type Mount struct {
	Type   string
	Name   string
	Source string
	Dest   string
}

type CreateSpec struct {
	Name       string
	Image      string
	Cmd        []string
	Labels     map[string]string
	Mounts     []Mount
	AutoRemove bool
}

type Volume struct {
	Name string
}
