package model

import "time"

type ResourceType string

const (
	TypeContainer      ResourceType = "Container"
	TypeImage          ResourceType = "Image"
	TypeVolume         ResourceType = "Volume"
	TypeNetwork        ResourceType = "Network"
	TypeBuildCache     ResourceType = "BuildCache"
	TypeComposeProject ResourceType = "ComposeProject"
)

type BaseResource struct {
	ID        string
	Name      string
	Type      ResourceType
	CreatedAt time.Time
	LastUsed  *time.Time
	SizeBytes int64
	Labels    map[string]string
	Project   string
}

type ContainerNode struct {
	BaseResource
	State  string
	Status string
}

type ImageNode struct {
	BaseResource
	Dangling bool
}

type VolumeNode struct {
	BaseResource
	Attached bool
}

type NetworkNode struct {
	BaseResource
}

type BuildCacheNode struct {
	BaseResource
	InUse bool
}

type ComposeProjectNode struct {
	BaseResource
	Containers []string // IDs
	Volumes    []string
	Networks   []string
}

type Node struct {
	ID       string
	Resource interface{}
}

type Edge struct {
	FromID       string
	ToID         string
	Relationship string // "Mounts", "CreatedFrom", "ConnectedTo", "BelongsToProject"
}

type Graph struct {
	Nodes map[string]*Node
	Edges []Edge
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]Edge, 0),
	}
}

func (g *Graph) AddNode(id string, resource interface{}) {
	g.Nodes[id] = &Node{
		ID:       id,
		Resource: resource,
	}
}

func (g *Graph) AddEdge(fromID, toID, relationship string) {
	// Simple deduplication could be added here
	g.Edges = append(g.Edges, Edge{
		FromID:       fromID,
		ToID:         toID,
		Relationship: relationship,
	})
}

func (g *Graph) FindIncomingEdges(nodeID string) []Edge {
	var incoming []Edge
	for _, e := range g.Edges {
		if e.ToID == nodeID {
			incoming = append(incoming, e)
		}
	}
	return incoming
}

func (g *Graph) FindOutgoingEdges(nodeID string) []Edge {
	var outgoing []Edge
	for _, e := range g.Edges {
		if e.FromID == nodeID {
			outgoing = append(outgoing, e)
		}
	}
	return outgoing
}
