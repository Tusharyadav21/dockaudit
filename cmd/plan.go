package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"dockaudit/internal/analyzer/relationships"
	"dockaudit/internal/docker/discovery"
	"dockaudit/internal/planner"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate a safe cleanup plan",
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

		fmt.Println("Safe Cleanup Plan")
		fmt.Println("=================")
		fmt.Println()

		var safeCount int
		for _, action := range plan.Actions {
			if action.Risk == planner.RiskSafe || action.Risk == planner.RiskLikelySafe {
				safeCount++
				fmt.Printf("[DELETE] %s %s\n", action.ResourceType, action.Name)
				fmt.Printf("         Risk: %s\n", action.Risk)
				fmt.Printf("         Reason: %s\n\n", action.Reason)
			}
		}

		if safeCount == 0 {
			fmt.Println("No strictly safe resources found to clean up.")
		}

		fmt.Printf("Total Safe Recovery: %.2f GB\n", float64(plan.TotalRecoveryBytes)/(1024*1024*1024))
		fmt.Println("\nTo execute this plan, run: dockaudit clean")
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
