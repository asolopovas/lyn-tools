package lyn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "site")
	if err := os.MkdirAll(filepath.Join(project, "wp-content"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "wp-config.php"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	item, ok := DetectProject(project)
	if !ok {
		t.Fatal("project not detected")
	}
	if item.Kind != "wordpress" {
		t.Fatalf("unexpected kind %q", item.Kind)
	}
}

func TestDetectCommonProjectManifests(t *testing.T) {
	cases := []struct {
		name string
		file string
		kind string
	}{
		{name: "go-app", file: "go.mod", kind: "go"},
		{name: "node-app", file: "package.json", kind: "node"},
		{name: "rust-app", file: "Cargo.toml", kind: "rust"},
	}
	root := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			project := filepath.Join(root, tc.name)
			if err := os.MkdirAll(project, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, tc.file), []byte{}, 0o644); err != nil {
				t.Fatal(err)
			}
			item, ok := DetectProject(project)
			if !ok {
				t.Fatal("project not detected")
			}
			if item.Kind != tc.kind {
				t.Fatalf("unexpected kind %q", item.Kind)
			}
		})
	}
}

func TestDetectCodeWorkspaceFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.code-workspace")
	if err := os.WriteFile(path, []byte(`{"folders":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	item, ok := DetectProject(path)
	if !ok {
		t.Fatal("workspace not detected")
	}
	if item.Name != "client" || item.Kind != "vscode-workspace" || item.Path != path {
		t.Fatalf("unexpected workspace %#v", item)
	}
}

func TestScanProjectsFindsCodeWorkspaceFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "client.code-workspace"), []byte(`{"folders":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := ScannerConfig{Roots: []string{root}, MaxDepth: 2, Concurrency: 2, Timeout: "20s"}
	projects, skipped, err := ScanProjects(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped roots %#v", skipped)
	}
	if len(projects) != 1 {
		t.Fatalf("unexpected project count %d", len(projects))
	}
	if projects[0].Kind != "vscode-workspace" {
		t.Fatalf("unexpected kind %q", projects[0].Kind)
	}
}

func TestDetectGitRepositoryWithoutManifest(t *testing.T) {
	project := filepath.Join(t.TempDir(), "dotfiles")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("dotfiles"), 0o644); err != nil {
		t.Fatal(err)
	}
	item, ok := DetectProject(project)
	if !ok {
		t.Fatal("git repository not detected")
	}
	if item.Kind != "git" || item.Name != "dotfiles" {
		t.Fatalf("unexpected project %#v", item)
	}
}

func TestDetectManifestTakesPrecedenceOverGit(t *testing.T) {
	cases := []struct {
		name string
		file string
		kind string
	}{
		{name: "go-repo", file: "go.mod", kind: "go"},
		{name: "vscode-repo", file: filepath.Join(".vscode", "settings.json"), kind: "vscode-workspace"},
	}
	root := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			project := filepath.Join(root, tc.name)
			if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(filepath.Join(project, tc.file)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, tc.file), []byte{}, 0o644); err != nil {
				t.Fatal(err)
			}
			item, ok := DetectProject(project)
			if !ok {
				t.Fatal("project not detected")
			}
			if item.Kind != tc.kind {
				t.Fatalf("unexpected kind %q", item.Kind)
			}
		})
	}
}

func TestScanProjectsIndexesGitRepositoryRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "winconf")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := ScannerConfig{Roots: []string{root}, MaxDepth: 5, Concurrency: 2, Timeout: "20s"}
	projects, skipped, err := ScanProjects(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped roots %#v", skipped)
	}
	if len(projects) != 1 || projects[0].Kind != "git" || projects[0].Path != root {
		t.Fatalf("unexpected projects %#v", projects)
	}
}

func TestScanProjectsReportsUnreachableRootsAndKeepsResults(t *testing.T) {
	good := t.TempDir()
	if err := os.WriteFile(filepath.Join(good, "go.mod"), []byte("module example.test/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	cfg := ScannerConfig{Roots: []string{good, missing}, MaxDepth: 2, Concurrency: 2, Timeout: "20s"}
	projects, skipped, err := ScanProjects(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Kind != "go" {
		t.Fatalf("expected the reachable root to still be scanned, got %#v", projects)
	}
	if len(skipped) != 1 || skipped[0] != filepath.Clean(missing) {
		t.Fatalf("unexpected skipped roots %#v", skipped)
	}
}

func TestScanProjectsReportsEveryUnreachableRoot(t *testing.T) {
	first := filepath.Join(t.TempDir(), "absent-a")
	second := filepath.Join(t.TempDir(), "absent-b")
	cfg := ScannerConfig{Roots: []string{first, second}, MaxDepth: 2, Concurrency: 2, Timeout: "20s"}
	projects, skipped, err := ScanProjects(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected no projects, got %#v", projects)
	}
	if len(skipped) != 2 {
		t.Fatalf("expected both roots reported as skipped, got %#v", skipped)
	}
}

func TestExpandRoots(t *testing.T) {
	items := expandRoots([]string{"~/src", "~/www"})
	if len(items) != 2 {
		t.Fatalf("unexpected count %d", len(items))
	}
	if filepath.Base(items[0]) != "src" {
		t.Fatalf("unexpected root %q", items[0])
	}
}

func TestExpandRootsConvertsWindowsWslPaths(t *testing.T) {
	items := expandRootsForOS([]string{"/home/me/src"}, "windows", func(path string) (string, bool) {
		if path != "/home/me/src" {
			t.Fatalf("unexpected path %q", path)
		}
		return `\\wsl.localhost\Ubuntu\home\me\src`, true
	})
	if len(items) != 1 {
		t.Fatalf("unexpected count %d", len(items))
	}
	if items[0] != `\\wsl.localhost\Ubuntu\home\me\src` {
		t.Fatalf("unexpected root %q", items[0])
	}
}

func TestExpandRootsKeepsWindowsWslUNCPaths(t *testing.T) {
	root := `\\wsl.localhost\Ubuntu\home\me\src`
	items := expandRootsForOS([]string{root}, "windows", func(string) (string, bool) {
		t.Fatal("unexpected converter call")
		return "", false
	})
	if len(items) != 1 || items[0] != root {
		t.Fatalf("unexpected roots %#v", items)
	}
}

func TestIsUnixPath(t *testing.T) {
	if !isUnixPath("/home/me/src") {
		t.Fatal("expected Linux path to be Unix style")
	}
	if isUnixPath(`C:\Users\me\src`) {
		t.Fatal("did not expect Windows path to be Unix style")
	}
}

func BenchmarkScanProjects(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 50; i++ {
		project := filepath.Join(root, fmt.Sprintf("site-%02d", i), "nested")
		if err := os.MkdirAll(filepath.Join(project, ".vscode"), 0o755); err != nil {
			b.Fatal(err)
		}
	}
	cfg := ScannerConfig{Roots: []string{root}, MaxDepth: 4, Concurrency: 4, Timeout: "20s"}
	for b.Loop() {
		projects, _, err := ScanProjects(context.Background(), cfg)
		if err != nil {
			b.Fatal(err)
		}
		if len(projects) != 50 {
			b.Fatalf("unexpected project count %d", len(projects))
		}
	}
}
