package lyn

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var applicationInstallerMarkers = []string{"uninstall", "install", "setup"}

func ScanApplications(ctx context.Context) ([]Project, error) {
	return scanApplications(ctx, applicationDirs(runtime.GOOS), runtime.GOOS)
}

func scanApplications(ctx context.Context, dirs []string, goos string) ([]Project, error) {
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	if err := addApplicationsFromDirs(ctx, seen, seenNames, dirs, goos); err != nil {
		return seen.sorted(), err
	}
	if goos == "windows" {
		for _, tool := range windowsSystemTools() {
			addApplication(seen, seenNames, tool)
		}
		if err := addWindowsStartApplications(ctx, seen, seenNames); err != nil {
			return seen.sorted(), err
		}
		if err := addWindowsPathApplications(ctx, seen, seenNames, windowsPathDirs()); err != nil {
			return seen.sorted(), err
		}
	}
	return seen.sorted(), nil
}

func scanApplicationDirs(ctx context.Context, dirs []string, goos string) ([]Project, error) {
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	if err := addApplicationsFromDirs(ctx, seen, seenNames, dirs, goos); err != nil {
		return seen.sorted(), err
	}
	return seen.sorted(), nil
}

func addApplicationsFromDirs(ctx context.Context, seen projectSet, seenNames stringSet, dirs []string, goos string) error {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || ctx.Err() != nil {
				return ctx.Err()
			}
			if entry.IsDir() {
				if isWindowsStartupDir(entry.Name(), goos) {
					return filepath.SkipDir
				}
				return nil
			}
			if app, ok := detectApplication(path, goos); ok {
				addApplication(seen, seenNames, app)
			}
			return nil
		})
		if err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

func applicationDirs(goos string) []string {
	switch goos {
	case "windows":
		return windowsApplicationDirs()
	case "linux":
		return linuxApplicationDirs()
	default:
		return nil
	}
}

func detectApplication(path string, goos string) (Project, bool) {
	switch goos {
	case "windows":
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".lnk" && ext != ".appref-ms" && ext != ".url" {
			return Project{}, false
		}
		if ext == ".url" && isWindowsWebInternetShortcut(path) {
			return Project{}, false
		}
		if ext == ".lnk" && !isWindowsShortcutGUIApplication(path) {
			return Project{}, false
		}
		return newProject(trimApplicationName(filepath.Base(path)), path, projectKindApp), true
	case "linux":
		if strings.ToLower(filepath.Ext(path)) != ".desktop" {
			return Project{}, false
		}
		name, ok := desktopApplicationName(path)
		if !ok {
			return Project{}, false
		}
		return newProject(name, path, projectKindApp), true
	default:
		return Project{}, false
	}
}

func addApplication(seen projectSet, seenNames stringSet, app Project) bool {
	if !applicationNameAllowed(app.Name) {
		return false
	}
	if !seenNames.addFold(app.Name) {
		return false
	}
	seen.add(app)
	return true
}

func applicationNameAllowed(name string) bool {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if lowerName == "" {
		return false
	}
	for _, marker := range applicationInstallerMarkers {
		if strings.Contains(lowerName, marker) {
			return false
		}
	}
	return true
}
