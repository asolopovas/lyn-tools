package lyn

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type desktopEntryReader func(string) map[string]string

var (
	windowsAppIcon = emptyWindowsAppIcon
	linuxAppIcon   = emptyLinuxAppIcon
)

func (a *App) Icon(path string) (string, error) {
	ctx, cfg, _ := a.snapshot()
	return resolveIcon(ctx, cfg.Cache.Dir, path, runtime.GOOS, readDesktopEntry)
}

func resolveIcon(ctx context.Context, cacheDir string, path string, goos string, readDesktopEntry desktopEntryReader) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	switch goos {
	case "windows":
		return windowsAppIcon(ctx, filepath.Join(cacheDir, "icons"), path)
	case "linux":
		return linuxAppIcon(path, readDesktopEntry)
	default:
		return "", nil
	}
}

func emptyWindowsAppIcon(context.Context, string, string) (string, error) {
	return "", nil
}

func emptyLinuxAppIcon(string, desktopEntryReader) (string, error) {
	return "", nil
}

func iconDataURI(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mime := iconMime(path)
	if mime == "" {
		return "", nil
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func iconMime(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	default:
		return ""
	}
}
