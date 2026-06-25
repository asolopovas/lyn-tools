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
	blocked := []string{"Uninstall WhatsApp", "WhatsApp Uninstaller", "Install Example", "Example Setup", ""}
	for _, name := range blocked {
		if applicationNameAllowed(name) {
			t.Fatalf("expected %q to be blocked", name)
		}
	}
	for _, name := range []string{"WhatsApp", "Control Panel", "Administrative Tools"} {
		if !applicationNameAllowed(name) {
			t.Fatalf("expected %q to be allowed", name)
		}
	}
}

func TestWindowsSystemToolsResolveExistingPaths(t *testing.T) {
	root := t.TempDir()
	system32 := filepath.Join(root, "System32")
	if err := os.MkdirAll(system32, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"control.exe", "services.msc"} {
		if err := os.WriteFile(filepath.Join(system32, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "regedit.exe"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := windowsSystemRoot
	windowsSystemRoot = func() string { return root }
	t.Cleanup(func() { windowsSystemRoot = original })

	byName := map[string]string{}
	for _, tool := range windowsSystemTools() {
		if tool.Kind != projectKindApp {
			t.Fatalf("expected app kind, got %#v", tool)
		}
		byName[tool.Name] = tool.Path
	}
	if byName["Control Panel"] != filepath.Join(system32, "control.exe") {
		t.Fatalf("unexpected Control Panel path %q", byName["Control Panel"])
	}
	if byName["Registry Editor"] != filepath.Join(root, "regedit.exe") {
		t.Fatalf("unexpected Registry Editor path %q", byName["Registry Editor"])
	}
	if byName["Settings"] != "ms-settings:" {
		t.Fatalf("expected Settings to use a URI, got %q", byName["Settings"])
	}
	if _, ok := byName["Task Manager"]; ok {
		t.Fatal("expected missing Taskmgr.exe to be skipped")
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

func TestWindowsShortcutToDocumentIsSkipped(t *testing.T) {
	dir := t.TempDir()
	for name, target := range map[string]string{
		"CMake Documentation.lnk": `C:\Program Files\CMake\doc\index.html`,
		"PuTTY Manual.lnk":        `C:\Program Files\PuTTY\putty.chm`,
		"TransMac Read Me.lnk":    `C:\Program Files\TransMac\Readme.txt`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), fakeWindowsShortcut(target), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("unexpected app count %d: %#v", len(apps), apps)
	}
}

func TestWindowsWebInternetShortcutIsSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Mp3tag Website.url"), []byte("[InternetShortcut]\nURL=https://www.mp3tag.de/en/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Counter-Strike 2.url"), []byte("[InternetShortcut]\nURL=steam://rungameid/730\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := scanApplicationDirs(context.Background(), []string{dir}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Name != "Counter-Strike 2" {
		t.Fatalf("expected only the steam launcher, got %#v", apps)
	}
}

func TestTrimApplicationNameKeepsDottedNames(t *testing.T) {
	cases := map[string]string{
		"Brave.lnk":                        "Brave",
		"Node.js documentation.url":        "Node.js documentation",
		"IDLE (Python 3.14 64-bit).lnk":    "IDLE (Python 3.14 64-bit)",
		"AOMEI Partition Assistant 10.lnk": "AOMEI Partition Assistant 10",
		"code.exe":                         "code",
	}
	for name, want := range cases {
		if got := trimApplicationName(name); got != want {
			t.Fatalf("trimApplicationName(%q) = %q, want %q", name, got, want)
		}
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

func TestWindowsPackagedAppLogoResolvesScaledAsset(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "Example.App_1.0.0.0_x64__abc123")
	assets := filepath.Join(packageDir, "Assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10" xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"><Identity Name="Example.App" Publisher="CN=test" Version="1" ProcessorArchitecture="x64" /><Properties><DisplayName>Example</DisplayName></Properties><Applications><Application Id="App" Executable="Example.exe"><uap:VisualElements DisplayName="Example App" Square44x44Logo="Assets\Square44x44Logo.png" /></Application></Applications></Package>`
	if err := os.WriteFile(filepath.Join(packageDir, "AppxManifest.xml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Square44x44Logo.scale-100.png", "Square44x44Logo.scale-200.png", "Square44x44Logo.scale-200_contrast-black.png"} {
		if err := os.WriteFile(filepath.Join(assets, name), []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	original := windowsPackagedAppRoots
	windowsPackagedAppRoots = func() []string { return []string{root} }
	t.Cleanup(func() { windowsPackagedAppRoots = original })
	asset, ok := windowsPackagedAppLogo("Example.App_abc123!App")
	if !ok {
		t.Fatal("expected a logo asset")
	}
	if filepath.Base(asset) != "Square44x44Logo.scale-200.png" {
		t.Fatalf("expected scale-200 asset, got %q", asset)
	}
}

func TestWindowsPackagedAppLogoUsesExactAssetWhenPresent(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "Example.App_1.0.0.0_x64__abc123")
	assets := filepath.Join(packageDir, "Assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10" xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"><Identity Name="Example.App" Publisher="CN=test" Version="1" ProcessorArchitecture="x64" /><Properties><DisplayName>Example</DisplayName></Properties><Applications><Application Id="App" Executable="Example.exe"><uap:VisualElements DisplayName="Example App" Square44x44Logo="Assets\Square44x44Logo.png" /></Application></Applications></Package>`
	if err := os.WriteFile(filepath.Join(packageDir, "AppxManifest.xml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "Square44x44Logo.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := windowsPackagedAppRoots
	windowsPackagedAppRoots = func() []string { return []string{root} }
	t.Cleanup(func() { windowsPackagedAppRoots = original })
	asset, ok := windowsPackagedAppLogo("Example.App_abc123!App")
	if !ok || filepath.Base(asset) != "Square44x44Logo.png" {
		t.Fatalf("expected exact logo asset, got %q (ok=%v)", asset, ok)
	}
}

func TestWindowsPackagedAppLogoMissesUnknownApp(t *testing.T) {
	original := windowsPackagedAppRoots
	windowsPackagedAppRoots = func() []string { return []string{t.TempDir()} }
	t.Cleanup(func() { windowsPackagedAppRoots = original })
	if asset, ok := windowsPackagedAppLogo("Missing.App_abc123!App"); ok {
		t.Fatalf("expected no logo, got %q", asset)
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
