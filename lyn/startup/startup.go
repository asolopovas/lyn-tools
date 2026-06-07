package startup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsupported = errors.New("startup is unsupported on this platform")

var configureStartup = unsupportedStartup

func Configure(enabled bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	return configureStartup(enabled, exe)
}

func unsupportedStartup(enabled bool, _ string) error {
	if enabled {
		return ErrUnsupported
	}
	return nil
}

func windowsStartupValue(exe string) string {
	return `"` + exe + `" --start-hidden`
}

func linuxDesktopEntry(exe string) string {
	return strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=Lyn",
		"Comment=Project launcher",
		"Exec=" + desktopExecValue(exe),
		"Terminal=false",
		"X-GNOME-Autostart-enabled=true",
		"",
	}, "\n")
}

func desktopExecValue(exe string) string {
	return `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
}
