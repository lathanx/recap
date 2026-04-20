package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lathanx/recap/internal/server"
)

// setupTestDir creates a temp data dir with daily log files and returns the path.
func setupTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	dailyDir := filepath.Join(dir, "logs", "daily")
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dailyDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestListDays_ReturnsDaysWithEntries(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md": "# Daily Log: 2026-03-28\n\n## Session 1\nDid stuff\n\n## Session 2\nMore stuff\n",
		"2026-03-29.md": "# Daily Log: 2026-03-29\n\n## Session 1\nOne session\n",
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var days []server.DayEntry
	if err := json.NewDecoder(rec.Body).Decode(&days); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(days) != 2 {
		t.Fatalf("got %d days, want 2", len(days))
	}

	// Days should be sorted descending (newest first)
	if days[0].Date != "2026-03-29" {
		t.Errorf("days[0].Date = %q, want %q", days[0].Date, "2026-03-29")
	}
	if days[1].Date != "2026-03-28" {
		t.Errorf("days[1].Date = %q, want %q", days[1].Date, "2026-03-28")
	}

	// Check sessions count for multi-session day
	if days[1].Sessions != 2 {
		t.Errorf("days[1].Sessions = %d, want 2", days[1].Sessions)
	}

	// Size should be > 0
	if days[0].Size <= 0 {
		t.Errorf("days[0].Size = %d, want > 0", days[0].Size)
	}
}

func TestListDays_EmptyDirectory(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var days []server.DayEntry
	if err := json.NewDecoder(rec.Body).Decode(&days); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(days) != 0 {
		t.Fatalf("got %d days, want 0", len(days))
	}
}

func TestListDays_WithSummaryFlag(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md":         "# Daily Log: 2026-03-28\n\n## Session 1\nWork\n",
		"2026-03-28.summary.md": "Summary of the day.\n",
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var days []server.DayEntry
	if err := json.NewDecoder(rec.Body).Decode(&days); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(days) != 1 {
		t.Fatalf("got %d days, want 1", len(days))
	}
	if !days[0].HasSummary {
		t.Error("days[0].HasSummary = false, want true")
	}
}

func TestGetDay_ReturnsContent(t *testing.T) {
	content := "# Daily Log: 2026-03-28\n\n## Session 1\nDid some work\n"
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md": content,
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days/2026-03-28", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days/2026-03-28: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var day server.DayDetail
	if err := json.NewDecoder(rec.Body).Decode(&day); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if day.Date != "2026-03-28" {
		t.Errorf("day.Date = %q, want %q", day.Date, "2026-03-28")
	}
	if day.Content != content {
		t.Errorf("day.Content = %q, want %q", day.Content, content)
	}
	if day.Summary != "" {
		t.Errorf("day.Summary = %q, want empty", day.Summary)
	}
}

func TestGetDay_WithSummary(t *testing.T) {
	content := "# Daily Log: 2026-03-28\n\n## Session 1\nWork\n"
	summary := "Summary of the day.\n"
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md":         content,
		"2026-03-28.summary.md": summary,
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days/2026-03-28", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days/2026-03-28: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var day server.DayDetail
	if err := json.NewDecoder(rec.Body).Decode(&day); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if day.Summary != summary {
		t.Errorf("day.Summary = %q, want %q", day.Summary, summary)
	}
}

func TestGetDay_NotFound(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days/2026-03-28", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/days/2026-03-28: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetDay_EmptyDate(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET /api/days/: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestListDays_IgnoresNonMarkdownFiles(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md":  "# Daily Log: 2026-03-28\n\n## Session 1\nWork\n",
		"notes.txt":      "random file",
		".hidden":        "hidden file",
		"2026-03-28.summary.md": "Summary\n",
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var days []server.DayEntry
	if err := json.NewDecoder(rec.Body).Decode(&days); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Should only list the one daily log, not summary files or non-md files
	if len(days) != 1 {
		t.Fatalf("got %d days, want 1", len(days))
	}
	if days[0].Date != "2026-03-28" {
		t.Errorf("days[0].Date = %q, want %q", days[0].Date, "2026-03-28")
	}
}

func TestGetDay_ContentTypeJSON(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md": "# Daily Log: 2026-03-28\n",
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days/2026-03-28", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestListDays_JSONFieldNames(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{
		"2026-03-28.md":         "# Daily Log: 2026-03-28\n\n## Session 1\nWork\n",
		"2026-03-28.summary.md": "Summary of the day.\n",
	})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/days: status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()

	// The frontend expects camelCase JSON keys (matching the hunny sister project).
	if !strings.Contains(body, `"hasSummary"`) {
		t.Errorf("JSON body should contain camelCase key \"hasSummary\", got: %s", body)
	}
	if strings.Contains(body, `"has_summary"`) {
		t.Errorf("JSON body should NOT contain snake_case key \"has_summary\", got: %s", body)
	}
}

func TestListDays_ContentTypeJSON(t *testing.T) {
	dataDir := setupTestDir(t, map[string]string{})

	handler := server.New(dataDir, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/days", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}
