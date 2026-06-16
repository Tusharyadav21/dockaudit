package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"dockaudit/internal/analyzer/relationships"
	"dockaudit/internal/docker/discovery"
	"dockaudit/internal/planner"
	"dockaudit/internal/report"
)

var reportFormat string
var reportOutput string

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate an audit report",
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

		switch reportFormat {
		case "json":
			if reportOutput == "" {
				reportOutput = "dockaudit-report.json"
			}
			err = report.GenerateJSON(plan, reportOutput)
		case "html":
			if reportOutput == "" {
				reportOutput = "dockaudit-report.html"
			}
			err = report.GenerateHTML(plan, reportOutput)
		case "md", "markdown":
			if reportOutput == "" {
				reportOutput = "dockaudit-report.md"
			}
			err = report.GenerateMarkdown(plan, reportOutput)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported format: %s. Use json, html, or md.\n", reportFormat)
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate report: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Report successfully generated: %s\n", reportOutput)
	},
}

func init() {
	reportCmd.Flags().StringVarP(&reportFormat, "format", "f", "html", "Report format (json, md, html)")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Output file name")
	rootCmd.AddCommand(reportCmd)
}
