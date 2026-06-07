package lyn

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strings"
)

func ScanVSCodeRecentProjects(ctx context.Context) ([]Project, error) {
	return scanVSCodeRecentSourceSets(ctx, vscodeRecentSourceSets(runtime.GOOS), runtime.GOOS)
}

func scanVSCodeRecentProjects(ctx context.Context, roots []string, goos string) ([]Project, error) {
	sets := make([]vscodeRecentSourceSet, 0, len(roots))
	for _, root := range roots {
		sets = append(sets, vscodeRecentSourceSet{Fallback: vscodeRecentSources(root)})
	}
	return scanVSCodeRecentSourceSets(ctx, sets, goos)
}

func scanVSCodeRecentSourceSets(ctx context.Context, sets []vscodeRecentSourceSet, goos string) ([]Project, error) {
	seen := newProjectSet(0)
	var firstErr error
	for _, set := range sets {
		if set.Primary != "" {
			items, err := vscodeRecentFromState(ctx, filepath.Join(set.Primary, "state.vscdb"), goos)
			if err == nil {
				seen.addAll(items)
				continue
			}
			if shouldReportVSCodeRecentError(err) {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}
		for _, source := range set.Fallback {
			items, err := vscodeRecentFromSource(ctx, source, goos)
			if shouldReportVSCodeRecentError(err) && firstErr == nil {
				firstErr = err
			}
			seen.addAll(items)
		}
	}
	return seen.sorted(), firstErr
}

type vscodeRecentSourceSet struct {
	Primary  string
	Fallback []string
}

func shouldReportVSCodeRecentError(err error) bool {
	return err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, sql.ErrNoRows)
}

func vscodeRecentFromSource(ctx context.Context, source string, goos string) ([]Project, error) {
	return vscodeRecentFromState(ctx, filepath.Join(source, "state.vscdb"), goos)
}

var vscodeStorageProducts = []struct {
	Storage string
	Shared  string
}{
	{Storage: "Code", Shared: ".vscode-shared"},
	{Storage: "Code - Insiders", Shared: ".vscode-insiders-shared"},
	{Storage: "Code - Exploration", Shared: ".vscode-exploration-shared"},
	{Storage: "VSCodium", Shared: ".vscodium-shared"},
	{Storage: "VSCodium - Insiders", Shared: ".vscodium-insiders-shared"},
	{Storage: "Code - OSS", Shared: ".vscode-oss-shared"},
}

func vscodeStorageRoots(goos string) []string {
	return vscodeStoragePaths(vscodeStorageBase(goos))
}

func vscodeStorageBase(goos string) string {
	home, _ := os.UserHomeDir()
	switch goos {
	case "windows":
		return os.Getenv("AppData")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support")
	default:
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome != "" {
			return configHome
		}
		return filepath.Join(home, ".config")
	}
}

func vscodeStoragePaths(base string) []string {
	items := make([]string, 0, len(vscodeStorageProducts))
	for _, product := range vscodeStorageProducts {
		items = append(items, filepath.Join(base, product.Storage))
	}
	return items
}

func vscodeRecentSourceSets(goos string) []vscodeRecentSourceSet {
	base := vscodeStorageBase(goos)
	home, _ := os.UserHomeDir()
	items := make([]vscodeRecentSourceSet, 0, len(vscodeStorageProducts))
	for _, product := range vscodeStorageProducts {
		root := filepath.Join(base, product.Storage)
		items = append(items, vscodeRecentSourceSet{
			Primary:  filepath.Join(home, product.Shared, "sharedStorage"),
			Fallback: vscodeRecentSources(root),
		})
	}
	return items
}

func vscodeSharedStorageDirs(goos string) []string {
	items := make([]string, 0, len(vscodeStorageProducts))
	home, _ := os.UserHomeDir()
	for _, product := range vscodeStorageProducts {
		items = append(items, filepath.Join(home, product.Shared, "sharedStorage"))
	}
	return items
}

func vscodeStorageDirs(goos string) []string {
	roots := vscodeStorageRoots(goos)
	items := make([]string, 0, len(roots))
	for _, root := range roots {
		items = append(items, filepath.Join(root, "User", "globalStorage"))
	}
	return items
}

func vscodeRecentSources(root string) []string {
	return []string{filepath.Join(root, "User", "globalStorage")}
}

func vscodeRecentFromState(ctx context.Context, statePath string, goos string) ([]Project, error) {
	if _, err := os.Stat(statePath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(statePath)+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	raw, err := vscodeRecentStateValue(ctx, db)
	if err != nil {
		return nil, err
	}
	var list vscodeOpenedPathsList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil, err
	}
	return projectsFromVSCodeOpenedPaths(list, goos), nil
}

func vscodeRecentStateValue(ctx context.Context, db *sql.DB) (string, error) {
	var raw string
	err := db.QueryRowContext(ctx, "SELECT value FROM ItemTable WHERE key = ?", "history.recentlyOpenedPathsList").Scan(&raw)
	return raw, err
}

type vscodeOpenedPathsList struct {
	Entries     []vscodeRecentEntry `json:"entries"`
	Workspaces  []vscodeRecentEntry `json:"workspaces"`
	Workspaces3 []string            `json:"workspaces3"`
}

type vscodeRecentEntry struct {
	FolderURI       vscodeURI             `json:"folderUri"`
	FileURI         vscodeURI             `json:"fileUri"`
	Label           string                `json:"label"`
	RemoteAuthority string                `json:"remoteAuthority"`
	Workspace       vscodeRecentWorkspace `json:"workspace"`
}

type vscodeRecentWorkspace struct {
	ConfigPath vscodeURI `json:"configPath"`
	ConfigURI  vscodeURI `json:"configURI"`
	URI        vscodeURI `json:"uri"`
}

type vscodeURI string

func (v *vscodeURI) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = vscodeURI(text)
		return nil
	}
	var object struct {
		External  string `json:"external"`
		Scheme    string `json:"scheme"`
		Authority string `json:"authority"`
		Path      string `json:"path"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	if object.External != "" {
		*v = vscodeURI(object.External)
		return nil
	}
	if object.Scheme != "" {
		*v = vscodeURI((&url.URL{Scheme: object.Scheme, Host: object.Authority, Path: object.Path}).String())
	}
	return nil
}

func (v vscodeURI) String() string {
	return string(v)
}

func projectsFromVSCodeOpenedPaths(list vscodeOpenedPathsList, goos string) []Project {
	seen := newProjectSet(0)
	for _, uri := range list.Workspaces3 {
		path := uriToVSCodeRecentPath(uri, "", goos)
		if path == "" || !vscodeRecentPathExists(path, goos) {
			continue
		}
		seen.add(newVSCodeRecentProject("", path, projectKindVSCodeRecent, vscodeRecentURIAuthority(uri, "")))
	}
	addEntries := func(entries []vscodeRecentEntry) {
		for _, entry := range entries {
			path, kind, remoteAuthority := vscodeRecentPath(entry, goos)
			if path == "" || !vscodeRecentPathExists(path, goos) {
				continue
			}
			seen.add(newVSCodeRecentProject(entry.Label, path, kind, remoteAuthority))
		}
	}
	addEntries(list.Entries)
	addEntries(list.Workspaces)
	return seen.sorted()
}

func vscodeRecentPath(entry vscodeRecentEntry, goos string) (string, string, string) {
	for _, value := range []vscodeURI{entry.Workspace.ConfigPath, entry.Workspace.ConfigURI, entry.Workspace.URI} {
		if value == "" {
			continue
		}
		if path := uriToVSCodeRecentPath(value.String(), entry.RemoteAuthority, goos); path != "" {
			return path, projectKindVSCodeWorkspace, vscodeRecentURIAuthority(value.String(), entry.RemoteAuthority)
		}
	}
	if entry.FolderURI != "" {
		if path := uriToVSCodeRecentPath(entry.FolderURI.String(), entry.RemoteAuthority, goos); path != "" {
			return path, projectKindVSCodeRecent, vscodeRecentURIAuthority(entry.FolderURI.String(), entry.RemoteAuthority)
		}
	}
	if entry.FileURI != "" && strings.EqualFold(pathpkg.Ext(entry.FileURI.String()), ".code-workspace") {
		if path := uriToVSCodeRecentPath(entry.FileURI.String(), entry.RemoteAuthority, goos); path != "" {
			return path, projectKindVSCodeWorkspace, vscodeRecentURIAuthority(entry.FileURI.String(), entry.RemoteAuthority)
		}
	}
	return "", "", ""
}

func vscodeRecentURIAuthority(value string, remoteAuthority string) string {
	if authority := vscodeRemoteAuthority(remoteAuthority); authority != "" {
		return authority
	}
	if unescaped, err := url.PathUnescape(value); err == nil {
		value = unescaped
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "vscode-remote" {
		return ""
	}
	return vscodeRemoteAuthority(parsed.Host)
}

func uriToVSCodeRecentPath(value string, remoteAuthority string, goos string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(value); err == nil {
		value = unescaped
	}
	remoteAuthority = vscodeRemoteAuthority(remoteAuthority)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		if remoteAuthority != "" && strings.HasPrefix(value, "/") {
			return remoteRecentPath(remoteAuthority, value)
		}
		return filepath.Clean(value)
	}
	if parsed.Scheme == "file" {
		if remoteAuthority != "" {
			return remoteRecentPath(remoteAuthority, parsed.Path)
		}
		path := parsed.Path
		if goos == "windows" {
			if parsed.Host != "" {
				path = `\\` + parsed.Host + filepath.FromSlash(path)
			} else {
				path = strings.TrimPrefix(path, "/")
				path = filepath.FromSlash(path)
			}
		} else {
			path = filepath.FromSlash(path)
		}
		return filepath.Clean(path)
	}
	remoteHost := vscodeRemoteAuthority(parsed.Host)
	if parsed.Scheme == "vscode-remote" && remoteHost != "" && parsed.Path != "" {
		return remoteRecentPath(remoteHost, parsed.Path)
	}
	if remoteAuthority != "" && parsed.Path != "" {
		return remoteRecentPath(remoteAuthority, parsed.Path)
	}
	return ""
}

func remoteRecentPath(remoteAuthority string, remotePath string) string {
	if isWslRemoteAuthority(remoteAuthority) {
		return pathpkg.Clean(remotePath)
	}
	return (&url.URL{Scheme: "vscode-remote", Host: remoteAuthority, Path: pathpkg.Clean(remotePath)}).String()
}

func vscodeRemoteAuthority(value string) string {
	if unescaped, err := url.PathUnescape(value); err == nil {
		return unescaped
	}
	return value
}

func isVSCodeRemoteURI(path string) bool {
	_, ok := parseVSCodeRemoteURI(path)
	return ok
}

func parseVSCodeRemoteURI(path string) (*url.URL, bool) {
	parsed, err := url.Parse(path)
	return parsed, err == nil && parsed.Scheme == "vscode-remote" && parsed.Host != ""
}

func vscodeRecentName(path string) string {
	if parsed, ok := parseVSCodeRemoteURI(path); ok && parsed.Path != "" {
		name := pathpkg.Base(parsed.Path)
		return strings.TrimSuffix(name, pathpkg.Ext(name))
	}
	return workspaceName(path)
}

func newVSCodeRecentProject(label string, path string, kind string, remoteAuthority string) Project {
	displayName := vscodeRecentDisplayName(label, path, kind, remoteAuthority)
	project := newProject(displayName, path, kind)
	project.DisplayName = displayName
	return project
}

func vscodeRecentDisplayName(label string, path string, kind string, remoteAuthority string) string {
	if name, _ := splitVSCodeRecentLabel(strings.TrimSpace(label)); name != "" {
		return name
	}
	name := vscodeRecentName(path)
	if kind == projectKindVSCodeWorkspace {
		name += " (Workspace)"
	}
	if suffix := vscodeRemoteSuffix(path, remoteAuthority); suffix != "" {
		name += " [" + suffix + "]"
	}
	return name
}

func splitVSCodeRecentLabel(label string) (string, string) {
	if label == "" {
		return "", ""
	}
	if strings.HasSuffix(label, "]") {
		if suffixIndex := strings.LastIndex(label[:len(label)-1], " ["); suffixIndex != -1 {
			name, parentPath := splitVSCodeRecentName(label[:suffixIndex])
			return name + label[suffixIndex:], parentPath
		}
	}
	return splitVSCodeRecentName(label)
}

func splitVSCodeRecentName(fullPath string) (string, string) {
	if strings.Contains(fullPath, "/") {
		name := pathpkg.Base(fullPath)
		parentPath := pathpkg.Dir(fullPath)
		if name != "" && name != "." {
			return name, parentPath
		}
		if parentPath == "." {
			return fullPath, ""
		}
		return parentPath, ""
	}
	if index := strings.LastIndex(fullPath, `\`); index != -1 {
		name := fullPath[index+1:]
		parentPath := fullPath[:index]
		if name != "" {
			return name, parentPath
		}
		return parentPath, ""
	}
	return fullPath, ""
}

func vscodeRemoteSuffix(path string, remoteAuthority string) string {
	authority := vscodeRemoteAuthority(remoteAuthority)
	if authority == "" {
		if parsed, ok := parseVSCodeRemoteURI(path); ok {
			authority = parsed.Host
		}
	}
	lower := strings.ToLower(authority)
	if strings.HasPrefix(lower, "ssh-remote+") {
		return "SSH: " + authority[len("ssh-remote+"):]
	}
	if strings.HasPrefix(lower, "wsl+") {
		distro := authority[len("wsl+"):]
		if strings.EqualFold(distro, "default") || distro == "" {
			return "WSL"
		}
		return "WSL: " + distro
	}
	return authority
}

func isWslRemoteAuthority(value string) bool {
	return strings.HasPrefix(strings.ToLower(value), "wsl+")
}

func vscodeRecentPathExists(path string, goos string) bool {
	if isVSCodeRemoteURI(path) {
		return true
	}
	if isUnixPath(path) && goos == "windows" {
		return true
	}
	if !vscodeRecentPathAllowed(path, goos) {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func vscodeRecentPathAllowed(path string, goos string) bool {
	if goos != "windows" || isUnixPath(path) {
		return true
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		clean = filepath.Clean(path)
	}
	for _, systemDir := range absoluteCleanPaths(windowsSystemDirs()) {
		if isPathWithin(clean, systemDir) {
			return false
		}
	}
	return true
}

func isPathWithin(path string, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if strings.EqualFold(path, root) {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
