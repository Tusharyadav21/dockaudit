package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"dockaudit/internal/analyzer/relationships"
	"dockaudit/internal/cleanup"
	"dockaudit/internal/docker/discovery"
	"dockaudit/internal/planner"
)

var autoConfirm bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Interactively execute the safe cleanup plan",
	Run: func(cmd *cobra.Command, args []string) {
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

		p := planner.NewPlanner(graph)
		plan := p.GeneratePlan()

		cleaner, err := cleanup.NewCleaner()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating cleaner: %v\n", err)
			os.Exit(1)
		}
		defer cleaner.Close()

		fmt.Println("Interactive Cleanup")
		fmt.Println("===================")

		reader := bufio.NewReader(os.Stdin)
		var recovered int64

		for _, action := range plan.Actions {
			if action.Risk == planner.RiskSafe || action.Risk == planner.RiskLikelySafe {
				fmt.Printf("\n[DELETE] %s: %s\n", action.ResourceType, action.Name)
				fmt.Printf("Reason: %s\n", action.Reason)

				confirm := true
				if !autoConfirm {
					fmt.Print("Delete this resource? [y/N]: ")
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(strings.ToLower(input))
					if input != "y" && input != "yes" {
						confirm = false
					}
				}

				if confirm {
					err := cleaner.RemoveResource(ctx, action.NodeID, action.ResourceType)
					if err != nil {
						fmt.Printf("❌ Failed to delete %s: %v\n", action.Name, err)
					} else {
						fmt.Printf("✅ Deleted %s\n", action.Name)
						recovered += action.SizeBytes
					}
				} else {
					fmt.Println("⏭️  Skipped")
				}
			}
		}

		fmt.Printf("\nCleanup complete. Recovered %.2f GB\n", float64(recovered)/(1024*1024*1024))
	},
}

func init() {
	cleanCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm all deletions")
	rootCmd.AddCommand(cleanCmd)
}
