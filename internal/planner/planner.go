package planner

import (
	"dockaudit/internal/graph/model"
)

type RiskCategory string

const (
	RiskSafe       RiskCategory = "SAFE"
	RiskLikelySafe RiskCategory = "LIKELY SAFE"
	RiskReview     RiskCategory = "REVIEW"
	RiskRisky      RiskCategory = "RISKY"
	RiskDangerous  RiskCategory = "DANGEROUS"
)

type Action struct {
	NodeID       string
	ResourceType model.ResourceType
	Name         string
	SizeBytes    int64
	Risk         RiskCategory
	Reason       string
}

type Plan struct {
	Actions           []Action
	TotalRecoveryBytes int64
}

type Planner struct {
	graph *model.Graph
}

func NewPlanner(g *model.Graph) *Planner {
	return &Planner{graph: g}
}

func (p *Planner) GeneratePlan() *Plan {
	plan := &Plan{
		Actions: make([]Action, 0),
	}

	for _, node := range p.graph.Nodes {
		var base model.BaseResource
		
		switch v := node.Resource.(type) {
		case *model.ContainerNode:
			base = v.BaseResource
		case *model.ImageNode:
			base = v.BaseResource
		case *model.VolumeNode:
			base = v.BaseResource
		case *model.NetworkNode:
			base = v.BaseResource
		case *model.ComposeProjectNode:
			continue // We don't "delete" compose projects directly via planner yet
		default:
			continue
		}

		risk, reason := p.evaluateRisk(node)
		
		plan.Actions = append(plan.Actions, Action{
			NodeID:       node.ID,
			ResourceType: base.Type,
			Name:         base.Name,
			SizeBytes:    base.SizeBytes,
			Risk:         risk,
			Reason:       reason,
		})
		if risk == RiskSafe || risk == RiskLikelySafe {
			plan.TotalRecoveryBytes += base.SizeBytes
		}
	}

	return plan
}

func (p *Planner) evaluateRisk(node *model.Node) (RiskCategory, string) {
	incoming := p.graph.FindIncomingEdges(node.ID)

	switch v := node.Resource.(type) {
	case *model.ImageNode:
		if v.Dangling {
			return RiskSafe, "Dangling image with no references"
		}
		if len(incoming) > 0 {
			return RiskDangerous, "Image is used by active containers"
		}
		return RiskLikelySafe, "Unused image"

	case *model.VolumeNode:
		if len(incoming) > 0 {
			for _, e := range incoming {
				if e.Relationship == "Mounts" {
					return RiskDangerous, "Volume is attached to a container"
				}
			}
			return RiskReview, "Volume belongs to a project but is not currently mounted"
		}
		return RiskSafe, "Orphaned volume with no attachments or project links"

	case *model.NetworkNode:
		if len(incoming) > 0 {
			for _, e := range incoming {
				if e.Relationship == "ConnectedTo" {
					return RiskDangerous, "Network has connected containers"
				}
			}
		}
		if len(incoming) == 0 {
			return RiskLikelySafe, "Unused network"
		}
		return RiskReview, "Network has incoming edges of unknown type"

	case *model.ContainerNode:
		if v.State == "running" || v.State == "restarting" {
			return RiskDangerous, "Container is currently running"
		}
		return RiskReview, "Stopped container"

	default:
		return RiskReview, "Unknown resource type"
	}
}
