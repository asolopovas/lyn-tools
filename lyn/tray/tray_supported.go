//go:build windows || (linux && cgo)

package tray

import (
	"embed"
	"runtime"
	"sync"

	"github.com/getlantern/systray"
)

//go:embed tray.ico tray.png
var trayAssets embed.FS

var trayOnce sync.Once

func init() {
	startTray = startSupportedTray
	stopTray = systray.Quit
}

func startSupportedTray(controller Controller) {
	trayOnce.Do(func() {
		go systray.Run(func() {
			systray.SetIcon(trayIcon())
			systray.SetTitle("Lyn")
			systray.SetTooltip("Lyn project launcher")
			configureMenu(controller)
		}, func() {})
	})
}

func configureMenu(controller Controller) {
	show := systray.AddMenuItem("Show", "Show Lyn")
	settings := systray.AddMenuItem("Settings", "Open Lyn settings")
	restart := systray.AddMenuItem("Restart", "Restart Lyn")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Quit Lyn")
	go func() {
		for range show.ClickedCh {
			controller.Show()
		}
	}()
	go func() {
		for range settings.ClickedCh {
			controller.ShowSettings()
		}
	}()
	go func() {
		for range restart.ClickedCh {
			controller.Restart()
		}
	}()
	go func() {
		for range quit.ClickedCh {
			controller.Quit()
			systray.Quit()
		}
	}()
}

func trayIcon() []byte {
	name := "tray.png"
	if runtime.GOOS == "windows" {
		name = "tray.ico"
	}
	icon, _ := trayAssets.ReadFile(name)
	return icon
}
