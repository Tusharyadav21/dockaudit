package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"dockaudit/internal/graph/model"
)

type Discoverer struct {
	client *client.Client
}

func NewDiscoverer() (*Discoverer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Discoverer{client: cli}, nil
}

func (d *Discoverer) Close() {
	if d.client != nil {
		d.client.Close()
	}
}

func (d *Discoverer) FetchContainers(ctx context.Context) ([]*model.ContainerNode, error) {
	containers, err := d.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var nodes []*model.ContainerNode
	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		nodes = append(nodes, &model.ContainerNode{
			BaseResource: model.BaseResource{
				ID:        c.ID,
				Name:      name,
				Type:      model.TypeContainer,
				CreatedAt: time.Unix(c.Created, 0),
				Labels:    c.Labels,
				Project:   project,
			},
			State:  c.State,
			Status: c.Status,
		})
	}
	return nodes, nil
}

func (d *Discoverer) FetchImages(ctx context.Context) ([]*model.ImageNode, error) {
	images, err := d.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var nodes []*model.ImageNode
	for _, i := range images {
		name := ""
		if len(i.RepoTags) > 0 {
			name = i.RepoTags[0]
		} else if len(i.RepoDigests) > 0 {
			name = i.RepoDigests[0]
		}

		dangling := false
		if val, ok := i.Labels["dangling"]; ok && val == "true" {
			dangling = true
		} else if name == "<none>:<none>" || name == "" {
			dangling = true
		}

		nodes = append(nodes, &model.ImageNode{
			BaseResource: model.BaseResource{
				ID:        i.ID,
				Name:      name,
				Type:      model.TypeImage,
				CreatedAt: time.Unix(i.Created, 0),
				Labels:    i.Labels,
				SizeBytes: i.Size,
			},
			Dangling: dangling,
		})
	}
	return nodes, nil
}

func (d *Discoverer) FetchVolumes(ctx context.Context) ([]*model.VolumeNode, error) {
	volumesList, err := d.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	var nodes []*model.VolumeNode
	for _, v := range volumesList.Volumes {
		project := v.Labels["com.docker.compose.project"]
		
		// To get exact size we'd need to inspect or use usage data, but we'll use UsageData if available
		// Note: Volume sizes might need docker system df API
		
		createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)

		nodes = append(nodes, &model.VolumeNode{
			BaseResource: model.BaseResource{
				ID:        v.Name, // Volumes use Name as primary ID
				Name:      v.Name,
				Type:      model.TypeVolume,
				CreatedAt: createdAt,
				Labels:    v.Labels,
				Project:   project,
			},
			Attached: false, // Will be set by graph traversal
		})
	}
	return nodes, nil
}

func (d *Discoverer) FetchNetworks(ctx context.Context) ([]*model.NetworkNode, error) {
	networks, err := d.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var nodes []*model.NetworkNode
	for _, nw := range networks {
		project := nw.Labels["com.docker.compose.project"]
		createdAt := nw.Created

		nodes = append(nodes, &model.NetworkNode{
			BaseResource: model.BaseResource{
				ID:        nw.ID,
				Name:      nw.Name,
				Type:      model.TypeNetwork,
				CreatedAt: createdAt,
				Labels:    nw.Labels,
				Project:   project,
			},
		})
	}
	return nodes, nil
}

func (d *Discoverer) FetchDiskUsage(ctx context.Context) (int64, int64, error) {
	// Let's use DiskUsage to get total sizes including build cache
	du, err := d.client.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch disk usage: %w", err)
	}

	var buildCacheSize int64
	var totalSize int64

	for _, bc := range du.BuildCache {
		buildCacheSize += bc.Size
		totalSize += bc.Size
	}

	for _, img := range du.Images {
		totalSize += img.Size
	}

	for _, vol := range du.Volumes {
		totalSize += vol.UsageData.Size
	}

	// Containers size is complex, we just sum up SizeRw
	for _, c := range du.Containers {
		totalSize += c.SizeRw
	}

	return buildCacheSize, totalSize, nil
}
