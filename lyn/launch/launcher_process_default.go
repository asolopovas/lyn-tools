//go:build !windows

package launch

import "os/exec"

func startLaunchCommand(_ string, cmd launchCommand, _ string) error {
	return exec.Command(cmd.Name, cmd.Args...).Start()
}
