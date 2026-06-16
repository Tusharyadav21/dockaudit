package cleanup

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"dockaudit/internal/graph/model"
)

type Cleaner struct {
	client *client.Client
}

func NewCleaner() (*Cleaner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Cleaner{client: cli}, nil
}

func (c *Cleaner) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

func (c *Cleaner) RemoveContainer(ctx context.Context, id string) error {
	return c.client.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
}

func (c *Cleaner) RemoveImage(ctx context.Context, id string) error {
	_, err := c.client.ImageRemove(ctx, id, image.RemoveOptions{Force: true})
	return err
}

func (c *Cleaner) RemoveVolume(ctx context.Context, id string) error {
	return c.client.VolumeRemove(ctx, id, true)
}

func (c *Cleaner) RemoveNetwork(ctx context.Context, id string) error {
	return c.client.NetworkRemove(ctx, id)
}

func (c *Cleaner) RemoveResource(ctx context.Context, id string, resType model.ResourceType) error {
	switch resType {
	case model.TypeContainer:
		return c.RemoveContainer(ctx, id)
	case model.TypeImage:
		return c.RemoveImage(ctx, id)
	case model.TypeVolume:
		return c.RemoveVolume(ctx, id)
	case model.TypeNetwork:
		return c.RemoveNetwork(ctx, id)
	default:
		return fmt.Errorf("unsupported resource type for removal: %s", resType)
	}
}
