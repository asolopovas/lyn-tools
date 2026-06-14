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
