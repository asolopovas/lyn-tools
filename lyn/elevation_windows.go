//go:build windows

package lyn

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowsShellShowNormal = 1

var elevationShellExecute = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

func init() {
	detectElevationStatus = detectWindowsElevationStatus
	startElevationProcess = startWindowsElevationProcess
}

func detectWindowsElevationStatus() ElevationStatus {
	if windows.GetCurrentProcessToken().IsElevated() {
		return ElevationStatus{Mode: elevationModeAdmin, CanSwitch: true, Message: "Running as administrator."}
	}
	return ElevationStatus{Mode: elevationModeStandard, CanSwitch: true, Message: "Running in standard mode."}
}

func startWindowsElevationProcess(mode string, args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args = restartArgs(args)
	parameters := windowsCommandLine(args)
	workingDirectory := filepath.Dir(exe)
	if mode == elevationModeAdmin {
		return shellExecuteProcess("runas", exe, parameters, workingDirectory)
	}
	return shellExecuteProcess("open", "explorer.exe", windowsCommandLine(append([]string{exe}, args...)), workingDirectory)
}

func shellExecuteProcess(operation string, target string, parameters string, workingDirectory string) error {
	operationPtr, err := syscall.UTF16PtrFromString(operation)
	if err != nil {
		return err
	}
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	parametersPtr := uintptr(0)
	if parameters != "" {
		value, err := syscall.UTF16PtrFromString(parameters)
		if err != nil {
			return err
		}
		parametersPtr = uintptr(unsafe.Pointer(value))
	}
	workingDirectoryPtr := uintptr(0)
	if workingDirectory != "" {
		value, err := syscall.UTF16PtrFromString(workingDirectory)
		if err != nil {
			return err
		}
		workingDirectoryPtr = uintptr(unsafe.Pointer(value))
	}
	ret, _, callErr := elevationShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(operationPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		parametersPtr,
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

func windowsCommandLine(args []string) string {
	if len(args) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(args))
	for _, arg := range args {
		escaped = append(escaped, syscall.EscapeArg(arg))
	}
	return strings.Join(escaped, " ")
}
