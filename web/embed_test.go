package web

import (
	"io/fs"
	"testing"
)

func TestStaticFilesReturnsNonNil(t *testing.T) {
	got := StaticFiles()
	if got == nil {
		t.Fatal("StaticFiles() returned nil")
	}
}

func TestStaticFilesOpensIndexHTML(t *testing.T) {
	fsys := StaticFiles()
	f, err := fsys.Open("index.html")
	if err != nil {
		t.Fatalf("failed to open index.html: %v", err)
	}
	f.Close()
}

func TestStaticFilesOpensStyleCSS(t *testing.T) {
	fsys := StaticFiles()
	f, err := fsys.Open("style.css")
	if err != nil {
		t.Fatalf("failed to open style.css: %v", err)
	}
	f.Close()
}

func TestStaticFilesOpensAppJS(t *testing.T) {
	fsys := StaticFiles()
	f, err := fsys.Open("app.js")
	if err != nil {
		t.Fatalf("failed to open app.js: %v", err)
	}
	f.Close()
}

func TestStaticFilesOpensMarkedMinJS(t *testing.T) {
	fsys := StaticFiles()
	f, err := fsys.Open("vendor/marked.min.js")
	if err != nil {
		t.Fatalf("failed to open vendor/marked.min.js: %v", err)
	}
	f.Close()
}

// Verify the return type satisfies fs.FS at compile time.
var _ fs.FS = StaticFiles()
