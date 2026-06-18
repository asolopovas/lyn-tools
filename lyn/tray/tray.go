package tray

type Controller interface {
	Show()
	ShowSettings()
	Restart()
	Quit()
}

type LogFunc func(event string, fields ...any)

var (
	startTray func(Controller, LogFunc)
	stopTray  func()
)

func Start(controller Controller, logf LogFunc) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if startTray != nil {
		startTray(controller, logf)
	}
}

func Stop() {
	if stopTray != nil {
		stopTray()
	}
}
