package lyn

import (
	"os"
	"path/filepath"
	"strings"
)

var skippedScanDirs = map[string]struct{}{
	".git": {}, "node_modules": {}, "vendor": {}, ".next": {},
	"dist": {}, "build": {}, "target": {}, ".cache": {},
	"uploads": {}, "wp-includes": {}, ".svn": {},
}

func shouldSkip(name string) bool {
	if _, ok := skippedScanDirs[name]; ok {
		return true
	}
	return isPackagedDependency(name)
}

func isPackagedDependency(name string) bool {
	for i := 0; i+1 < len(name); i++ {
		if name[i] == '@' && name[i+1] >= '0' && name[i+1] <= '9' {
			return true
		}
	}
	return false
}

func isWindowsStartupDir(name string, goos string) bool {
	return goos == "windows" && strings.EqualFold(name, "Startup")
}

func withinWindowsSystemDir(path string) bool {
	clean := absoluteCleanPath(path)
	for _, root := range absoluteCleanPaths(windowsSystemDirs()) {
		if isPathWithin(clean, root) {
			return true
		}
	}
	return false
}

func isPathWithin(path string, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if strings.EqualFold(path, root) {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

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
