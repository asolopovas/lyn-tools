package lyn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigUsesDefaultWhenMissing(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scanner.Roots[0] != "~/src" {
		t.Fatalf("unexpected root %q", cfg.Scanner.Roots[0])
	}
	if cfg.UI.Theme != "power-run" {
		t.Fatalf("unexpected theme %q", cfg.UI.Theme)
	}
	if cfg.UI.WindowPlacement != "center" {
		t.Fatalf("unexpected placement %q", cfg.UI.WindowPlacement)
	}
	if cfg.UI.SelectionColor != "#333333" {
		t.Fatalf("unexpected selection color %q", cfg.UI.SelectionColor)
	}
	if !cfg.UI.ClearQueryOnShow {
		t.Fatal("expected clear query on show default")
	}
	if cfg.UI.WorkspaceQueryShortcut != "{" {
		t.Fatalf("unexpected workspace shortcut %q", cfg.UI.WorkspaceQueryShortcut)
	}
	if cfg.Startup.StartHidden != true {
		t.Fatal("expected start hidden default")
	}
	if !cfg.Scanner.Watch {
		t.Fatal("expected scanner watch default")
	}
}

func TestLoadConfigFromJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lyn.json")
	data := []byte(`{"scanner":{"roots":["~/src","D:/work"],"maxDepth":7,"timeout":"5s","watch":false},"hotkey":{"binding":"Ctrl+Shift+K"},"ui":{"theme":"tron-legacy","backgroundOpacity":0.85,"selectionColor":"#20d8ff","windowPlacement":"center","clearQueryOnShow":false,"workspaceQueryShortcut":"["},"startup":{"enabled":true,"startHidden":true}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Path != path {
		t.Fatalf("unexpected path %q", cfg.Path)
	}
	if cfg.Scanner.Roots[1] != "D:/work" {
		t.Fatalf("unexpected root %q", cfg.Scanner.Roots[1])
	}
	if cfg.Hotkey.Binding != "Ctrl+Shift+K" {
		t.Fatalf("unexpected binding %q", cfg.Hotkey.Binding)
	}
	if cfg.UI.Theme != "tron-legacy" {
		t.Fatalf("unexpected theme %q", cfg.UI.Theme)
	}
	if cfg.UI.WindowPlacement != "center" {
		t.Fatalf("unexpected placement %q", cfg.UI.WindowPlacement)
	}
	if cfg.UI.SelectionColor != "#20d8ff" {
		t.Fatalf("unexpected selection color %q", cfg.UI.SelectionColor)
	}
	if cfg.UI.ClearQueryOnShow {
		t.Fatal("expected clear query override")
	}
	if cfg.UI.WorkspaceQueryShortcut != "[" {
		t.Fatalf("unexpected workspace shortcut %q", cfg.UI.WorkspaceQueryShortcut)
	}
	if cfg.Scanner.Watch {
		t.Fatal("expected scanner watch override")
	}
	if !cfg.Startup.Enabled || !cfg.Startup.StartHidden {
		t.Fatalf("unexpected startup config %#v", cfg.Startup)
	}
}

func TestNormalizeConfigUsesSingleWorkspaceShortcutCharacter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UI.WorkspaceQueryShortcut = "  [["
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.UI.WorkspaceQueryShortcut != "[" {
		t.Fatalf("unexpected shortcut %q", normalized.UI.WorkspaceQueryShortcut)
	}
}

func TestNormalizeConfigRejectsUnknownPlacement(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UI.WindowPlacement = "corner"
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.UI.WindowPlacement != "center" {
		t.Fatalf("unexpected placement %q", normalized.UI.WindowPlacement)
	}
}

func TestSaveConfigWritesJSON(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Path = filepath.Join(t.TempDir(), "lyn.json")
	cfg.Scanner.Roots = []string{"D:/work"}
	saved, err := SaveConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(saved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Scanner.Roots[0] != "D:/work" {
		t.Fatalf("unexpected root %q", loaded.Scanner.Roots[0])
	}
}
