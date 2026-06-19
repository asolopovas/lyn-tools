package lyn

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func requireWindowsHost(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("Windows path semantics require a Windows host; covered by the windows CI job")
	}
}

func TestCleanUniquePathsTrimsCleansAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	dirty := filepath.Join(root, ".")
	got := cleanUniquePaths([]string{"", " " + dirty + " ", root})
	want := []string{filepath.Clean(root)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected paths %#v, want %#v", got, want)
	}
}

func TestShouldSkipDirectories(t *testing.T) {
	skipped := []string{".git", "node_modules", "vendor", ".next", "dist", "build", "target", ".cache"}
	for _, name := range skipped {
		if !shouldSkip(name) {
			t.Fatalf("expected %q to be skipped", name)
		}
	}
	kept := []string{"src", "lyn", "my-app", "app", "internal", "@scope"}
	for _, name := range kept {
		if shouldSkip(name) {
			t.Fatalf("expected %q to be kept", name)
		}
	}
}

func TestIsPackagedDependency(t *testing.T) {
	packaged := []string{"accepts@1.3.8@@@1", "acorn@8.15.0", "@babel+core@7.0.0", "react@18", "@scope/name@2.0.0"}
	for _, name := range packaged {
		if !isPackagedDependency(name) {
			t.Fatalf("expected %q to be a packaged dependency", name)
		}
	}
	plain := []string{"my-app", "user@host", "@scope", "v1", "name@beta", ""}
	for _, name := range plain {
		if isPackagedDependency(name) {
			t.Fatalf("expected %q not to be a packaged dependency", name)
		}
	}
}

func TestIsWindowsStartupDir(t *testing.T) {
	if !isWindowsStartupDir("Startup", "windows") || !isWindowsStartupDir("startup", "windows") {
		t.Fatal("expected the Startup folder to be detected on windows")
	}
	if isWindowsStartupDir("Startup", "linux") {
		t.Fatal("did not expect the Startup folder to be skipped off windows")
	}
	if isWindowsStartupDir("Programs", "windows") {
		t.Fatal("did not expect a non-startup folder to match")
	}
}

func TestWithinWindowsSystemDir(t *testing.T) {
	requireWindowsHost(t)
	t.Setenv("SystemRoot", `C:\Windows`)
	t.Setenv("WINDIR", `C:\Windows`)
	within := []string{`C:\Windows`, `C:\Windows\System32`, `C:\Windows\System32\drivers`}
	for _, path := range within {
		if !withinWindowsSystemDir(path) {
			t.Fatalf("expected %q to be inside a system directory", path)
		}
	}
	outside := []string{`C:\Program Files\App\bin`, `C:\Users\example\project`}
	for _, path := range outside {
		if withinWindowsSystemDir(path) {
			t.Fatalf("expected %q to be outside system directories", path)
		}
	}
}

func TestIsPathWithin(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "nested", "leaf")
	if !isPathWithin(root, root) {
		t.Fatal("expected a path to contain itself")
	}
	if !isPathWithin(child, root) {
		t.Fatalf("expected %q to be within %q", child, root)
	}
	sibling := filepath.Join(filepath.Dir(root), "other")
	if isPathWithin(sibling, root) {
		t.Fatalf("expected %q to be outside %q", sibling, root)
	}
}

func TestWindowsSystemDirsFallsBackWhenEnvIsEmpty(t *testing.T) {
	t.Setenv("SystemRoot", "")
	t.Setenv("WINDIR", "")
	got := windowsSystemDirs()
	want := []string{filepath.Clean(`C:\Windows`)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected system dirs %#v, want %#v", got, want)
	}
}
