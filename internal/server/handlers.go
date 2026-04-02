package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (s *apiServer) dailyDir() string {
	return filepath.Join(s.dataDir, "logs", "daily")
}

func countSessions(content []byte) int {
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "## Session ") {
			count++
		}
	}
	return count
}

func (s *apiServer) handleListDays(w http.ResponseWriter, r *http.Request) {
	dir := s.dailyDir()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	days := []DayEntry{}
	summaries := map[string]bool{}

	// First pass: identify summary files
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".summary.md") {
			date := strings.TrimSuffix(name, ".summary.md")
			summaries[date] = true
		}
	}

	// Second pass: build day entries from non-summary .md files
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".summary.md") {
			continue
		}
		date := strings.TrimSuffix(name, ".md")
		info, err := e.Info()
		if err != nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		days = append(days, DayEntry{
			Date:       date,
			Size:       info.Size(),
			Sessions:   countSessions(content),
			HasSummary: summaries[date],
		})
	}

	sort.Slice(days, func(i, j int) bool {
		return days[i].Date > days[j].Date
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(days)
}

func (s *apiServer) handleGetDay(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimPrefix(r.URL.Path, "/api/days/")
	if date == "" {
		http.Error(w, "missing date", http.StatusBadRequest)
		return
	}

	dir := s.dailyDir()
	logPath := filepath.Join(dir, date+".md")
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detail := DayDetail{
		Date:    date,
		Content: string(content),
	}

	summaryPath := filepath.Join(dir, date+".summary.md")
	if summaryContent, err := os.ReadFile(summaryPath); err == nil {
		detail.Summary = string(summaryContent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (s *apiServer) handleSync(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimPrefix(r.URL.Path, "/api/sync/")
	if date == "" {
		http.Error(w, "missing date", http.StatusBadRequest)
		return
	}
	cmd := exec.Command(s.recapBin, "sync", "--date", date)
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "output": string(output), "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "output": string(output)})
}

func (s *apiServer) handleSummarize(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimPrefix(r.URL.Path, "/api/summarize/")
	if date == "" {
		http.Error(w, "missing date", http.StatusBadRequest)
		return
	}
	cmd := exec.Command(s.recapBin, "summarize", "--date", date)
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "output": string(output), "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "output": string(output)})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
