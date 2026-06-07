package startup

import (
	"strings"
	"testing"
)

func TestWindowsStartupValueQuotesExecutable(t *testing.T) {
	value := windowsStartupValue(`C:\Program Files\Lyn\lyn.exe`)
	if value != `"C:\Program Files\Lyn\lyn.exe" --start-hidden` {
		t.Fatalf("unexpected value %q", value)
	}
}

func TestLinuxDesktopEntryIncludesExecutable(t *testing.T) {
	entry := linuxDesktopEntry(`/home/me/apps/lyn app`)
	if !strings.Contains(entry, `Exec="/home/me/apps/lyn app"`) {
		t.Fatalf("missing executable in entry %q", entry)
	}
	if !strings.Contains(entry, "X-GNOME-Autostart-enabled=true") {
		t.Fatalf("missing autostart flag %q", entry)
	}
}

func TestDesktopExecValueEscapesQuotes(t *testing.T) {
	value := desktopExecValue(`/home/me/lyn "dev"`)
	if value != `"/home/me/lyn \"dev\""` {
		t.Fatalf("unexpected value %q", value)
	}
}
