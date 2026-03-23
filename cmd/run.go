package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var runDate string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run sync + summarize (cron target)",
	RunE:  runPipeline,
}

func init() {
	runCmd.Flags().StringVar(&runDate, "date", "", "Date to run (YYYY-MM-DD or 'yesterday', defaults to today)")
	rootCmd.AddCommand(runCmd)
}

func resolveDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Now(), nil
	}
	if dateStr == "yesterday" {
		return time.Now().AddDate(0, 0, -1), nil
	}
	return time.Parse("2006-01-02", dateStr)
}

func runPipeline(cmd *cobra.Command, args []string) error {
	start := time.Now()
	resolved, err := resolveDate(runDate)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	dateStr := resolved.Format("2006-01-02")
	fmt.Printf("Running daily pipeline for %s\n\n", dateStr)

	// Step 1: Sync
	fmt.Println("=== Step 1: Sync ===")
	syncDate = dateStr
	if err := runSync(cmd, args); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}
	fmt.Println()

	// Step 2: Summarize
	fmt.Println("=== Step 2: Summarize ===")
	summarizeDate = dateStr
	if err := runSummarize(cmd, args); err != nil {
		fmt.Printf("Warning: summarize failed: %v\n", err)
	}

	fmt.Printf("\nDaily pipeline completed in %.1fs\n", time.Since(start).Seconds())
	return nil
}
