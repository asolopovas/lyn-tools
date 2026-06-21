package sysadmin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestToolsAreUniqueAndNamed(t *testing.T) {
	tools := Tools()
	if len(tools) == 0 {
		t.Fatal("expected at least one system tool")
	}
	seen := map[string]bool{}
	for _, tool := range tools {
		if tool.Key == "" || tool.Name == "" {
			t.Fatalf("tool has empty key or name: %+v", tool)
		}
		if seen[tool.Key] {
			t.Fatalf("duplicate tool key %q", tool.Key)
		}
		seen[tool.Key] = true
	}
}

func TestEnsureScriptWritesExecutableAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path, err := EnsureScript(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("script written outside target dir: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("script is not executable: %v", info.Mode())
	}
	again, err := EnsureScript(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again != path {
		t.Fatalf("idempotent call returned different path %q vs %q", again, path)
	}
}
