package lyn

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func linuxApplicationDirs() []string {
	home, _ := os.UserHomeDir()
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	dataDirs := strings.TrimSpace(os.Getenv("XDG_DATA_DIRS"))
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	roots := append([]string{dataHome}, filepath.SplitList(dataDirs)...)
	dirs := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		dir := filepath.Clean(filepath.Join(root, "applications"))
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	return dirs
}

func desktopApplicationName(path string) (string, bool) {
	entry := readDesktopEntry(path)
	if strings.EqualFold(entry["NoDisplay"], "true") || strings.EqualFold(entry["Hidden"], "true") {
		return "", false
	}
	if !desktopShownIn(entry, currentDesktops()) {
		return "", false
	}
	if tryExec := strings.TrimSpace(entry["TryExec"]); tryExec != "" {
		if _, err := exec.LookPath(tryExec); err != nil {
			return "", false
		}
	}
	name := strings.TrimSpace(entry["Name"])
	return name, name != ""
}

func currentDesktops() []string {
	return splitDesktopList(os.Getenv("XDG_CURRENT_DESKTOP"), ":")
}

func desktopShownIn(entry map[string]string, current []string) bool {
	if only := splitDesktopList(entry["OnlyShowIn"], ";"); len(only) > 0 && !desktopListsIntersect(only, current) {
		return false
	}
	if not := splitDesktopList(entry["NotShowIn"], ";"); len(not) > 0 && desktopListsIntersect(not, current) {
		return false
	}
	return true
}

func splitDesktopList(value, sep string) []string {
	var out []string
	for _, part := range strings.Split(value, sep) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func desktopListsIntersect(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if strings.EqualFold(x, y) {
				return true
			}
		}
	}
	return false
}

func readDesktopEntry(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	entry := map[string]string{}
	inSection := false
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "[Desktop Entry]" {
			inSection = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = false
			continue
		}
		if !inSection || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		entry[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return entry
}
