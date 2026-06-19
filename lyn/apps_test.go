package lyn

import (
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScanWindowsApplications(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Example App.lnk"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("unexpected app count %d", len(apps))
	}
	if apps[0].Name != "Example App" || apps[0].Kind != "app" {
		t.Fatalf("unexpected app %#v", apps[0])
	}
}

func TestScanWindowsApplicationsDeduplicatesNames(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	if err := os.WriteFile(filepath.Join(first, "Brave.lnk"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "Brave.lnk"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{first, second}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("unexpected app count %d", len(apps))
	}
	if apps[0].Path != filepath.Join(first, "Brave.lnk") {
		t.Fatalf("unexpected app %#v", apps[0])
	}
}

func TestWindowsShortcutToScriptIsSkipped(t *testing.T) {
	dir := t.TempDir()
	shortcut := filepath.Join(dir, "Console Tool.lnk")
	if err := os.WriteFile(shortcut, fakeWindowsShortcut(`C:\Tools\Console.cmd`), 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("unexpected app count %d", len(apps))
	}
}

func TestWindowsUninstallShortcutIsSkipped(t *testing.T) {
	dir := t.TempDir()
	shortcut := filepath.Join(dir, "Uninstall WhatsApp.lnk")
	if err := os.WriteFile(shortcut, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("unexpected app count %d: %#v", len(apps), apps)
	}
}

func TestWindowsStartupFolderIsSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Example App.lnk"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	startup := filepath.Join(dir, "Startup")
	if err := os.MkdirAll(startup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(startup, "Background Helper.lnk"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Name != "Example App" {
		t.Fatalf("expected only the non-startup app, got %#v", apps)
	}
}

func TestApplicationNameAllowedSkipsJunk(t *testing.T) {
	blocked := []string{"Uninstall WhatsApp", "WhatsApp Uninstaller", "Administrative Tools", ""}
	for _, name := range blocked {
		if applicationNameAllowed(name) {
			t.Fatalf("expected %q to be blocked", name)
		}
	}
	if !applicationNameAllowed("WhatsApp") {
		t.Fatal("expected WhatsApp to be allowed")
	}
}

func TestUnresolvedWindowsShortcutIsKept(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Mystery App.lnk"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Name != "Mystery App" {
		t.Fatalf("unexpected apps %#v", apps)
	}
}

func TestWindowsShortcutToFolderIsSkipped(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "Administrative Tools")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	shortcut := filepath.Join(dir, "Administrative Tools.lnk")
	if err := os.WriteFile(shortcut, fakeWindowsShortcut(folder), 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("unexpected app count %d: %#v", len(apps), apps)
	}
}

func TestAddWindowsStartApplicationsIncludesPackagedApps(t *testing.T) {
	original := windowsStartApps
	windowsStartApps = func(context.Context) ([]windowsStartApp, error) {
		return []windowsStartApp{
			{Name: "WhatsApp", AppID: "5319275A.WhatsAppDesktop_cv1g1gvanyjgm!App"},
			{Name: "Uninstall Noisy App", AppID: "Noisy.Uninstall!App"},
		}, nil
	}
	t.Cleanup(func() { windowsStartApps = original })
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	if err := addWindowsStartApplications(context.Background(), seen, seenNames); err != nil {
		t.Fatal(err)
	}
	apps := seen.sorted()
	if len(apps) != 1 {
		t.Fatalf("unexpected app count %d: %#v", len(apps), apps)
	}
	if apps[0].Name != "WhatsApp" || apps[0].Path != `shell:AppsFolder\5319275A.WhatsAppDesktop_cv1g1gvanyjgm!App` {
		t.Fatalf("unexpected app %#v", apps[0])
	}
}

func TestAddWindowsStartApplicationsDeduplicatesShortcutNames(t *testing.T) {
	original := windowsStartApps
	windowsStartApps = func(context.Context) ([]windowsStartApp, error) {
		return []windowsStartApp{{Name: "Brave", AppID: "Brave.App!App"}}, nil
	}
	t.Cleanup(func() { windowsStartApps = original })
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	addApplication(seen, seenNames, newProject("Brave", `C:\Start Menu\Brave.lnk`, projectKindApp))
	if err := addWindowsStartApplications(context.Background(), seen, seenNames); err != nil {
		t.Fatal(err)
	}
	apps := seen.sorted()
	if len(apps) != 1 || apps[0].Path != `C:\Start Menu\Brave.lnk` {
		t.Fatalf("unexpected apps %#v", apps)
	}
}

func TestParseWindowsPackageManifest(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "5319275A.WhatsAppDesktop_2.2620.102.0_x64__cv1g1gvanyjgm")
	if err := os.Mkdir(packageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10" xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10">
  <Identity Name="5319275A.WhatsAppDesktop" Publisher="CN=test" Version="1.0.0.0" ProcessorArchitecture="x64" />
  <Properties><DisplayName>Fallback</DisplayName></Properties>
  <Applications>
    <Application Id="App" Executable="WhatsApp.Root.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="WhatsApp" Description="WhatsApp" />
    </Application>
    <Application Id="Hidden" Executable="Hidden.exe">
      <uap:VisualElements DisplayName="Hidden" AppListEntry="none" />
    </Application>
  </Applications>
</Package>`
	manifestPath := filepath.Join(packageDir, "AppxManifest.xml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := parseWindowsPackageManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Name != "WhatsApp" || apps[0].AppID != "5319275A.WhatsAppDesktop_cv1g1gvanyjgm!App" {
		t.Fatalf("unexpected apps %#v", apps)
	}
}

func TestQueryWindowsStartAppsUsesManifestRootsWithoutPowerShell(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "Example.App_1.0.0.0_x64__abc123")
	if err := os.Mkdir(packageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10" xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"><Identity Name="Example.App" Publisher="CN=test" Version="1" ProcessorArchitecture="x64" /><Properties><DisplayName>Example</DisplayName></Properties><Applications><Application Id="App" Executable="Example.exe"><uap:VisualElements DisplayName="Example App" /></Application></Applications></Package>`
	if err := os.WriteFile(filepath.Join(packageDir, "AppxManifest.xml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	original := windowsPackagedAppRoots
	windowsPackagedAppRoots = func() []string { return []string{root} }
	t.Cleanup(func() { windowsPackagedAppRoots = original })
	apps, err := queryWindowsStartApps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Name != "Example App" || apps[0].AppID != "Example.App_abc123!App" {
		t.Fatalf("unexpected apps %#v", apps)
	}
}

func TestScanWindowsPathApplicationsOnlyIncludesGuiExecutables(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows executable subsystem test")
	}
	root := t.TempDir()
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gui := filepath.Join(root, "GuiApp.exe")
	console := filepath.Join(root, "ConsoleApp.exe")
	if output, err := exec.Command("go", "build", "-ldflags=-H=windowsgui", "-o", gui, source).CombinedOutput(); err != nil {
		t.Fatalf("build gui: %v\n%s", err, output)
	}
	if output, err := exec.Command("go", "build", "-o", console, source).CombinedOutput(); err != nil {
		t.Fatalf("build console: %v\n%s", err, output)
	}
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	if err := addWindowsPathApplications(context.Background(), seen, seenNames, []string{root}); err != nil {
		t.Fatal(err)
	}
	apps := seen.sorted()
	if len(apps) != 1 {
		t.Fatalf("unexpected app count %d: %#v", len(apps), apps)
	}
	if apps[0].Name != "GuiApp" || apps[0].Path != gui {
		t.Fatalf("unexpected app %#v", apps[0])
	}
}

func fakeWindowsShortcut(target string) []byte {
	header := make([]byte, 0x4c)
	binary.LittleEndian.PutUint32(header[0:4], 0x4c)
	binary.LittleEndian.PutUint32(header[0x14:0x18], 0x2)
	encoded := append([]byte(target), 0)
	size := 0x1c + len(encoded)
	linkInfo := make([]byte, size)
	binary.LittleEndian.PutUint32(linkInfo[0:4], uint32(size))
	binary.LittleEndian.PutUint32(linkInfo[4:8], 0x1c)
	binary.LittleEndian.PutUint32(linkInfo[8:12], 0x1)
	binary.LittleEndian.PutUint32(linkInfo[0x10:0x14], 0x1c)
	copy(linkInfo[0x1c:], encoded)
	return append(header, linkInfo...)
}

func TestScanLinuxApplications(t *testing.T) {
	dir := t.TempDir()
	data := []byte("[Desktop Entry]\nName=Example App\nExec=example\n")
	if err := os.WriteFile(filepath.Join(dir, "example.desktop"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("unexpected app count %d", len(apps))
	}
	if apps[0].Name != "Example App" || apps[0].Kind != "app" {
		t.Fatalf("unexpected app %#v", apps[0])
	}
}

func TestHiddenLinuxApplicationIsSkipped(t *testing.T) {
	dir := t.TempDir()
	data := []byte("[Desktop Entry]\nName=Hidden App\nNoDisplay=true\nExec=hidden\n")
	if err := os.WriteFile(filepath.Join(dir, "hidden.desktop"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("unexpected app count %d", len(apps))
	}
}

func TestLinuxApplicationRespectsShowInEnvironment(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("only-lxqt.desktop", "[Desktop Entry]\nName=Lxqt Only\nOnlyShowIn=LXQt;\nExec=lxqt-leave --logout\n")
	write("not-xmonad.desktop", "[Desktop Entry]\nName=Not Xmonad\nNotShowIn=Xmonad;\nExec=foo\n")
	write("plain.desktop", "[Desktop Entry]\nName=Plain App\nExec=plain\n")

	t.Setenv("XDG_CURRENT_DESKTOP", "Xmonad")
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Name != "Plain App" {
		t.Fatalf("expected only the plain app under Xmonad, got %#v", apps)
	}

	t.Setenv("XDG_CURRENT_DESKTOP", "LXQt")
	apps, err = scanApplicationDirs(context.Background(), []string{dir}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 3 {
		t.Fatalf("expected all apps under LXQt, got %d", len(apps))
	}
}
