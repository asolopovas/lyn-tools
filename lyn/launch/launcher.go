package launch

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strings"
)

type Request struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Distro string `json:"distro,omitempty"`
}

type Result struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Error   string   `json:"error,omitempty"`
}

type launchCommand struct {
	Name string
	Args []string
}

var startCommand = startLaunchCommand

func Launch(req Request) Result {
	action := NormalizedAction(req.Action)
	cmd, err := BuildLaunchCommand(req, runtime.GOOS)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if err := startCommand(req.Path, cmd, action); err != nil {
		return Result{Command: cmd.Name, Args: cmd.Args, Error: err.Error()}
	}
	return Result{Command: cmd.Name, Args: cmd.Args}
}

func BuildLaunchCommand(req Request, goos string) (launchCommand, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return launchCommand{}, errors.New("launch path is required")
	}
	action := NormalizedAction(req.Action)
	if strings.HasPrefix(strings.ToLower(path), "lyn:system:") {
		return launchCommand{}, errors.New("system commands are disabled")
	}
	if goos == "windows" && isWindowsPackagedAppPath(path) && action != "open" && action != "code" {
		action = "open"
	}
	switch action {
	case "code":
		return codeCommand(path, req.Distro, goos), nil
	case "open":
		return openCommand(path, req.Distro, goos), nil
	case "reveal":
		return revealCommand(path, req.Distro, goos), nil
	case "run-admin", "run-user":
		return elevatedCommand(path, goos, action)
	case "terminal":
		return terminalCommand(path, req.Distro, goos), nil
	default:
		return launchCommand{}, errors.New("unsupported launch action")
	}
}

func wslArgs(distro string, rest ...string) []string {
	if d := strings.TrimSpace(distro); d != "" {
		return append([]string{"-d", d}, rest...)
	}
	return rest
}

func wslRemote(distro string) string {
	if d := strings.TrimSpace(distro); d != "" {
		return "wsl+" + d
	}
	return "wsl+default"
}

func NormalizedAction(action string) string {
	value := strings.TrimSpace(strings.ToLower(action))
	if value == "" {
		return "open"
	}
	return value
}

func codeCommand(path string, distro string, goos string) launchCommand {
	name := "code"
	if goos == "windows" {
		name = windowsCodeCommandName()
	}
	if parsed, ok := parseVSCodeRemoteURI(path); ok {
		if isVSCodeCLIPathRemote(parsed.Host) {
			return launchCommand{Name: name, Args: []string{"--remote", parsed.Host, parsed.Path}}
		}
		flag := "--folder-uri"
		if strings.EqualFold(pathpkg.Ext(parsed.Path), ".code-workspace") {
			flag = "--file-uri"
		}
		return launchCommand{Name: name, Args: []string{flag, path}}
	}
	if goos == "windows" && isUnixPath(path) {
		return launchCommand{Name: name, Args: []string{"--remote", wslRemote(distro), path}}
	}
	return launchCommand{Name: name, Args: []string{path}}
}

func isVSCodeCLIPathRemote(authority string) bool {
	return strings.HasPrefix(authority, "ssh-remote+") || strings.HasPrefix(authority, "wsl+")
}

func parseVSCodeRemoteURI(path string) (*url.URL, bool) {
	parsed, err := url.Parse(path)
	return parsed, err == nil && parsed.Scheme == "vscode-remote" && parsed.Host != ""
}

func windowsCodeCommandName() string {
	path, err := exec.LookPath("code")
	if err != nil {
		return "code"
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return path
	}
	name := "Code.exe"
	if strings.EqualFold(filepath.Base(path), "code-insiders"+ext) {
		name = "Code - Insiders.exe"
	}
	exe := filepath.Join(filepath.Dir(filepath.Dir(path)), name)
	if _, err := os.Stat(exe); err == nil {
		return exe
	}
	return path
}

func isWindowsPackagedAppPath(path string) bool {
	return strings.HasPrefix(path, `shell:AppsFolder\`)
}

func openCommand(path string, distro string, goos string) launchCommand {
	switch goos {
	case "windows":
		if isUnixPath(path) {
			return launchCommand{Name: "wsl.exe", Args: wslArgs(distro, "sh", "-lc", "explorer.exe \"$(wslpath -w \"$1\")\"", "sh", path)}
		}
		if isWindowsAppShortcut(path) {
			return launchCommand{Name: "rundll32.exe", Args: []string{"url.dll,FileProtocolHandler", path}}
		}
		return launchCommand{Name: "explorer.exe", Args: []string{path}}
	case "darwin":
		return launchCommand{Name: "open", Args: []string{path}}
	default:
		if strings.EqualFold(filepath.Ext(path), ".desktop") {
			return launchCommand{Name: "gtk-launch", Args: []string{strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))}}
		}
		return launchCommand{Name: "xdg-open", Args: []string{path}}
	}
}

func revealCommand(path string, distro string, goos string) launchCommand {
	switch goos {
	case "windows":
		if isUnixPath(path) {
			return launchCommand{Name: "wsl.exe", Args: wslArgs(distro, "sh", "-lc", "explorer.exe \"$(wslpath -w \"$(dirname \"$1\")\")\"", "sh", path)}
		}
		return launchCommand{Name: "explorer.exe", Args: []string{containingLocation(path, goos)}}
	case "darwin":
		return launchCommand{Name: "open", Args: []string{containingLocation(path, goos)}}
	default:
		return launchCommand{Name: "xdg-open", Args: []string{containingLocation(path, goos)}}
	}
}

func containingLocation(path string, goos string) string {
	if goos == "windows" {
		dir := filepath.Dir(path)
		if dir == "." || dir == path {
			return path
		}
		return dir
	}
	dir := pathpkg.Dir(filepath.ToSlash(path))
	if dir == "." || dir == path {
		return path
	}
	return dir
}

func elevatedCommand(path string, goos string, action string) (launchCommand, error) {
	verb := "runAs"
	description := "run as administrator"
	if action == "run-user" {
		verb = "runAsUser"
		description = "run as different user"
	}
	if goos != "windows" || isUnixPath(path) {
		return launchCommand{}, errors.New(description + " is only supported for local Windows applications")
	}
	return launchCommand{Name: "ShellExecuteW", Args: []string{verb, path}}, nil
}

func terminalCommand(path string, distro string, goos string) launchCommand {
	dir := terminalWorkingDirectory(path, goos)
	switch goos {
	case "windows":
		if isUnixPath(dir) {
			return launchCommand{Name: "wsl.exe", Args: wslArgs(distro, "--cd", dir)}
		}
		return launchCommand{Name: "wt.exe", Args: []string{"-d", dir}}
	case "darwin":
		return launchCommand{Name: "open", Args: []string{"-a", "Terminal", dir}}
	default:
		return launchCommand{Name: "x-terminal-emulator", Args: []string{"--working-directory", dir}}
	}
}

func terminalWorkingDirectory(path string, goos string) string {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return containingLocation(path, goos)
	}
	return path
}

func isWindowsAppShortcut(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".lnk", ".appref-ms", ".url":
		return true
	default:
		return false
	}
}

func isUnixPath(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), "/")
}
