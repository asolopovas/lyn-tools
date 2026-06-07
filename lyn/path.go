package lyn

import (
	"os"
	"path/filepath"
	"strings"
)

func cleanUniquePaths(paths []string) []string {
	seen := newStringSet(len(paths))
	items := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if !seen.add(clean) {
			continue
		}
		items = append(items, clean)
	}
	return items
}

func windowsSystemDirs() []string {
	values := []string{os.Getenv("SystemRoot"), os.Getenv("WINDIR")}
	if strings.TrimSpace(values[0]) == "" && strings.TrimSpace(values[1]) == "" {
		values = append(values, `C:\Windows`)
	}
	return cleanUniquePaths(values)
}

func absoluteCleanPath(path string) string {
	full, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return full
}

func absoluteCleanPaths(paths []string) []string {
	items := make([]string, 0, len(paths))
	for _, path := range paths {
		items = append(items, absoluteCleanPath(path))
	}
	return cleanUniquePaths(items)
}
