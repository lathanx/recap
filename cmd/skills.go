package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lathanx/recap/internal/jsonl"
	"github.com/spf13/cobra"
)

var (
	skillsDays  int
	skillsSince string
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Show skill usage frequency across sessions",
	RunE:  runSkills,
}

func init() {
	skillsCmd.Flags().IntVar(&skillsDays, "days", 7, "Number of days to look back")
	skillsCmd.Flags().StringVar(&skillsSince, "since", "", "Start date (YYYY-MM-DD), overrides --days")
	rootCmd.AddCommand(skillsCmd)
}

// skillsDateRange computes the [start, end) date range from flags.
// end is always today (inclusive). start is determined by --since or --days.
func skillsDateRange(now time.Time, days int, since string) (time.Time, time.Time, error) {
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if since != "" {
		start, err := time.ParseInLocation("2006-01-02", since, now.Location())
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --since date: %w", err)
		}
		if start.After(end) {
			return time.Time{}, time.Time{}, fmt.Errorf("--since date is in the future")
		}
		return start, end, nil
	}

	if days < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("--days must be at least 1")
	}

	start := end.AddDate(0, 0, -(days - 1))
	return start, end, nil
}

func runSkills(cmd *cobra.Command, args []string) error {
	now := time.Now()
	start, end, err := skillsDateRange(now, skillsDays, skillsSince)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")

	// Collect skills per day
	var days []dayData
	var allSkills []jsonl.SkillCall

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		jsonlFiles, err := findJSONLFiles(projectsDir, d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error finding files for %s: %v\n", d.Format("2006-01-02"), err)
			continue
		}

		var daySkills []jsonl.SkillCall
		for _, path := range jsonlFiles {
			msgs, err := jsonl.ParseFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: error parsing %s: %v\n", filepath.Base(path), err)
				continue
			}
			daySkills = append(daySkills, collectSkillCalls(msgs)...)
		}

		if len(daySkills) > 0 {
			days = append(days, dayData{date: d, skills: daySkills})
			allSkills = append(allSkills, daySkills...)
		}
	}

	numDays := int(end.Sub(start).Hours()/24) + 1

	if len(allSkills) == 0 {
		fmt.Printf("No skill usage found: %s → %s (%d days)\n",
			start.Format("2006-01-02"), end.Format("2006-01-02"), numDays)
		return nil
	}

	output := formatSkillsReport(start, end, numDays, allSkills, days)
	fmt.Print(output)
	return nil
}

type dayData struct {
	date   time.Time
	skills []jsonl.SkillCall
}

func formatSkillsReport(start, end time.Time, numDays int, allSkills []jsonl.SkillCall, days []dayData) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Skill usage: %s → %s (%d days)\n\n",
		start.Format("2006-01-02"), end.Format("2006-01-02"), numDays)

	// Build frequency table
	counts := make(map[string]int)
	for _, sc := range allSkills {
		counts[sc.Skill]++
	}

	type entry struct {
		name  string
		count int
	}
	entries := make([]entry, 0, len(counts))
	for name, count := range counts {
		entries = append(entries, entry{name, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].name < entries[j].name
	})

	// Find column widths
	nameWidth := len("Skill")
	countWidth := len("Count")
	for _, e := range entries {
		if len(e.name) > nameWidth {
			nameWidth = len(e.name)
		}
		cw := len(fmt.Sprintf("%d", e.count))
		if cw > countWidth {
			countWidth = cw
		}
	}

	// Header
	fmt.Fprintf(&b, "  %-*s  %*s\n", nameWidth, "Skill", countWidth, "Count")
	fmt.Fprintf(&b, "  %-*s  %*s\n", nameWidth, strings.Repeat("─", nameWidth), countWidth, strings.Repeat("─", countWidth))

	// Rows
	for _, e := range entries {
		fmt.Fprintf(&b, "  %-*s  %*d\n", nameWidth, e.name, countWidth, e.count)
	}

	// Total
	fmt.Fprintf(&b, "\n  Total: %d invocations, %d unique skills\n", len(allSkills), len(counts))

	// Per-day breakdown (most recent first)
	if len(days) > 0 {
		b.WriteString("\n  Per day:\n")
		for i := len(days) - 1; i >= 0; i-- {
			dd := days[i]
			dayCounts := make(map[string]int)
			for _, sc := range dd.skills {
				dayCounts[sc.Skill]++
			}
			// Sort by count desc, then alpha
			type de struct {
				name  string
				count int
			}
			des := make([]de, 0, len(dayCounts))
			for name, count := range dayCounts {
				des = append(des, de{name, count})
			}
			sort.Slice(des, func(a, bIdx int) bool {
				if des[a].count != des[bIdx].count {
					return des[a].count > des[bIdx].count
				}
				return des[a].name < des[bIdx].name
			})
			var parts []string
			for _, d := range des {
				parts = append(parts, fmt.Sprintf("%s:%d", d.name, d.count))
			}
			fmt.Fprintf(&b, "  %s  %3d  (%s)\n", dd.date.Format("2006-01-02"), len(dd.skills), strings.Join(parts, ", "))
		}
	}

	return b.String()
}
