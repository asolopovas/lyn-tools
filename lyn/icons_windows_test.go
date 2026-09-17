//go:build windows

package lyn

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestResolveWindowsAppIconUsesShellNamespace(t *testing.T) {
	path := `shell:AppsFolder\windows.immersivecontrolpanel_cw5n1h2txyewy!microsoft.windows.immersivecontrolpanel`
	cacheDir := t.TempDir()
	if err := os.WriteFile(iconCachePath(cacheDir, path), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	icon, err := resolveWindowsAppIcon(t.Context(), cacheDir, path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, ok := strings.CutPrefix(icon, "data:image/png;base64,")
	if !ok {
		t.Fatalf("expected packaged app icon, got %q", icon)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("decode packaged app icon: %v", err)
	}
}
