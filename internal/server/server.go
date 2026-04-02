package server

import (
	"io/fs"
	"net/http"
	"strings"
)

// DayEntry represents a single day in the listing.
type DayEntry struct {
	Date       string `json:"date"`
	Size       int64  `json:"size"`
	Sessions   int    `json:"sessions"`
	HasSummary bool   `json:"hasSummary"`
}

// DayDetail represents the full content of a single day.
type DayDetail struct {
	Date    string `json:"date"`
	Content string `json:"content"`
	Summary string `json:"summary,omitempty"`
}

// New creates an http.Handler serving the recap API and optional static files.
// If recapBin is non-empty, sync/summarize endpoints are enabled.
func New(dataDir string, staticFS fs.FS, recapBin string) http.Handler {
	mux := http.NewServeMux()
	s := &apiServer{dataDir: dataDir, recapBin: recapBin}
	mux.HandleFunc("GET /api/days", s.handleListDays)
	mux.HandleFunc("GET /api/days/", s.handleGetDay)
	mux.HandleFunc("POST /api/sync/", s.handleSync)
	mux.HandleFunc("POST /api/summarize/", s.handleSummarize)

	if staticFS != nil {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && !strings.Contains(r.URL.Path, ".") {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	return mux
}

type apiServer struct {
	dataDir  string
	recapBin string
}
