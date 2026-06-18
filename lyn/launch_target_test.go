package lyn

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lyn.tools/launcher/lyn/launch"
)

func TestWordpressCodeTargetPrefersWorkspaceThenNewestTheme(t *testing.T) {
	root := t.TempDir()
	site := filepath.Join(root, "site")
	older := filepath.Join(site, "wp-content", "themes", "older")
	newer := filepath.Join(site, "wp-content", "themes", "newer")
	for _, dir := range []string{older, newer} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-time.Hour)
	recent := time.Now()
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, recent, recent); err != nil {
		t.Fatal(err)
	}
	project := Project{Path: site, Kind: projectKindWordPress}

	if target, ok := wordpressCodeTarget(project, nil); !ok || target != newer {
		t.Fatalf("expected newest theme %q, got %q ok=%v", newer, target, ok)
	}

	workspace := filepath.Join(site, "site.code-workspace")
	if err := os.WriteFile(workspace, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if target, ok := wordpressCodeTarget(project, []string{workspace}); !ok || target != workspace {
		t.Fatalf("expected workspace %q, got %q ok=%v", workspace, target, ok)
	}
}

func TestResolveLaunchTargetWordpressVsOthers(t *testing.T) {
	root := t.TempDir()
	site := filepath.Join(root, "site")
	theme := filepath.Join(site, "wp-content", "themes", "main")
	if err := os.MkdirAll(theme, 0o755); err != nil {
		t.Fatal(err)
	}
	laravel := filepath.Join(root, "api")
	goProject := filepath.Join(root, "cli")
	if err := os.MkdirAll(laravel, 0o755); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.setSearchIndex([]Project{
		{Path: site, Kind: projectKindWordPress},
		{Path: laravel, Kind: projectKindLaravel},
		{Path: goProject, Kind: projectKindGo},
	})

	if got := app.resolveLaunchTarget(launch.Request{Path: site, Action: "code"}); got.Path != theme {
		t.Fatalf("wordpress should open theme %q, got %q", theme, got.Path)
	}
	for _, path := range []string{laravel, goProject} {
		if got := app.resolveLaunchTarget(launch.Request{Path: path, Action: "code"}); got.Path != path {
			t.Fatalf("non-wordpress root should be unchanged, got %q for %q", got.Path, path)
		}
	}
	if got := app.resolveLaunchTarget(launch.Request{Path: site, Action: "reveal"}); got.Path != site {
		t.Fatalf("non-code action should be unchanged, got %q", got.Path)
	}
}

func TestIsLaunchPathUnder(t *testing.T) {
	cases := []struct {
		child  string
		parent string
		want   bool
	}{
		{"/home/me/www/site/x.code-workspace", "/home/me/www/site", true},
		{`C:\www\site\x.code-workspace`, `C:\www\site`, true},
		{"/home/me/www/other/x", "/home/me/www/site", false},
		{"/home/me/www/site", "/home/me/www/site", false},
	}
	for _, tc := range cases {
		if got := isLaunchPathUnder(tc.child, tc.parent); got != tc.want {
			t.Fatalf("isLaunchPathUnder(%q,%q)=%v want %v", tc.child, tc.parent, got, tc.want)
		}
	}
}
