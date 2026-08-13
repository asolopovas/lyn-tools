package lyn

import (
	"context"
	"debug/pe"
	"encoding/binary"
	"encoding/xml"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf16"
)

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
	Square44Logo string `xml:"Square44x44Logo,attr"`
	Square30Logo string `xml:"Square30x30Logo,attr"`
	Logo         string `xml:"Logo,attr"`
}

const windowsAppsFolderPrefix = `shell:AppsFolder\`

var windowsStartApps = queryWindowsStartApps
var windowsPackagedAppRoots = defaultWindowsPackagedAppRoots

func windowsApplicationDirs() []string {
	return []string{
		filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("AppData"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("Public"), "Desktop"),
		filepath.Join(os.Getenv("UserProfile"), "Desktop"),
	}
}

var windowsSystemRoot = defaultWindowsSystemRoot

func defaultWindowsSystemRoot() string {
	if root := strings.TrimSpace(os.Getenv("SystemRoot")); root != "" {
		return root
	}
	return `C:\Windows`
}

type windowsSystemTool struct {
	name string
	uri  string
	rel  []string
}

func windowsSystemTools() []Project {
	root := windowsSystemRoot()
	defs := []windowsSystemTool{
		{name: "Control Panel", rel: []string{"System32", "control.exe"}},
		{name: "Task Manager", rel: []string{"System32", "Taskmgr.exe"}},
		{name: "Device Manager", rel: []string{"System32", "devmgmt.msc"}},
		{name: "Services", rel: []string{"System32", "services.msc"}},
		{name: "Event Viewer", rel: []string{"System32", "eventvwr.msc"}},
		{name: "Disk Management", rel: []string{"System32", "diskmgmt.msc"}},
		{name: "Computer Management", rel: []string{"System32", "compmgmt.msc"}},
		{name: "Registry Editor", rel: []string{"regedit.exe"}},
		{name: "Task Scheduler", rel: []string{"System32", "taskschd.msc"}},
		{name: "Performance Monitor", rel: []string{"System32", "perfmon.exe"}},
		{name: "System Information", rel: []string{"System32", "msinfo32.exe"}},
		{name: "System Configuration", rel: []string{"System32", "msconfig.exe"}},
		{name: "Settings", uri: "ms-settings:"},
	}
	tools := make([]Project, 0, len(defs))
	for _, def := range defs {
		if def.uri != "" {
			tools = append(tools, newProject(def.name, def.uri, projectKindApp))
			continue
		}
		path := filepath.Join(append([]string{root}, def.rel...)...)
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			continue
		}
		tools = append(tools, newProject(def.name, path, projectKindApp))
	}
	return tools
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
		addApplication(seen, seenNames, newProject(name, windowsAppsFolderPrefix+appID, projectKindApp))
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

func windowsPackagedAppLogo(appID string) (string, bool) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "", false
	}
	for _, root := range windowsPackagedAppRoots() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			packageDir := filepath.Join(root, entry.Name())
			logo, ok := windowsManifestAppLogo(filepath.Join(packageDir, "AppxManifest.xml"), appID)
			if !ok {
				continue
			}
			if asset, ok := resolveWindowsAssetFile(packageDir, logo); ok {
				return asset, true
			}
		}
	}
	return "", false
}

func windowsManifestAppLogo(manifestPath string, appID string) (string, bool) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", false
	}
	var manifest appxPackageManifest
	if err := xml.Unmarshal(data, &manifest); err != nil {
		return "", false
	}
	family := windowsPackageFamilyName(filepath.Base(filepath.Dir(manifestPath)), manifest.Identity.Name)
	if family == "" {
		return "", false
	}
	for _, application := range manifest.Applications {
		id := strings.TrimSpace(application.ID)
		if id == "" || family+"!"+id != appID {
			continue
		}
		logo := firstNonEmpty(
			application.VisualElements.Square44Logo,
			application.VisualElements.Square30Logo,
			application.VisualElements.Logo,
		)
		return strings.TrimSpace(logo), strings.TrimSpace(logo) != ""
	}
	return "", false
}

func resolveWindowsAssetFile(packageDir string, logo string) (string, bool) {
	rel := filepath.FromSlash(strings.ReplaceAll(logo, `\`, "/"))
	full := filepath.Join(packageDir, rel)
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		return full, true
	}
	return bestScaledWindowsAsset(full)
}

func bestScaledWindowsAsset(full string) (string, bool) {
	dir := filepath.Dir(full)
	ext := filepath.Ext(full)
	prefix := strings.TrimSuffix(filepath.Base(full), ext) + "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	best := ""
	bestScore := -1
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.EqualFold(filepath.Ext(name), ext) {
			continue
		}
		qualifier := strings.ToLower(name[len(prefix) : len(name)-len(ext)])
		if score := scaledAssetScore(qualifier); score > bestScore {
			bestScore = score
			best = filepath.Join(dir, name)
		}
	}
	return best, best != ""
}

func scaledAssetScore(qualifier string) int {
	if strings.Contains(qualifier, "contrast-") {
		return 1
	}
	score := 10
	switch {
	case strings.Contains(qualifier, "scale-200"):
		score = 100
	case strings.Contains(qualifier, "scale-150"):
		score = 90
	case strings.Contains(qualifier, "scale-125"):
		score = 85
	case strings.Contains(qualifier, "scale-100"):
		score = 80
	case strings.Contains(qualifier, "targetsize-"):
		score = 70
	}
	if strings.Contains(qualifier, "altform-unplated") {
		score += 5
	}
	return score
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
	return isWindowsLaunchableShortcutTarget(target)
}

func isWindowsLaunchableShortcutTarget(target string) bool {
	switch strings.ToLower(filepath.Ext(target)) {
	case ".cmd", ".bat", ".ps1", ".psm1", ".vbs", ".js", ".jse", ".wsf", ".wsh":
		return false
	case ".htm", ".html", ".mht", ".mhtml", ".url", ".chm", ".hlp", ".txt", ".rtf", ".pdf", ".md":
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

func isWindowsWebInternetShortcut(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(strings.ToLower(line), "url=") {
			continue
		}
		target := strings.ToLower(strings.TrimSpace(line[len("url="):]))
		return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")
	}
	return false
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
	return strings.TrimSuffix(name, filepath.Ext(name))
}
