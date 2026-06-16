package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"dockaudit/internal/analyzer/relationships"
	"dockaudit/internal/docker/discovery"
	"dockaudit/internal/graph/model"
)

var explainCmd = &cobra.Command{
	Use:   "explain [resource_type] [resource_name_or_id]",
	Short: "Explain why a resource exists and its risk of deletion",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resType := args[0]
		resName := args[1]

		ctx := context.Background()
		disc, err := discovery.NewDiscoverer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
			os.Exit(1)
		}
		defer disc.Close()

		containers, _ := disc.FetchContainers(ctx)
		images, _ := disc.FetchImages(ctx)
		volumes, _ := disc.FetchVolumes(ctx)
		networks, _ := disc.FetchNetworks(ctx)

		builder, err := relationships.NewGraphBuilder()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating graph builder: %v\n", err)
			os.Exit(1)
		}
		defer builder.Close()

		graph, err := builder.Build(ctx, containers, images, volumes, networks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building graph: %v\n", err)
			os.Exit(1)
		}

		// Find the node
		var targetNode *model.Node
		for _, n := range graph.Nodes {
			var base model.BaseResource
			switch v := n.Resource.(type) {
			case *model.ContainerNode:
				base = v.BaseResource
			case *model.ImageNode:
				base = v.BaseResource
			case *model.VolumeNode:
				base = v.BaseResource
			case *model.NetworkNode:
				base = v.BaseResource
			case *model.ComposeProjectNode:
				base = v.BaseResource
			}

			if base.Name == resName || base.ID == resName || (resType == "project" && base.Type == model.TypeComposeProject && base.Name == resName) {
				targetNode = n
				break
			}
		}

		if targetNode == nil {
			fmt.Printf("Resource '%s' of type '%s' not found.\n", resName, resType)
			return
		}

		fmt.Printf("Resource: %s\n", resName)
		
		// Very basic risk assessment logic for M2
		incoming := graph.FindIncomingEdges(targetNode.ID)
		
		isDangerous := false
		var reason string

		if len(incoming) == 0 {
			fmt.Println("Risk: SAFE")
			reason = "Not referenced by any known container or project."
		} else {
			fmt.Println("Risk: DANGEROUS")
			isDangerous = true
			
			for _, e := range incoming {
				sourceNode := graph.Nodes[e.FromID]
				var sourceName string
				switch v := sourceNode.Resource.(type) {
				case *model.ContainerNode:
					sourceName = "container " + v.Name
				case *model.ComposeProjectNode:
					sourceName = "project " + v.Name
				default:
					sourceName = e.FromID
				}
				reason += fmt.Sprintf("Referenced by %s (%s)\n", sourceName, e.Relationship)
			}
		}

		if !isDangerous && targetNode.Resource != nil {
			switch v := targetNode.Resource.(type) {
			case *model.VolumeNode:
				if v.Project != "" {
					reason += fmt.Sprintf("\nCreated by docker-compose project '%s'", v.Project)
				}
			}
		}

		fmt.Printf("Reason:\n%s\n", reason)
	},
}

func init() {
	rootCmd.AddCommand(explainCmd)
}
