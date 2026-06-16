package relationships

import (
	"context"
	"github.com/docker/docker/client"
	"dockaudit/internal/graph/model"
)

type GraphBuilder struct {
	client *client.Client
}

func NewGraphBuilder() (*GraphBuilder, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &GraphBuilder{client: cli}, nil
}

func (gb *GraphBuilder) Close() {
	if gb.client != nil {
		gb.client.Close()
	}
}

// Build constructs the full dependency graph from nodes
func (gb *GraphBuilder) Build(
	ctx context.Context,
	containers []*model.ContainerNode,
	images []*model.ImageNode,
	volumes []*model.VolumeNode,
	networks []*model.NetworkNode,
) (*model.Graph, error) {
	
	g := model.NewGraph()

	// Add all nodes
	for _, c := range containers {
		g.AddNode(c.ID, c)
	}
	for _, img := range images {
		g.AddNode(img.ID, img)
	}
	for _, vol := range volumes {
		g.AddNode(vol.ID, vol)
	}
	for _, nw := range networks {
		g.AddNode(nw.ID, nw)
	}

	// For edges, we need to inspect containers to find mounts and networks
	// (Since FetchContainers list doesn't include deep mount/network info)
	for _, c := range containers {
		cJson, err := gb.client.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue // skip if we can't inspect (maybe deleted in between)
		}

		// Edge: Container -> Image
		if cJson.Image != "" {
			g.AddEdge(c.ID, cJson.Image, "CreatedFrom")
		}

		// Edge: Container -> Volume
		for _, m := range cJson.Mounts {
			if m.Type == "volume" {
				g.AddEdge(c.ID, m.Name, "Mounts")
				
				// Mark volume as attached
				if volNode, ok := g.Nodes[m.Name]; ok {
					if v, isVol := volNode.Resource.(*model.VolumeNode); isVol {
						v.Attached = true
					}
				}
			}
		}

		// Edge: Container -> Network
		if cJson.NetworkSettings != nil {
			for netName, netState := range cJson.NetworkSettings.Networks {
				if netState.NetworkID != "" {
					g.AddEdge(c.ID, netState.NetworkID, "ConnectedTo")
					// Using networkName to track might be easier, but ID is more solid.
				} else {
					// try to find network by name
					for _, nw := range networks {
						if nw.Name == netName {
							g.AddEdge(c.ID, nw.ID, "ConnectedTo")
							break
						}
					}
				}
			}
		}

		// Add edge to Compose Project if it belongs to one
		if c.Project != "" {
			projID := "project:" + c.Project
			if _, ok := g.Nodes[projID]; !ok {
				g.AddNode(projID, &model.ComposeProjectNode{
					BaseResource: model.BaseResource{
						ID:   projID,
						Name: c.Project,
						Type: model.TypeComposeProject,
					},
				})
			}
			g.AddEdge(projID, c.ID, "BelongsToProject")
		}
	}

	// Connect volumes and networks to compose projects
	for _, vol := range volumes {
		if vol.Project != "" {
			projID := "project:" + vol.Project
			g.AddEdge(projID, vol.ID, "BelongsToProject")
		}
	}

	for _, nw := range networks {
		if nw.Project != "" {
			projID := "project:" + nw.Project
			g.AddEdge(projID, nw.ID, "BelongsToProject")
		}
	}

	return g, nil
}
