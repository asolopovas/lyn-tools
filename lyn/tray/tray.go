package tray

type Controller interface {
	Show()
	ShowSettings()
	Restart()
	Quit()
}

var (
	startTray func(Controller)
	stopTray  func()
)

func Start(controller Controller) {
	if startTray != nil {
		startTray(controller)
	}
}

func Stop() {
	if stopTray != nil {
		stopTray()
	}
}
