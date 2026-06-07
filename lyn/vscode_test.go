package lyn

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "modernc.org/sqlite"
)

func TestScanVSCodeRecentProjectsFromStateDB(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	storage := filepath.Join(root, "Code")
	if err := os.MkdirAll(filepath.Join(storage, "User", "globalStorage"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(storage, "User", "globalStorage", "state.vscdb")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE ItemTable(key TEXT PRIMARY KEY, value TEXT)"); err != nil {
		t.Fatal(err)
	}
	raw := `{"entries":[{"folderUri":"` + testFileURI(project) + `","label":"Recent Project"}]}`
	if _, err := db.Exec("INSERT INTO ItemTable(key, value) VALUES(?, ?)", "history.recentlyOpenedPathsList", raw); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	items, err := scanVSCodeRecentProjects(t.Context(), []string{storage}, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "Recent Project" || items[0].Kind != "vscode-recent" || items[0].Path != project {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestVSCodeSharedStorageOverridesStaleLegacyStorage(t *testing.T) {
	root := t.TempDir()
	currentProject := filepath.Join(root, "current")
	staleProject := filepath.Join(root, "stale")
	for _, project := range []string{currentProject, staleProject} {
		if err := os.Mkdir(project, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	shared := filepath.Join(root, ".vscode-shared", "sharedStorage")
	legacy := filepath.Join(root, "Code", "User", "globalStorage")
	if err := writeVSCodeRecentState(filepath.Join(shared, "state.vscdb"), `{"entries":[{"folderUri":"`+testFileURI(currentProject)+`","label":"current"}]}`); err != nil {
		t.Fatal(err)
	}
	if err := writeVSCodeRecentState(filepath.Join(legacy, "state.vscdb"), `{"entries":[{"folderUri":"`+testFileURI(staleProject)+`","label":"stale"}]}`); err != nil {
		t.Fatal(err)
	}
	items, err := scanVSCodeRecentSourceSets(t.Context(), []vscodeRecentSourceSet{{Primary: shared, Fallback: []string{legacy}}}, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectPaths(t, items, []string{currentProject})
}

func TestVSCodeRecentSourcePathsFollowVSCodeSharedStorage(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("AppData", appData)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	windowsSets := vscodeRecentSourceSets("windows")
	if len(windowsSets) == 0 || windowsSets[0].Primary != filepath.Join(home, ".vscode-shared", "sharedStorage") || windowsSets[0].Fallback[0] != filepath.Join(appData, "Code", "User", "globalStorage") {
		t.Fatalf("unexpected windows VS Code sources %#v", windowsSets[:1])
	}
	linuxSets := vscodeRecentSourceSets("linux")
	if len(linuxSets) == 0 || linuxSets[0].Primary != filepath.Join(home, ".vscode-shared", "sharedStorage") || linuxSets[0].Fallback[0] != filepath.Join(configHome, "Code", "User", "globalStorage") {
		t.Fatalf("unexpected linux VS Code sources %#v", linuxSets[:1])
	}
}

func TestVSCodeRecentStateReadsOfficialHistoryKey(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "current")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	storage := filepath.Join(root, "Code")
	dbPath := filepath.Join(storage, "User", "globalStorage", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE ItemTable(key TEXT PRIMARY KEY, value TEXT)"); err != nil {
		t.Fatal(err)
	}
	raw := `{"entries":[{"folderUri":"` + testFileURI(project) + `","label":"current"}]}`
	if _, err := db.Exec("INSERT INTO ItemTable(key, value) VALUES(?, ?)", "history.recentlyOpenedPathsList", raw); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	items, err := scanVSCodeRecentProjects(t.Context(), []string{storage}, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectPaths(t, items, []string{project})
}

func TestParseVSCodeRemoteWslRecentProject(t *testing.T) {
	raw := `{"entries":[{"folderUri":"vscode-remote://wsl%2Bubuntu/home/example/project","remoteAuthority":"wsl+ubuntu"}]}`
	items := parseVSCodeRecentProjectsForTest(raw, "windows")
	if len(items) != 1 || items[0].Name != "project [WSL: ubuntu]" || items[0].Path != "/home/example/project" {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestParseVSCodeRemoteWslWorkspaceWithoutRemoteAuthority(t *testing.T) {
	raw := `{"entries":[{"workspace":{"id":"id","configPath":"vscode-remote://wsl%2Bubuntu/home/example/www/example.test/wp-content/themes/example_theme/example.code-workspace"}}]}`
	items := parseVSCodeRecentProjectsForTest(raw, "windows")
	want := "/home/example/www/example.test/wp-content/themes/example_theme/example.code-workspace"
	if len(items) != 1 || items[0].Name != "example (Workspace) [WSL: ubuntu]" || items[0].DisplayName != items[0].Name || items[0].Kind != "vscode-workspace" || items[0].Path != want {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestParseVSCodeRecentSplitsVSCodePathLabel(t *testing.T) {
	raw := `{"entries":[{"label":"/srv/www/example.test/wp-content/themes/example_theme/example (Workspace) [SSH: examplehost]","workspace":{"configPath":"vscode-remote://ssh-remote%2Bexamplehost/srv/www/example.test/wp-content/themes/example_theme/example.code-workspace"}}]}`
	items := parseVSCodeRecentProjectsForTest(raw, "windows")
	want := "vscode-remote://ssh-remote+examplehost/srv/www/example.test/wp-content/themes/example_theme/example.code-workspace"
	if len(items) != 1 || items[0].Name != "example (Workspace) [SSH: examplehost]" || items[0].DisplayName != items[0].Name || items[0].Kind != "vscode-workspace" || items[0].Path != want {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestParseVSCodeRecentSplitsWindowsPathLabel(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"entries":[{"folderUri":"` + testFileURI(project) + `","label":"C:\\Users\\example\\project"}]}`
	items := parseVSCodeRecentProjectsForTest(raw, runtime.GOOS)
	if len(items) != 1 || items[0].Name != "project" || items[0].DisplayName != items[0].Name || items[0].Path != project {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestParseVSCodeRemoteSSHWorkspaceRecent(t *testing.T) {
	raw := `{"entries":[{"label":"example","workspace":{"configPath":"vscode-remote://ssh-remote%2Bexamplehost/home/deploy/example.code-workspace"}}]}`
	items := parseVSCodeRecentProjectsForTest(raw, "windows")
	want := "vscode-remote://ssh-remote+examplehost/home/deploy/example.code-workspace"
	if len(items) != 1 || items[0].Name != "example" || items[0].Kind != "vscode-workspace" || items[0].Path != want {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestParseVSCodeRecentURIObject(t *testing.T) {
	raw := `{"entries":[{"folderUri":{"$mid":1,"scheme":"vscode-remote","authority":"ssh-remote+examplehost","path":"/srv/www/example"}}]}`
	items := parseVSCodeRecentProjectsForTest(raw, "windows")
	want := "vscode-remote://ssh-remote+examplehost/srv/www/example"
	if len(items) != 1 || items[0].Name != "example [SSH: examplehost]" || items[0].DisplayName != items[0].Name || items[0].Kind != "vscode-recent" || items[0].Path != want {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestWindowsSystemRecentPathIsSkipped(t *testing.T) {
	t.Setenv("SystemRoot", `C:\Windows`)
	t.Setenv("WINDIR", `C:\Windows`)
	if vscodeRecentPathAllowed(`C:\Windows`, "windows") {
		t.Fatal("expected Windows directory to be skipped")
	}
	if vscodeRecentPathAllowed(`C:\Windows\System32`, "windows") {
		t.Fatal("expected Windows child directory to be skipped")
	}
	if !vscodeRecentPathAllowed(`C:\Users\example\project`, "windows") {
		t.Fatal("expected user folder to be allowed")
	}
	if !vscodeRecentPathAllowed(`/home/example/project`, "windows") {
		t.Fatal("expected WSL path to be allowed")
	}
}

func parseVSCodeRecentProjectsForTest(raw string, goos string) []Project {
	var list vscodeOpenedPathsList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	return projectsFromVSCodeOpenedPaths(list, goos)
}

func writeVSCodeRecentState(path string, raw string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE ItemTable(key TEXT PRIMARY KEY, value TEXT)"); err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO ItemTable(key, value) VALUES(?, ?)", "history.recentlyOpenedPathsList", raw)
	return err
}

func testFileURI(path string) string {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		return "file:///" + path
	}
	return "file://" + path
}

func assertProjectPaths(t *testing.T, items []Project, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("unexpected items %#v, want paths %#v", items, want)
	}
	for i, path := range want {
		if items[i].Path != path {
			t.Fatalf("unexpected items %#v, want paths %#v", items, want)
		}
	}
}
