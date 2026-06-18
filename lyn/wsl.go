package lyn

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	wslLocalhostPrefix = `\\wsl.localhost\`
	wslLegacyPrefix    = `\\wsl$\`
)

var wslSystemDistros = map[string]bool{
	"docker-desktop":      true,
	"docker-desktop-data": true,
	"rancher-desktop":     true,
}

func isWSLSystemDistro(name string) bool {
	return wslSystemDistros[strings.ToLower(strings.TrimSpace(name))]
}

func stripWSLPrefix(path string) (string, bool) {
	for _, prefix := range []string{wslLocalhostPrefix, wslLegacyPrefix} {
		if len(path) >= len(prefix) && strings.EqualFold(path[:len(prefix)], prefix) {
			return path[len(prefix):], true
		}
	}
	return "", false
}

func wslUnixFromUNC(uncPath string) (distro string, unixPath string, ok bool) {
	rest, matched := stripWSLPrefix(strings.TrimSpace(uncPath))
	if !matched {
		return "", "", false
	}
	rest = strings.ReplaceAll(rest, `\`, "/")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return "", "", false
	}
	distro, tail, _ := strings.Cut(rest, "/")
	if distro == "" {
		return "", "", false
	}
	tail = strings.Trim(tail, "/")
	if tail == "" {
		return distro, "/", true
	}
	return distro, "/" + tail, true
}

func wslWindowsRoot(distro string, unixPath string) string {
	clean := strings.Trim(strings.ReplaceAll(unixPath, "/", `\`), `\`)
	if clean == "" {
		return wslLocalhostPrefix + distro
	}
	return wslLocalhostPrefix + distro + `\` + clean
}

func wslCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("wsl.exe", args...)
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	hideConsoleWindow(cmd)
	return cmd
}

func listWSLDistros() []string {
	out, err := wslCommand("-l", "-q").Output()
	if err != nil {
		return nil
	}
	var distros []string
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.Trim(strings.TrimSpace(line), "\x00\r")
		if name == "" || isWSLSystemDistro(name) {
			continue
		}
		distros = append(distros, name)
	}
	return distros
}

var (
	defaultDistroOnce  sync.Once
	defaultDistroValue string
)

func defaultWSLDistro() string {
	defaultDistroOnce.Do(func() {
		defaultDistroValue = resolveDefaultWSLDistro()
	})
	return defaultDistroValue
}

func resolveDefaultWSLDistro() string {
	out, err := wslCommand("-l", "-v").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			trimmed := strings.Trim(strings.TrimSpace(line), "\x00\r")
			if !strings.HasPrefix(trimmed, "*") {
				continue
			}
			fields := strings.Fields(strings.TrimPrefix(trimmed, "*"))
			if len(fields) > 0 && !isWSLSystemDistro(fields[0]) {
				return fields[0]
			}
		}
	}
	if distros := listWSLDistros(); len(distros) > 0 {
		return distros[0]
	}
	return ""
}

func resolveWSLDistro(distro string) string {
	if strings.TrimSpace(distro) != "" {
		return distro
	}
	return defaultWSLDistro()
}
