//go:build windows

package lyn

import (
	"strings"
	"testing"
)

func TestResolveWindowsAppIconUsesShellNamespace(t *testing.T) {
	path := `shell:AppsFolder\windows.immersivecontrolpanel_cw5n1h2txyewy!microsoft.windows.immersivecontrolpanel`
	icon, err := resolveWindowsAppIcon(t.Context(), t.TempDir(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(icon, "data:image/png;base64,") {
		t.Fatalf("expected packaged app icon, got %q", icon)
	}
}
