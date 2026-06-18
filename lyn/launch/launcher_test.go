package launch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildLaunchCommandWindowsOpen(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\Users\me\src\app`, Action: "open"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "explorer.exe" || cmd.Args[0] != `C:\Users\me\src\app` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsCodeUsesPath(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\Users\me\src\app`, Action: "code"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 1 || cmd.Args[0] != `C:\Users\me\src\app` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsWslCodeUsesFolderURI(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/home/me/src/app", Action: "code"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "--folder-uri" || cmd.Args[1] != "vscode-remote://wsl+default/home/me/src/app" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsWslCodeUsesNamedDistro(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/home/me/src/app", Action: "code", Distro: "Ubuntu"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "--folder-uri" || cmd.Args[1] != "vscode-remote://wsl+Ubuntu/home/me/src/app" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsWslTerminalUsesNamedDistro(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/home/me/src/app", Action: "terminal", Distro: "Ubuntu"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "wsl.exe" || len(cmd.Args) != 4 || cmd.Args[0] != "-d" || cmd.Args[1] != "Ubuntu" || cmd.Args[2] != "--cd" || cmd.Args[3] != "/home/me/src/app" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandVSCodeRemoteSSHFolderUsesFolderURI(t *testing.T) {
	path := "vscode-remote://ssh-remote+examplehost/srv/www/example"
	cmd, err := BuildLaunchCommand(Request{Path: path, Action: "code"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "--folder-uri" || cmd.Args[1] != path {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandVSCodeRemoteSSHWorkspaceUsesFileURI(t *testing.T) {
	path := "vscode-remote://ssh-remote+examplehost/var/www/example.com/public_html/wp-content/themes/example_theme/example.code-workspace"
	cmd, err := BuildLaunchCommand(Request{Path: path, Action: "code"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "--file-uri" || cmd.Args[1] != path {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandVSCodeRemoteNonPathURI(t *testing.T) {
	path := "vscode-remote://dev-container+workspace/srv/www/example.code-workspace"
	cmd, err := BuildLaunchCommand(Request{Path: path, Action: "code"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "--file-uri" || cmd.Args[1] != path {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsWslTerminal(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/home/me/src/app", Action: "terminal"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "wsl.exe" || cmd.Args[0] != "--cd" || cmd.Args[1] != "/home/me/src/app" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandRejectsUnsupportedAction(t *testing.T) {
	_, err := BuildLaunchCommand(Request{Path: "/tmp/app", Action: "missing"}, "linux")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildLaunchCommandRejectsSystemCommands(t *testing.T) {
	for _, path := range []string{"lyn:system:restart", "lyn:system:shutdown", "lyn:system:logout"} {
		t.Run(path, func(t *testing.T) {
			_, err := BuildLaunchCommand(Request{Path: path, Action: "open"}, "windows")
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBuildLaunchCommandWindowsRunAsAdministrator(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\Program Files\App\App.exe`, Action: "run-admin"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "ShellExecuteW" || cmd.Args[0] != "runAs" || cmd.Args[1] != `C:\Program Files\App\App.exe` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsRunAsDifferentUser(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\Program Files\App\App.exe`, Action: "run-user"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "ShellExecuteW" || cmd.Args[0] != "runAsUser" || cmd.Args[1] != `C:\Program Files\App\App.exe` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandRejectsRunAsOffWindows(t *testing.T) {
	_, err := BuildLaunchCommand(Request{Path: "/tmp/app", Action: "run-admin"}, "linux")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildLaunchCommandWindowsAppShortcut(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\Users\me\Desktop\App.lnk`, Action: "open"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "rundll32.exe" || cmd.Args[0] != "url.dll,FileProtocolHandler" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsPackagedAppMapsUnsupportedActionsToOpen(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `shell:AppsFolder\WhatsApp!App`, Action: "reveal"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "explorer.exe" || cmd.Args[0] != `shell:AppsFolder\WhatsApp!App` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsRevealShortcut(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\ProgramData\Microsoft\Windows\Start Menu\Programs\App.lnk`, Action: "reveal"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "explorer.exe" || cmd.Args[0] != `C:\ProgramData\Microsoft\Windows\Start Menu\Programs` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandWindowsRevealFolder(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: `C:\Users\me\src\app`, Action: "reveal"}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "explorer.exe" || cmd.Args[0] != `C:\Users\me\src` {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandLinuxRevealFolderLocation(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/home/me/src/app", Action: "reveal"}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "xdg-open" || cmd.Args[0] != "/home/me/src" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandTerminalUsesContainingLocationForFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "client.code-workspace")
	if err := os.WriteFile(path, []byte(`{"folders":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, err := BuildLaunchCommand(Request{Path: path, Action: "terminal"}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "x-terminal-emulator" || cmd.Args[0] != "--working-directory" || cmd.Args[1] != filepath.ToSlash(root) {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandLinuxRevealDesktopFile(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/usr/share/applications/example.desktop", Action: "reveal"}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "xdg-open" || cmd.Args[0] != "/usr/share/applications" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}

func TestBuildLaunchCommandLinuxDesktopApp(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "/usr/share/applications/example.desktop", Action: "open"}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "gtk-launch" || cmd.Args[0] != "example" {
		t.Fatalf("unexpected command %#v", cmd)
	}
}
