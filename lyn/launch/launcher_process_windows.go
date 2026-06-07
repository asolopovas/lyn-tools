//go:build windows

package launch

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const (
	windowsDetachedProcess = 0x00000008
	windowsShellShowNormal = 1
)

var windowsShellExecute = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

func startLaunchCommand(path string, cmd launchCommand, action string) error {
	path = strings.TrimSpace(path)
	if windowsUsesShellExecute(path, action) {
		target := path
		operation := "open"
		workingDirectory := ""
		switch action {
		case "reveal":
			target = containingLocation(path, "windows")
		case "run-admin":
			operation = "runAs"
			workingDirectory = containingLocation(path, "windows")
		case "run-user":
			operation = "runAsUser"
			workingDirectory = containingLocation(path, "windows")
		}
		return shellExecute(target, operation, workingDirectory)
	}
	process := exec.Command(cmd.Name, cmd.Args...)
	return startBackgroundProcess(process, action)
}

func startBackgroundProcess(process *exec.Cmd, action string) error {
	configureLaunchProcess(process, action)
	if err := process.Start(); err != nil {
		return err
	}
	if err := process.Process.Release(); err != nil {
		return fmt.Errorf("release launched process: %w", err)
	}
	return nil
}

func configureLaunchProcess(process *exec.Cmd, action string) {
	if action == "terminal" || action == "code" {
		return
	}
	process.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windowsDetachedProcess,
	}
}

func windowsUsesShellExecute(path string, action string) bool {
	return (action == "open" || action == "reveal" || action == "run-admin" || action == "run-user") && path != "" && !isUnixPath(path)
}

func shellExecute(path string, operation string, workingDirectory string) error {
	operationPtr, err := syscall.UTF16PtrFromString(operation)
	if err != nil {
		return err
	}
	target, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	workingDirectoryPtr := uintptr(0)
	if workingDirectory != "" {
		dir, err := syscall.UTF16PtrFromString(workingDirectory)
		if err != nil {
			return err
		}
		workingDirectoryPtr = uintptr(unsafe.Pointer(dir))
	}
	ret, _, callErr := windowsShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(operationPtr)),
		uintptr(unsafe.Pointer(target)),
		0,
		workingDirectoryPtr,
		uintptr(windowsShellShowNormal),
	)
	if ret <= 32 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.Errno(ret)
	}
	return nil
}
