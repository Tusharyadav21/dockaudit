package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"dockaudit/internal/trends"
)

var trendsCmd = &cobra.Command{
	Use:   "trends",
	Short: "Display historical growth trends of Docker resources",
	Run: func(cmd *cobra.Command, args []string) {
		snapshots, err := trends.LoadSnapshots()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading snapshots: %v\n", err)
			os.Exit(1)
		}

		if len(snapshots) < 2 {
			fmt.Println("Not enough data to show trends. Run 'dockaudit doctor' a few times over a few days to build history.")
			return
		}

		first := snapshots[0]
		last := snapshots[len(snapshots)-1]

		duration := last.Timestamp.Sub(first.Timestamp)
		days := int(duration.Hours() / 24)
		if days == 0 {
			days = 1 // Prevent saying "Past 0 Days"
		}

		imgDiff := len(last.Images) - len(first.Images)
		volDiff := len(last.Volumes) - len(first.Volumes)
		netDiff := len(last.Networks) - len(first.Networks)
		diskDiff := last.DiskUsage - first.DiskUsage

		fmt.Printf("Historical Trends (Past %d Days)\n", days)
		fmt.Println("=================================")

		fmt.Printf("Images:   %+d (Currently: %d)\n", imgDiff, len(last.Images))
		fmt.Printf("Volumes:  %+d (Currently: %d)\n", volDiff, len(last.Volumes))
		fmt.Printf("Networks: %+d (Currently: %d)\n", netDiff, len(last.Networks))

		diskDiffGB := float64(diskDiff) / (1024 * 1024 * 1024)
		fmt.Printf("Disk:     %+.2f GB\n", diskDiffGB)
	},
}

func init() {
	rootCmd.AddCommand(trendsCmd)
}
