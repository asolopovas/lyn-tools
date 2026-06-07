package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoSingleFileNonDocumentationFolders(t *testing.T) {
	allowed := map[string]bool{
		".git":                            true,
		"build":                           true,
		filepath.Join("build", "bin"):     true,
		"frontend":                        true,
		filepath.Join("frontend", "src"):  true,
		filepath.Join("frontend", "dist"): true,
		filepath.Join("frontend", "node_modules"): true,
	}
	if err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == "." {
			return nil
		}
		clean := strings.TrimPrefix(filepath.Clean(path), "."+string(filepath.Separator))
		base := filepath.Base(clean)
		if base == ".git" || base == "node_modules" || clean == filepath.Join("frontend", "wailsjs") || strings.HasPrefix(clean, "docs") {
			return filepath.SkipDir
		}
		if allowed[clean] {
			return nil
		}
		count := 0
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, item := range entries {
			if !item.IsDir() {
				count++
			}
		}
		if count == 1 {
			t.Fatalf("single-file non-documentation folder %s", clean)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
