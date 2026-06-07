package lyn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIconMime(t *testing.T) {
	cases := map[string]string{
		"icon.png":  "image/png",
		"icon.svg":  "image/svg+xml",
		"icon.webp": "image/webp",
		"icon.ico":  "",
	}
	for path, want := range cases {
		if got := iconMime(path); got != want {
			t.Fatalf("iconMime(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestIconDataURI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(path, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	value, err := iconDataURI(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(value, "data:image/png;base64,") {
		t.Fatalf("unexpected data uri %q", value)
	}
}
