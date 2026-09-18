package engine

import (
	"fmt"

	moby "github.com/moby/moby/client"
)

const (
	HelperImage = "docker.io/library/alpine:3.21"
	HelperLabel = "dosnap.helper"
	HelperMount = "/dosnap-vol"
)

var _ Engine = (*Client)(nil)

type Client struct {
	api *moby.Client
}

func New() (Engine, error) {
	api, err := moby.New(moby.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Client{api: api}, nil
}

func (c *Client) Close() error {
	return c.api.Close()
}
