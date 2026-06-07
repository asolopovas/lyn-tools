package lyn

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestCleanUniquePathsTrimsCleansAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	dirty := filepath.Join(root, ".")
	got := cleanUniquePaths([]string{"", " " + dirty + " ", root})
	want := []string{filepath.Clean(root)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected paths %#v, want %#v", got, want)
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
