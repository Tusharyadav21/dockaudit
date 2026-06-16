package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"dockaudit/internal/docker/discovery"
	"dockaudit/internal/trends"
)

var ciMode bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Docker health and identify waste",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		disc, err := discovery.NewDiscoverer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
			os.Exit(1)
		}
		defer disc.Close()

		containers, err := disc.FetchContainers(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching containers: %v\n", err)
			os.Exit(1)
		}

		images, err := disc.FetchImages(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching images: %v\n", err)
			os.Exit(1)
		}

		volumes, err := disc.FetchVolumes(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching volumes: %v\n", err)
			os.Exit(1)
		}

		networks, err := disc.FetchNetworks(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching networks: %v\n", err)
			os.Exit(1)
		}

		buildCacheSize, _, err := disc.FetchDiskUsage(ctx)
		if err != nil {
			// fallback
			buildCacheSize = 0
		}

		// Very basic V1 logic to detect orphans/unused (will be replaced by Graph Traversal in M2)
		// We'll approximate for M1 output
		var orphanedVolumes int
		var unusedImages int
		var potentialRecovery int64

		// Approximate unused images (dangling)
		for _, img := range images {
			if img.Dangling {
				unusedImages++
				potentialRecovery += img.SizeBytes
			}
		}

		// Assume all volumes without active containers are orphaned for the M1 CLI stub
		// (this gets accurate in M2)
		orphanedVolumes = len(volumes) // purely placeholder until Graph is wired

		// Add build cache waste
		potentialRecovery += buildCacheSize
		recoveryGB := float64(potentialRecovery) / (1024 * 1024 * 1024)

		// Save snapshot for trends tracking
		_ = trends.SaveSnapshot(images, volumes, networks, buildCacheSize+potentialRecovery) // Assuming total size roughly

		if ciMode {
			status := "healthy"
			if recoveryGB > 5.0 || orphanedVolumes > 5 || unusedImages > 5 {
				status = "warning"
			}

			out := map[string]interface{}{
				"status":           status,
				"recovery_gb":      fmt.Sprintf("%.2f", recoveryGB),
				"orphaned_volumes": orphanedVolumes,
				"unused_images":    unusedImages,
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
			return
		}

		fmt.Println("Docker Doctor Report")
		fmt.Println()
		fmt.Println("✓ Daemon healthy")

		if orphanedVolumes > 0 {
			fmt.Printf("⚠ %d volumes need review\n", orphanedVolumes)
		}
		if unusedImages > 0 {
			fmt.Printf("⚠ %d unused/dangling images\n", unusedImages)
		}
		if buildCacheSize > 0 {
			fmt.Printf("⚠ Build cache consuming %.2f GB\n", float64(buildCacheSize)/(1024*1024*1024))
		}

		fmt.Println()
		fmt.Printf("Recovery Potential: %.2f GB\n", recoveryGB)
		
		// Just a diagnostic output for M1
		fmt.Printf("Total Containers: %d\n", len(containers))
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&ciMode, "ci", false, "Output in JSON format for CI/CD pipelines")
	rootCmd.AddCommand(doctorCmd)
}
