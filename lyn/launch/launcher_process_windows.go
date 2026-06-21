//go:build windows

package launch

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	windowsDetachedProcess   = 0x00000008
	windowsShellShowNormal   = 1
	logonWithProfile         = 0x00000001
	createUnicodeEnvironment = 0x00000400
)

var (
	windowsShellExecute          = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	procCreateProcessWithToken   = syscall.NewLazyDLL("advapi32.dll").NewProc("CreateProcessWithTokenW")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procGetShellWindow           = user32.NewProc("GetShellWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

const windowsAllowForegroundAny = ^uintptr(0)

var errNoShellWindow = errors.New("no shell window available to launch as the user")

func startLaunchCommand(path string, cmd launchCommand, action string) error {
	path = strings.TrimSpace(path)
	if windowsUsesShellExecute(path, action) {
		switch action {
		case "run-admin":
			return shellExecute(path, "runAs", containingLocation(path, "windows"))
		case "run-user":
			return shellExecute(path, "runAsUser", containingLocation(path, "windows"))
		}
		target := path
		if action == "reveal" {
			target = containingLocation(path, "windows")
		}
		return openPathForUser(target)
	}
	allowForegroundForInteractiveLaunch(action)
	return startProcessForUser(cmd, action)
}

func allowForegroundForInteractiveLaunch(action string) {
	if action != "code" && action != "terminal" {
		return
	}
	_, _, _ = procAllowSetForegroundWindow.Call(windowsAllowForegroundAny)
}

func openPathForUser(target string) error {
	if processIsElevated() {
		return startProcessAsShellUser("explorer.exe", []string{target}, false)
	}
	return shellExecute(target, "open", "")
}

func startProcessForUser(cmd launchCommand, action string) error {
	if !processIsElevated() {
		process := exec.Command(cmd.Name, cmd.Args...)
		return startBackgroundProcess(process, action)
	}
	hide := action != "terminal" && action != "code"
	return startProcessAsShellUser(cmd.Name, cmd.Args, hide)
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

func processIsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func startProcessAsShellUser(name string, args []string, hide bool) error {
	token, err := shellUserToken()
	if err != nil {
		return err
	}
	defer token.Close()
	if token.IsElevated() {
		return errors.New("shell token is elevated; refusing to launch as the user")
	}

	commandLine, err := windows.UTF16FromString(buildCommandLine(name, args))
	if err != nil {
		return err
	}
	desktop, err := windows.UTF16PtrFromString(`winsta0\default`)
	if err != nil {
		return err
	}

	var startup windows.StartupInfo
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Desktop = desktop
	flags := uint32(createUnicodeEnvironment)
	if hide {
		startup.Flags = windows.STARTF_USESHOWWINDOW
		startup.ShowWindow = windows.SW_HIDE
		flags |= windows.CREATE_NO_WINDOW
	}

	var info windows.ProcessInformation
	ret, _, callErr := procCreateProcessWithToken.Call(
		uintptr(token),
		logonWithProfile,
		0,
		uintptr(unsafe.Pointer(&commandLine[0])),
		uintptr(flags),
		0,
		0,
		uintptr(unsafe.Pointer(&startup)),
		uintptr(unsafe.Pointer(&info)),
	)
	if ret == 0 {
		return fmt.Errorf("launch as user: %w", callErr)
	}
	_ = windows.CloseHandle(info.Thread)
	_ = windows.CloseHandle(info.Process)
	return nil
}

func shellUserToken() (windows.Token, error) {
	pid, err := shellProcessID()
	if err != nil {
		return 0, err
	}
	proc, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return 0, fmt.Errorf("open shell process: %w", err)
	}
	defer windows.CloseHandle(proc)
	var shellToken windows.Token
	if err := windows.OpenProcessToken(proc, windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY|windows.TOKEN_ASSIGN_PRIMARY|windows.TOKEN_ADJUST_DEFAULT|windows.TOKEN_ADJUST_SESSIONID, &shellToken); err != nil {
		return 0, err
	}
	defer shellToken.Close()
	var primary windows.Token
	if err := windows.DuplicateTokenEx(shellToken, windows.TOKEN_ALL_ACCESS, nil, windows.SecurityImpersonation, windows.TokenPrimary, &primary); err != nil {
		return 0, err
	}
	return primary, nil
}

func shellProcessID() (uint32, error) {
	hwnd, _, _ := procGetShellWindow.Call()
	if hwnd == 0 {
		return 0, errNoShellWindow
	}
	var pid uint32
	if _, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid))); pid == 0 {
		return 0, errNoShellWindow
	}
	return pid, nil
}

func buildCommandLine(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, syscall.EscapeArg(name))
	for _, arg := range args {
		parts = append(parts, syscall.EscapeArg(arg))
	}
	return strings.Join(parts, " ")
}

func windowsUsesShellExecute(path string, action string) bool {
	return (action == "open" || action == "reveal" || action == "run-admin" || action == "run-user") && path != "" && !isUnixPath(path) && !isSystemCommandPath(path)
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
