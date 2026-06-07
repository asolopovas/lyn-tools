//go:build linux

package lyn

import (
	"os"
	"path/filepath"
)

func init() {
	linuxAppIcon = resolveLinuxAppIcon
}

func resolveLinuxAppIcon(path string, readDesktopEntry desktopEntryReader) (string, error) {
	name := desktopApplicationIcon(path, readDesktopEntry)
	if name == "" {
		return "", nil
	}
	if filepath.IsAbs(name) {
		return iconDataURI(name)
	}
	for _, candidate := range linuxIconCandidates(name) {
		if _, err := os.Stat(candidate); err == nil {
			return iconDataURI(candidate)
		}
	}
	return "", nil
}

func desktopApplicationIcon(path string, readDesktopEntry desktopEntryReader) string {
	if readDesktopEntry == nil {
		return ""
	}
	return readDesktopEntry(path)["Icon"]
}

func linuxIconCandidates(name string) []string {
	home, _ := os.UserHomeDir()
	roots := []string{
		filepath.Join(home, ".local", "share", "icons"),
		"/usr/share/icons",
		"/usr/local/share/icons",
		"/usr/share/pixmaps",
	}
	sizes := []string{"256x256", "128x128", "64x64", "48x48", "32x32", "scalable"}
	exts := []string{".png", ".svg", ".webp", ".jpg"}
	items := make([]string, 0, len(roots)*len(sizes)*len(exts))
	for _, root := range roots {
		for _, size := range sizes {
			for _, ext := range exts {
				items = append(items, filepath.Join(root, "hicolor", size, "apps", name+ext))
				items = append(items, filepath.Join(root, "Adwaita", size, "apps", name+ext))
			}
		}
		for _, ext := range exts {
			items = append(items, filepath.Join(root, name+ext))
		}
	}
	return items
}
