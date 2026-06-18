package lyn

import (
	"runtime"
	"testing"
)

func TestProjectKeyForOSCollapsesWindowsDriveCase(t *testing.T) {
	upper := projectKeyForOS(`C:\Users\asolo\winconf`, "windows")
	lower := projectKeyForOS(`c:\Users\asolo\winconf`, "windows")
	if upper != lower {
		t.Fatalf("expected case-insensitive key on windows, got %q and %q", upper, lower)
	}
}

func TestProjectKeyForOSKeepsRemoteAndUnixCasing(t *testing.T) {
	remote := "vscode-remote://ssh-remote+Host/srv/App"
	if projectKeyForOS(remote, "windows") != remote {
		t.Fatalf("expected remote path to keep casing, got %q", projectKeyForOS(remote, "windows"))
	}
	unix := "/home/me/Src/app"
	if projectKeyForOS(unix, "linux") != unix {
		t.Fatalf("expected unix path to keep casing, got %q", projectKeyForOS(unix, "linux"))
	}
	if projectKeyForOS(unix, "windows") != unix {
		t.Fatalf("expected unix path to keep casing on windows, got %q", projectKeyForOS(unix, "windows"))
	}
}

func TestKindRankOrdersSSHBelowWorkspaceAndWSL(t *testing.T) {
	workspace := Project{Kind: projectKindVSCodeWorkspace, Path: "/home/me/app.code-workspace"}
	wsl := newVSCodeRecentProject("", "vscode-remote://wsl+Ubuntu/home/me/app", projectKindVSCodeRecent, "")
	local := Project{Kind: projectKindGo, Path: "/home/me/local"}
	ssh := newVSCodeRecentProject("", "vscode-remote://ssh-remote+examplehost/srv/app", projectKindVSCodeRecent, "")
	system := newProject("Restart", systemCommandRestart, projectKindSystemCommand)

	if !(kindRank(workspace) < kindRank(wsl)) {
		t.Fatalf("workspace (%d) should rank above wsl (%d)", kindRank(workspace), kindRank(wsl))
	}
	if !(kindRank(wsl) < kindRank(ssh)) {
		t.Fatalf("wsl (%d) should rank above ssh (%d)", kindRank(wsl), kindRank(ssh))
	}
	if !(kindRank(local) < kindRank(ssh)) {
		t.Fatalf("local (%d) should rank above ssh (%d)", kindRank(local), kindRank(ssh))
	}
	if !(kindRank(ssh) < kindRank(system)) {
		t.Fatalf("ssh (%d) should rank above system command (%d)", kindRank(ssh), kindRank(system))
	}
}

func TestIsSSHRemoteProjectMatchesEncodedAuthority(t *testing.T) {
	cases := map[string]bool{
		"vscode-remote://ssh-remote+examplehost/srv/app": true,
		"vscode-remote://ssh-remote%2Bexamplehost/srv":   true,
		"vscode-remote://wsl+Ubuntu/home/me/app":         false,
		"/home/me/local":                                 false,
	}
	for path, want := range cases {
		if got := isSSHRemoteProject(Project{Path: path}); got != want {
			t.Fatalf("isSSHRemoteProject(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSearchIndexOrdersSSHBelowWSLForEqualMatch(t *testing.T) {
	wsl := newVSCodeRecentProject("app", "vscode-remote://wsl+Ubuntu/home/me/app", projectKindVSCodeRecent, "")
	ssh := newVSCodeRecentProject("app", "vscode-remote://ssh-remote+examplehost/srv/app", projectKindVSCodeRecent, "")
	matches := searchProjects([]Project{ssh, wsl}, "app", "")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if !isSSHRemoteProject(matches[1]) || isSSHRemoteProject(matches[0]) {
		t.Fatalf("expected wsl before ssh, got %q then %q", matches[0].Path, matches[1].Path)
	}
}

func TestMergeProjectsDedupesWindowsDriveCase(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows drive-case dedup only applies on windows")
	}
	scanned := Project{Name: "winconf", Path: `C:\Users\asolo\winconf`, Kind: projectKindGit, UsageCount: 3}
	recent := newVSCodeRecentProject("", `c:\Users\asolo\winconf`, projectKindVSCodeRecent, "")
	merged := mergeProjects([]Project{scanned}, []Project{recent})
	if len(merged) != 1 {
		t.Fatalf("expected a single merged entry, got %#v", merged)
	}
	if merged[0].Kind != projectKindVSCodeRecent {
		t.Fatalf("expected recent entry to win, got %q", merged[0].Kind)
	}
	if merged[0].UsageCount != 3 {
		t.Fatalf("expected usage count to carry over, got %d", merged[0].UsageCount)
	}
}
