package engine

import (
	"context"
	"fmt"
	"io"

	moby "github.com/moby/moby/client"
)

func (c *Client) CopyFrom(ctx context.Context, id, path string) (io.ReadCloser, error) {
	res, err := c.api.CopyFromContainer(ctx, id, moby.CopyFromContainerOptions{SourcePath: path})
	if err != nil {
		return nil, fmt.Errorf("copy from %s %s: %w", id, path, err)
	}
	return res.Content, nil
}

func (c *Client) CopyTo(ctx context.Context, id, path string, r io.Reader) error {
	_, err := c.api.CopyToContainer(ctx, id, moby.CopyToContainerOptions{
		DestinationPath: path,
		Content:         r,
	})
	if err != nil {
		return fmt.Errorf("copy to %s %s: %w", id, path, err)
	}
	return nil
}

func (c *Client) Exec(ctx context.Context, id string, cmd []string) error {
	created, err := c.api.ExecCreate(ctx, id, moby.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return fmt.Errorf("exec %s: %w", id, err)
	}
	att, err := c.api.ExecAttach(ctx, created.ID, moby.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("exec %s: %w", id, err)
	}
	defer att.Close()
	_, copyErr := io.Copy(io.Discard, att.Reader)
	insp, err := c.api.ExecInspect(ctx, created.ID, moby.ExecInspectOptions{})
	if err != nil {
		return fmt.Errorf("exec %s: %w", id, err)
	}
	if copyErr != nil {
		return fmt.Errorf("exec %s: %w", id, copyErr)
	}
	if insp.ExitCode != 0 {
		return fmt.Errorf("exec %s: exit %d", id, insp.ExitCode)
	}
	return nil
}
