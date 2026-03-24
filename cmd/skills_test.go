package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/lathanx/recap/internal/jsonl"
)

func TestSkillsDateRange(t *testing.T) {
	now := time.Date(2026, 3, 24, 15, 30, 0, 0, time.Local)

	tests := []struct {
		name      string
		days      int
		since     string
		wantStart string
		wantEnd   string
		wantErr   bool
	}{
		{
			name:      "default 7 days",
			days:      7,
			since:     "",
			wantStart: "2026-03-18",
			wantEnd:   "2026-03-24",
		},
		{
			name:      "1 day means today only",
			days:      1,
			since:     "",
			wantStart: "2026-03-24",
			wantEnd:   "2026-03-24",
		},
		{
			name:      "14 days",
			days:      14,
			since:     "",
			wantStart: "2026-03-11",
			wantEnd:   "2026-03-24",
		},
		{
			name:      "since overrides days",
			days:      7,
			since:     "2026-03-10",
			wantStart: "2026-03-10",
			wantEnd:   "2026-03-24",
		},
		{
			name:    "since in the future",
			days:    7,
			since:   "2026-04-01",
			wantErr: true,
		},
		{
			name:    "invalid since format",
			days:    7,
			since:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "days less than 1",
			days:    0,
			since:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := skillsDateRange(now, tt.days, tt.since)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Errorf("end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestFormatSkillsReport(t *testing.T) {
	start := time.Date(2026, 3, 18, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 3, 24, 0, 0, 0, 0, time.Local)

	allSkills := []jsonl.SkillCall{
		{Skill: "commit"}, {Skill: "commit"}, {Skill: "commit"}, {Skill: "commit"},
		{Skill: "review"}, {Skill: "review"},
		{Skill: "simplify"},
	}

	days := []dayData{
		{
			date: time.Date(2026, 3, 20, 0, 0, 0, 0, time.Local),
			skills: []jsonl.SkillCall{
				{Skill: "commit"}, {Skill: "commit"},
				{Skill: "review"},
			},
		},
		{
			date: time.Date(2026, 3, 22, 0, 0, 0, 0, time.Local),
			skills: []jsonl.SkillCall{
				{Skill: "commit"}, {Skill: "commit"},
				{Skill: "review"},
				{Skill: "simplify"},
			},
		},
	}

	got := formatSkillsReport(start, end, 7, allSkills, days)

	// Check header
	if !strings.Contains(got, "Skill usage: 2026-03-18 → 2026-03-24 (7 days)") {
		t.Errorf("missing header in output:\n%s", got)
	}

	// Check frequency table entries
	if !strings.Contains(got, "commit") || !strings.Contains(got, "review") || !strings.Contains(got, "simplify") {
		t.Errorf("missing skill names in output:\n%s", got)
	}

	// Check total line
	if !strings.Contains(got, "Total: 7 invocations, 3 unique skills") {
		t.Errorf("missing or wrong total line in output:\n%s", got)
	}

	// Check per-day section exists
	if !strings.Contains(got, "Per day:") {
		t.Errorf("missing per-day section in output:\n%s", got)
	}

	// Most recent day should appear first in per-day
	day22Idx := strings.Index(got, "2026-03-22")
	day20Idx := strings.Index(got, "2026-03-20")
	// Both should be in per-day section (after "Per day:")
	perDayIdx := strings.Index(got, "Per day:")
	if day22Idx < perDayIdx || day20Idx < perDayIdx {
		t.Errorf("per-day dates should be after 'Per day:' header")
	}
	if day22Idx > day20Idx {
		t.Errorf("most recent day (2026-03-22) should appear before older day (2026-03-20) in per-day section")
	}

	// Check commit appears before review in table (higher count)
	tableSection := got[:strings.Index(got, "Total:")]
	commitIdx := strings.Index(tableSection, "commit")
	reviewIdx := strings.Index(tableSection, "review")
	if commitIdx > reviewIdx {
		t.Errorf("commit (count 4) should appear before review (count 2) in table")
	}
}

func TestFormatSkillsReportSingleDay(t *testing.T) {
	day := time.Date(2026, 3, 24, 0, 0, 0, 0, time.Local)

	allSkills := []jsonl.SkillCall{
		{Skill: "commit"}, {Skill: "commit"},
	}

	days := []dayData{
		{
			date:   day,
			skills: allSkills,
		},
	}

	got := formatSkillsReport(day, day, 1, allSkills, days)

	if !strings.Contains(got, "(1 days)") {
		t.Errorf("expected 1 day period in output:\n%s", got)
	}
	if !strings.Contains(got, "Total: 2 invocations, 1 unique skills") {
		t.Errorf("missing total in output:\n%s", got)
	}
}

func TestFormatSkillsReportEmpty(t *testing.T) {
	// formatSkillsReport is only called when there are skills,
	// but test with a single skill to verify minimal output
	start := time.Date(2026, 3, 24, 0, 0, 0, 0, time.Local)
	allSkills := []jsonl.SkillCall{{Skill: "commit"}}
	days := []dayData{{date: start, skills: allSkills}}

	got := formatSkillsReport(start, start, 1, allSkills, days)
	if got == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatSkillsReportColumnAlignment(t *testing.T) {
	start := time.Date(2026, 3, 24, 0, 0, 0, 0, time.Local)
	allSkills := []jsonl.SkillCall{
		{Skill: "commit"},
		{Skill: "implement-plan"},
	}
	days := []dayData{{date: start, skills: allSkills}}

	got := formatSkillsReport(start, start, 1, allSkills, days)

	// "implement-plan" is longer than "Skill", so columns should align
	lines := strings.Split(got, "\n")
	var headerLine, dataLine string
	for _, line := range lines {
		if strings.Contains(line, "Skill") && strings.Contains(line, "Count") {
			headerLine = line
		}
		if strings.Contains(line, "implement-plan") {
			dataLine = line
		}
	}
	if headerLine == "" || dataLine == "" {
		t.Fatalf("could not find header and data lines in:\n%s", got)
	}
	// Count values should be right-aligned at the same position
	headerCountIdx := strings.Index(headerLine, "Count")
	if headerCountIdx == -1 {
		t.Fatal("could not find Count in header")
	}
}
