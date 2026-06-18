package lyn

import "testing"

func TestSplitScannerRootsSeparatesWSL(t *testing.T) {
	windows, wsl := splitScannerRoots([]string{
		`C:\Users\me\src`,
		`~/src`,
		`\\wsl.localhost\Ubuntu\home\me\src`,
		`/home/me/www`,
	}, nil)

	wantWindows := []string{`C:\Users\me\src`, `~/src`}
	if len(windows) != len(wantWindows) {
		t.Fatalf("windows roots = %#v, want %#v", windows, wantWindows)
	}
	for i, root := range wantWindows {
		if windows[i] != root {
			t.Fatalf("windows[%d] = %q, want %q", i, windows[i], root)
		}
	}

	if len(wsl) != 2 {
		t.Fatalf("wsl roots = %#v, want 2", wsl)
	}
	if wsl[0].Distro != "Ubuntu" || wsl[0].Path != "/home/me/src" {
		t.Fatalf("unexpected migrated UNC root %#v", wsl[0])
	}
	if wsl[1].Distro != "" || wsl[1].Path != "/home/me/www" {
		t.Fatalf("unexpected migrated unix root %#v", wsl[1])
	}
}

func TestNormalizeWSLRootsDedupesAndTrims(t *testing.T) {
	roots := normalizeWSLRoots([]WSLRoot{
		{Distro: "Ubuntu", Path: "/home/me/src/"},
		{Distro: "Ubuntu", Path: "/home/me/src"},
		{Distro: "", Path: "  "},
	})
	if len(roots) != 1 {
		t.Fatalf("expected 1 deduped root, got %#v", roots)
	}
	if roots[0].Path != "/home/me/src" {
		t.Fatalf("unexpected normalized path %q", roots[0].Path)
	}
}

func TestWSLScannedProjectConvertsUNCToUnix(t *testing.T) {
	project := wslScannedProject(Project{
		Name: "app",
		Path: `\\wsl.localhost\Ubuntu\home\me\src\app`,
		Kind: projectKindGo,
	})
	if project.Path != "/home/me/src/app" || project.Distro != "Ubuntu" {
		t.Fatalf("unexpected scanned project %#v", project)
	}

	local := wslScannedProject(Project{Name: "app", Path: `C:\src\app`, Kind: projectKindGo})
	if local.Path != `C:\src\app` || local.Distro != "" {
		t.Fatalf("local project must be unchanged, got %#v", local)
	}
}
