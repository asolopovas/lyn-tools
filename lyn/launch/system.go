package launch

import (
	"errors"
	"os/user"
	"strings"
)

const AdminToolPrefix = "lyn:system:admin:"

var lookupUsername = currentUsername

var adminScriptPath string

func SetAdminScriptPath(path string) {
	adminScriptPath = path
}

func systemCommand(path string, goos string) (launchCommand, error) {
	key := strings.ToLower(strings.TrimSpace(path))
	if tool, ok := strings.CutPrefix(key, AdminToolPrefix); ok {
		return adminToolCommand(tool)
	}
	switch goos {
	case "windows":
		switch key {
		case "lyn:system:restart":
			return launchCommand{Name: "shutdown.exe", Args: []string{"/r", "/t", "0"}}, nil
		case "lyn:system:shutdown":
			return launchCommand{Name: "shutdown.exe", Args: []string{"/s", "/t", "0"}}, nil
		case "lyn:system:logout":
			return launchCommand{Name: "shutdown.exe", Args: []string{"/l"}}, nil
		}
	case "darwin":
		switch key {
		case "lyn:system:restart":
			return osascriptCommand("restart"), nil
		case "lyn:system:shutdown":
			return osascriptCommand("shut down"), nil
		case "lyn:system:logout":
			return osascriptCommand("log out"), nil
		}
	default:
		switch key {
		case "lyn:system:restart":
			return launchCommand{Name: "systemctl", Args: []string{"reboot"}}, nil
		case "lyn:system:shutdown":
			return launchCommand{Name: "systemctl", Args: []string{"poweroff"}}, nil
		case "lyn:system:logout":
			name := lookupUsername()
			if name == "" {
				return launchCommand{}, errors.New("log out is unavailable: no active user session")
			}
			return launchCommand{Name: "loginctl", Args: []string{"terminate-user", name}}, nil
		}
	}
	return launchCommand{}, errors.New("unknown system command")
}

func adminToolCommand(tool string) (launchCommand, error) {
	if strings.TrimSpace(adminScriptPath) == "" {
		return launchCommand{}, errors.New("system tools are unavailable: helper script is missing")
	}
	return launchCommand{Name: "x-terminal-emulator", Args: []string{"-e", adminScriptPath, tool}}, nil
}

func osascriptCommand(verb string) launchCommand {
	return launchCommand{Name: "osascript", Args: []string{"-e", `tell application "System Events" to ` + verb}}
}

func currentUsername() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}
