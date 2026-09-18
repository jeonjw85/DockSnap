package engine

import (
	"bytes"
	"fmt"
	"sync"
)

var _ Engine = (*Fake)(nil)

type Fake struct {
	mu         sync.Mutex
	containers map[string]*Container
	volumes    map[string][]byte
	images     map[string]struct{}
	calls      []string
	seq        int
	failPause  map[string]struct{}
}

func NewFake() *Fake {
	return &Fake{
		containers: make(map[string]*Container),
		volumes:    make(map[string][]byte),
		images:     make(map[string]struct{}),
	}
}

func (f *Fake) AddVolume(name string, blob []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.volumes[name] = bytes.Clone(blob)
}

func (f *Fake) FailPauseOn(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failPause == nil {
		f.failPause = make(map[string]struct{})
	}
	f.failPause[id] = struct{}{}
}

func (f *Fake) AddContainer(c Container) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c.ID == "" {
		f.seq++
		c.ID = fmt.Sprintf("c%d", f.seq)
	}
	stored := cloneContainer(c)
	f.containers[stored.ID] = &stored
}

func (f *Fake) Blob(name string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return bytes.Clone(f.volumes[name])
}

func (f *Fake) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *Fake) lookup(id string) (*Container, bool) {
	if c, ok := f.containers[id]; ok {
		return c, true
	}
	for _, c := range f.containers {
		if c.Name == id {
			return c, true
		}
	}
	return nil, false
}

func (f *Fake) record(op, id string) {
	f.calls = append(f.calls, op+":"+id)
}

func (f *Fake) mountVolume(id, path string) (string, error) {
	c, ok := f.lookup(id)
	if !ok {
		return "", fmt.Errorf("container %s: not found", id)
	}
	for _, m := range c.Mounts {
		if m.Dest != path || m.Name == "" {
			continue
		}
		return m.Name, nil
	}
	return "", fmt.Errorf("container %s: no volume at %s", id, path)
}

func cloneContainer(c Container) Container {
	c.Labels = cloneMap(c.Labels)
	c.Mounts = cloneMounts(c.Mounts)
	return c
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneMounts(in []Mount) []Mount {
	if in == nil {
		return nil
	}
	out := make([]Mount, len(in))
	copy(out, in)
	return out
}
