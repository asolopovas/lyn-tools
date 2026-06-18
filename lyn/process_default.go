//go:build !windows

package lyn

import "os/exec"

func hideConsoleWindow(*exec.Cmd) {}
