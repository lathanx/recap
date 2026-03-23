package log

import (
	"strings"
	"testing"
	"time"
)

func TestRecapDirUsesRecapPath(t *testing.T) {
	dir := recapDir()
	if !strings.Contains(dir, ".recap") {
		t.Errorf("recapDir() = %q, want path containing .recap", dir)
	}
	if strings.Contains(dir, ".bartlebee") {
		t.Errorf("recapDir() = %q, should not contain .bartlebee", dir)
	}
}

func TestDailyLogPathUsesRecapPath(t *testing.T) {
	ts := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	path := DailyLogPath(ts)
	if !strings.Contains(path, ".recap") {
		t.Errorf("DailyLogPath() = %q, want path containing .recap", path)
	}
	if strings.Contains(path, ".bartlebee") {
		t.Errorf("DailyLogPath() = %q, should not contain .bartlebee", path)
	}
}
