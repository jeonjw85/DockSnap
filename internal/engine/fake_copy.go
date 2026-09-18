package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

func (f *Fake) CopyFrom(ctx context.Context, id, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	name, err := f.mountVolume(id, path)
	if err != nil {
		return nil, err
	}
	blob, ok := f.volumes[name]
	if !ok {
		return nil, fmt.Errorf("copy from %s: volume %s not found", path, name)
	}
	raw, err := encodeTar(blob)
	if err != nil {
		return nil, fmt.Errorf("copy from %s: %w", name, err)
	}
	f.record("CopyFrom", id)
	return io.NopCloser(bytes.NewReader(raw)), nil
}

func (f *Fake) CopyTo(ctx context.Context, id, path string, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	blob, err := decodeTar(r)
	if err != nil {
		return fmt.Errorf("copy to %s: %w", id, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	name, err := f.mountVolume(id, path)
	if err != nil {
		return err
	}
	f.volumes[name] = blob
	f.record("CopyTo", id)
	return nil
}

func (f *Fake) Exec(ctx context.Context, id string, cmd []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.lookup(id)
	if !ok {
		return fmt.Errorf("exec %s: not found", id)
	}
	f.record("Exec", id)
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "find") || !strings.Contains(joined, "rm") {
		return nil
	}
	for _, m := range c.Mounts {
		if m.Name == "" {
			continue
		}
		if _, ok := f.volumes[m.Name]; ok {
			f.volumes[m.Name] = []byte{}
		}
	}
	return nil
}

func encodeTar(blob []byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "data",
		Mode: 0o644,
		Size: int64(len(blob)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(blob); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeTar(r io.Reader) ([]byte, error) {
	tr := tar.NewReader(r)
	var out []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
}
