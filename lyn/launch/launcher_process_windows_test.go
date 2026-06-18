//go:build windows

package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestConfigureLaunchProcessHidesNonTerminalWindows(t *testing.T) {
	process := exec.Command("example.exe")
	configureLaunchProcess(process, "open")
	if process.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr")
	}
	if !process.SysProcAttr.HideWindow {
		t.Fatal("expected hidden process window")
	}
	if process.SysProcAttr.CreationFlags&windowsDetachedProcess == 0 {
		t.Fatal("expected detached-process creation flag")
	}
}

func TestConfigureLaunchProcessKeepsInteractiveWindowsVisible(t *testing.T) {
	for _, action := range []string{"terminal", "code"} {
		process := exec.Command("example.exe")
		configureLaunchProcess(process, action)
		if process.SysProcAttr != nil {
			t.Fatalf("expected %s launch to keep default process attrs, got %#v", action, process.SysProcAttr)
		}
	}
}

func TestAllowForegroundForInteractiveLaunch(t *testing.T) {
	for _, action := range []string{"code", "terminal"} {
		allowForegroundForInteractiveLaunch(action)
	}
	for _, action := range []string{"open", "reveal", "run-admin", "run-user", ""} {
		allowForegroundForInteractiveLaunch(action)
	}
}

func TestBuildCommandLineQuotesArguments(t *testing.T) {
	got := buildCommandLine("explorer.exe", []string{`C:\Program Files\App\x`})
	want := `explorer.exe "C:\Program Files\App\x"`
	if got != want {
		t.Fatalf("buildCommandLine = %q, want %q", got, want)
	}
}

func TestWindowsUsesShellExecuteForLocalOpenRevealAndRunAs(t *testing.T) {
	for _, action := range []string{"open", "reveal", "run-admin", "run-user"} {
		if !windowsUsesShellExecute(`C:\Users\me\Desktop\App.lnk`, action) {
			t.Fatalf("expected shell execute for %s", action)
		}
	}
}

func TestWindowsDoesNotUseShellExecuteForCodeTerminalOrWsl(t *testing.T) {
	cases := []struct {
		path   string
		action string
	}{
		{`C:\Users\me\src\app`, "code"},
		{`C:\Users\me\src\app`, "terminal"},
		{"/home/me/src/app", "code"},
		{"/home/me/src/app", "open"},
		{"", "open"},
	}
	for _, tc := range cases {
		if windowsUsesShellExecute(tc.path, tc.action) {
			t.Fatalf("unexpected shell execute for %#v", tc)
		}
	}
}

func TestWindowsCodeCommandPrefersExeBesideCodeCmd(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(bin, "code.cmd")
	exePath := filepath.Join(root, "Code.exe")
	if err := os.WriteFile(cmdPath, []byte("@echo off\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exePath, []byte{}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got := windowsCodeCommandName(); got != exePath {
		t.Fatalf("expected %q, got %q", exePath, got)
	}
}
