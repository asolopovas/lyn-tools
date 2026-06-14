package lyn

import (
	"context"
	"debug/pe"
	"encoding/binary"
	"encoding/xml"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode/utf16"
)

func ScanApplications(ctx context.Context) ([]Project, error) {
	return scanApplications(ctx, applicationDirs(runtime.GOOS), runtime.GOOS)
}

func scanApplications(ctx context.Context, dirs []string, goos string) ([]Project, error) {
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	if err := addApplicationsFromDirs(ctx, seen, seenNames, dirs, goos); err != nil {
		return seen.sorted(), err
	}
	if goos == "windows" {
		if err := addWindowsStartApplications(ctx, seen, seenNames); err != nil {
			return seen.sorted(), err
		}
		if err := addWindowsPathApplications(ctx, seen, seenNames, windowsPathDirs()); err != nil {
			return seen.sorted(), err
		}
	}
	return seen.sorted(), nil
}

func scanApplicationDirs(ctx context.Context, dirs []string, goos string) ([]Project, error) {
	seen := newProjectSet(0)
	seenNames := newStringSet(0)
	if err := addApplicationsFromDirs(ctx, seen, seenNames, dirs, goos); err != nil {
		return seen.sorted(), err
	}
	return seen.sorted(), nil
}

type windowsStartApp struct {
	Name  string
	AppID string
}

type appxPackageManifest struct {
	Identity     appxIdentity      `xml:"Identity"`
	Properties   appxProperties    `xml:"Properties"`
	Applications []appxApplication `xml:"Applications>Application"`
}

type appxIdentity struct {
	Name string `xml:"Name,attr"`
}

type appxProperties struct {
	DisplayName string `xml:"DisplayName"`
}

type appxApplication struct {
	ID             string             `xml:"Id,attr"`
	Executable     string             `xml:"Executable,attr"`
	EntryPoint     string             `xml:"EntryPoint,attr"`
	VisualElements appxVisualElements `xml:"VisualElements"`
}

type appxVisualElements struct {
	DisplayName  string `xml:"DisplayName,attr"`
	AppListEntry string `xml:"AppListEntry,attr"`
}

var windowsStartApps = queryWindowsStartApps
var windowsPackagedAppRoots = defaultWindowsPackagedAppRoots

func addApplicationsFromDirs(ctx context.Context, seen projectSet, seenNames stringSet, dirs []string, goos string) error {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || ctx.Err() != nil {
				return ctx.Err()
			}
			if entry.IsDir() {
				if isWindowsStartupDir(entry.Name(), goos) {
					return filepath.SkipDir
				}
				return nil
			}
			if app, ok := detectApplication(path, goos); ok {
				addApplication(seen, seenNames, app)
			}
			return nil
		})
		if err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

func applicationDirs(goos string) []string {
	switch goos {
	case "windows":
		return windowsApplicationDirs()
	case "linux":
		return linuxApplicationDirs()
	default:
		return nil
	}
}

func windowsApplicationDirs() []string {
	return []string{
		filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("AppData"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("Public"), "Desktop"),
		filepath.Join(os.Getenv("UserProfile"), "Desktop"),
	}
}

func linuxApplicationDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/usr/share/applications",
		"/usr/local/share/applications",
		filepath.Join(home, ".local", "share", "applications"),
	}
}

func detectApplication(path string, goos string) (Project, bool) {
	switch goos {
	case "windows":
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".lnk" && ext != ".appref-ms" && ext != ".url" {
			return Project{}, false
		}
		if ext == ".lnk" && !isWindowsShortcutGUIApplication(path) {
			return Project{}, false
		}
		return newProject(trimApplicationName(filepath.Base(path)), path, projectKindApp), true
	case "linux":
		if strings.ToLower(filepath.Ext(path)) != ".desktop" {
			return Project{}, false
		}
		name, ok := desktopApplicationName(path)
		if !ok {
			return Project{}, false
		}
		return newProject(name, path, projectKindApp), true
	default:
		return Project{}, false
	}
}

func addWindowsStartApplications(ctx context.Context, seen projectSet, seenNames stringSet) error {
	apps, err := windowsStartApps(ctx)
	if err != nil {
		return ctx.Err()
	}
	for _, app := range apps {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := strings.TrimSpace(app.Name)
		appID := strings.TrimSpace(app.AppID)
		if !windowsStartAppAllowed(name, appID) {
			continue
		}
		addApplication(seen, seenNames, newProject(name, `shell:AppsFolder\`+appID, projectKindApp))
	}
	return nil
}

func queryWindowsStartApps(ctx context.Context) ([]windowsStartApp, error) {
	var apps []windowsStartApp
	seen := newStringSet(0)
	for _, root := range windowsPackagedAppRoots() {
		if ctx.Err() != nil {
			return apps, ctx.Err()
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if ctx.Err() != nil {
				return apps, ctx.Err()
			}
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(root, entry.Name(), "AppxManifest.xml")
			manifestApps, err := parseWindowsPackageManifest(manifestPath)
			if err != nil {
				continue
			}
			for _, app := range manifestApps {
				if !seen.addFold(app.AppID) {
					continue
				}
				apps = append(apps, app)
			}
		}
	}
	return apps, nil
}

func defaultWindowsPackagedAppRoots() []string {
	return []string{filepath.Join(os.Getenv("ProgramFiles"), "WindowsApps")}
}

func parseWindowsPackageManifest(path string) ([]windowsStartApp, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest appxPackageManifest
	if err := xml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	family := windowsPackageFamilyName(filepath.Base(filepath.Dir(path)), manifest.Identity.Name)
	if family == "" {
		return nil, nil
	}
	apps := make([]windowsStartApp, 0, len(manifest.Applications))
	for _, application := range manifest.Applications {
		if !windowsPackageApplicationAllowed(application) {
			continue
		}
		name := windowsPackageApplicationName(application, manifest.Properties.DisplayName)
		if !applicationNameAllowed(name) || strings.HasPrefix(strings.ToLower(name), "ms-resource:") {
			continue
		}
		apps = append(apps, windowsStartApp{Name: name, AppID: family + "!" + strings.TrimSpace(application.ID)})
	}
	return apps, nil
}

func windowsPackageApplicationAllowed(application appxApplication) bool {
	if strings.TrimSpace(application.ID) == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(application.VisualElements.AppListEntry), "none") {
		return false
	}
	return strings.TrimSpace(application.Executable) != "" || strings.TrimSpace(application.EntryPoint) != ""
}

func windowsPackageApplicationName(application appxApplication, fallback string) string {
	name := strings.TrimSpace(application.VisualElements.DisplayName)
	if name != "" {
		return name
	}
	return strings.TrimSpace(fallback)
}

func windowsPackageFamilyName(packageDir string, identityName string) string {
	identityName = strings.TrimSpace(identityName)
	if identityName == "" {
		return ""
	}
	marker := "__"
	index := strings.LastIndex(packageDir, marker)
	if index < 0 || index+len(marker) >= len(packageDir) {
		return ""
	}
	publisherID := strings.TrimSpace(packageDir[index+len(marker):])
	if publisherID == "" {
		return ""
	}
	return identityName + "_" + publisherID
}

func windowsStartAppAllowed(name string, appID string) bool {
	return strings.TrimSpace(appID) != "" && applicationNameAllowed(name)
}

func addWindowsPathApplications(ctx context.Context, seen projectSet, seenNames stringSet, dirs []string) error {
	seenDirs := newStringSet(0)
	for _, dir := range dirs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		clean := strings.TrimSpace(dir)
		if clean == "" || withinWindowsSystemDir(clean) {
			continue
		}
		full, err := filepath.Abs(clean)
		if err != nil {
			continue
		}
		if !seenDirs.addFold(full) {
			continue
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".exe") {
				continue
			}
			path := filepath.Join(full, entry.Name())
			if !isWindowsGUIExecutable(path) {
				continue
			}
			app := newProject(trimApplicationName(entry.Name()), path, projectKindApp)
			addApplication(seen, seenNames, app)
		}
	}
	return nil
}

func addApplication(seen projectSet, seenNames stringSet, app Project) bool {
	if !applicationNameAllowed(app.Name) {
		return false
	}
	if !seenNames.addFold(app.Name) {
		return false
	}
	seen.add(app)
	return true
}

func applicationNameAllowed(name string) bool {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if lowerName == "" {
		return false
	}
	if strings.Contains(lowerName, "uninstall") || strings.Contains(lowerName, "uninstaller") {
		return false
	}
	return lowerName != "administrative tools"
}

func windowsPathDirs() []string {
	dirs := filepath.SplitList(os.Getenv("PATH"))
	for i := range dirs {
		dirs[i] = strings.TrimSpace(dirs[i])
	}
	return slices.DeleteFunc(dirs, func(dir string) bool {
		return dir == ""
	})
}

func isWindowsGUIExecutable(path string) bool {
	file, err := pe.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	switch header := file.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		return header.Subsystem == pe.IMAGE_SUBSYSTEM_WINDOWS_GUI
	case *pe.OptionalHeader64:
		return header.Subsystem == pe.IMAGE_SUBSYSTEM_WINDOWS_GUI
	default:
		return false
	}
}

func isWindowsShortcutGUIApplication(path string) bool {
	target, ok := windowsShortcutTarget(path)
	if !ok {
		return true
	}
	ext := strings.ToLower(filepath.Ext(target))
	switch ext {
	case ".cmd", ".bat", ".ps1", ".psm1", ".vbs", ".js", ".jse", ".wsf", ".wsh":
		return false
	case ".exe":
		return isWindowsGUIExecutable(target)
	default:
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return false
		}
		return true
	}
}

func windowsShortcutTarget(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 0x4c {
		return "", false
	}
	if binary.LittleEndian.Uint32(data[0:4]) != 0x4c {
		return "", false
	}
	flags := binary.LittleEndian.Uint32(data[0x14:0x18])
	offset := 0x4c
	if flags&0x1 != 0 {
		if len(data) < offset+2 {
			return "", false
		}
		offset += 2 + int(binary.LittleEndian.Uint16(data[offset:offset+2]))
	}
	if flags&0x2 == 0 || len(data) < offset+0x1c {
		return "", false
	}
	size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	headerSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
	if size <= 0 || len(data) < offset+size || headerSize < 0x1c {
		return "", false
	}
	localOffset := int(binary.LittleEndian.Uint32(data[offset+0x10 : offset+0x14]))
	suffixOffset := int(binary.LittleEndian.Uint32(data[offset+0x18 : offset+0x1c]))
	local := readShortcutString(data, offset, size, localOffset, false)
	suffix := readShortcutString(data, offset, size, suffixOffset, false)
	if headerSize >= 0x24 {
		localUnicodeOffset := int(binary.LittleEndian.Uint32(data[offset+0x1c : offset+0x20]))
		suffixUnicodeOffset := int(binary.LittleEndian.Uint32(data[offset+0x20 : offset+0x24]))
		if value := readShortcutString(data, offset, size, localUnicodeOffset, true); value != "" {
			local = value
		}
		if value := readShortcutString(data, offset, size, suffixUnicodeOffset, true); value != "" {
			suffix = value
		}
	}
	if strings.TrimSpace(local) != "" {
		return strings.TrimSpace(local), true
	}
	if strings.TrimSpace(suffix) != "" {
		return strings.TrimSpace(suffix), true
	}
	return "", false
}

func readShortcutString(data []byte, base int, size int, offset int, unicode bool) string {
	if offset <= 0 || offset >= size {
		return ""
	}
	start := base + offset
	end := base + size
	if start < 0 || start >= len(data) || end > len(data) || start >= end {
		return ""
	}
	if unicode {
		values := make([]uint16, 0, (end-start)/2)
		for i := start; i+1 < end; i += 2 {
			value := binary.LittleEndian.Uint16(data[i : i+2])
			if value == 0 {
				break
			}
			values = append(values, value)
		}
		return string(utf16.Decode(values))
	}
	for i := start; i < end; i++ {
		if data[i] == 0 {
			return string(data[start:i])
		}
	}
	return string(data[start:end])
}

func trimApplicationName(name string) string {
	for {
		ext := filepath.Ext(name)
		if ext == "" {
			return name
		}
		name = strings.TrimSuffix(name, ext)
	}
}

func desktopApplicationName(path string) (string, bool) {
	entry := readDesktopEntry(path)
	if strings.EqualFold(entry["NoDisplay"], "true") || strings.EqualFold(entry["Hidden"], "true") {
		return "", false
	}
	name := strings.TrimSpace(entry["Name"])
	return name, name != ""
}

func readDesktopEntry(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	entry := map[string]string{}
	inSection := false
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "[Desktop Entry]" {
			inSection = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = false
			continue
		}
		if !inSection || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		entry[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return entry
}
