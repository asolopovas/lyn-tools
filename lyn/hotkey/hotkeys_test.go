package hotkey

import "testing"

func TestParseHotkeySupportsRecordedCombos(t *testing.T) {
	for _, value := range []string{"Ctrl+Space", "Ctrl+Shift+K", "Alt+1", "Win+F12"} {
		if _, err := ParseHotkey(value); err != nil {
			t.Fatalf("expected %q to parse: %v", value, err)
		}
	}
}
