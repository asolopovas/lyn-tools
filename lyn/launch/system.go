package launch

import (
	"errors"
	"os/user"
	"strings"
)

var lookupUsername = currentUsername

func systemCommand(path string, goos string) (launchCommand, error) {
	key := strings.ToLower(strings.TrimSpace(path))
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

func osascriptCommand(verb string) launchCommand {
	return launchCommand{Name: "osascript", Args: []string{"-e", `tell application "System Events" to ` + verb}}
}

func currentUsername() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}
