//go:build linux

package startup

import (
	"os"
	"path/filepath"
)

func init() {
	configureStartup = configureLinuxStartup
}

func configureLinuxStartup(enabled bool, exe string) error {
	if enabled {
		path, err := linuxAutostartPath()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(linuxDesktopEntry(exe)), 0o644)
	}
	path, err := linuxAutostartPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func linuxAutostartPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || home == "" {
			if err != nil {
				return "", err
			}
			return "", homeErr
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "lyn.desktop"), nil
}
