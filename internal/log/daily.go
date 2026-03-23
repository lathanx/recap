package log

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func recapDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".recap")
	}
	return filepath.Join(home, ".recap")
}

func DailyLogPath(t time.Time) string {
	return filepath.Join(recapDir(), "logs", "daily", t.Format("2006-01-02")+".md")
}

// AppendToDaily appends an entry to today's daily log file with flock-based locking.
func AppendToDaily(t time.Time, entry string) error {
	logPath := DailyLogPath(t)

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquiring file lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Write header if file is empty
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat log file: %w", err)
	}
	if info.Size() == 0 {
		header := fmt.Sprintf("# Daily Log: %s\n", t.Format("2006-01-02"))
		if _, err := f.WriteString(header); err != nil {
			return fmt.Errorf("writing header: %w", err)
		}
	}

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	return nil
}
