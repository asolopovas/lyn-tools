package main

import (
	"log"
	"os"
	"slices"

	"lyn.tools/launcher/lyn"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func main() {
	debug := lyn.NewDebugLogger(os.Args[1:])
	initialConfig, configErr := lyn.LoadConfig("")
	mode := windowModeFromArgs(os.Args[1:])
	if mode != lyn.SettingsWindowMode && startHiddenArg(os.Args[1:]) {
		initialConfig.Startup.Enabled = true
		initialConfig.Startup.StartHidden = true
	}
	app := lyn.NewApp(debug)
	app.SetWindowMode(mode)
	if configErr == nil {
		app.UseConfig(initialConfig)
	} else {
		debug.Log("config.initial.error", "error", configErr)
	}
	err := wails.Run(newWailsOptions(app))
	if err != nil {
		debug.Log("wails.error", "error", err)
		debug.Close()
		log.Fatal(err)
	}
}

func newWailsOptions(app *lyn.App) *options.App {
	appOptions := &options.App{
		Title:            "Lyn",
		Width:            640,
		Height:           306,
		DisableResize:    true,
		Frameless:        true,
		CSSDragProperty:  "--wails-draggable",
		CSSDragValue:     "drag",
		BackgroundColour: &options.RGBA{R: 32, G: 32, B: 32, A: 0},
		StartHidden:      app.StartHidden(),
		AlwaysOnTop:      true,
		AssetServer:      &assetserver.Options{Assets: frontendAssets()},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			DisableWindowIcon:    true,
			WindowClassName:      lyn.NativeWindowClassName,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind:       []any{app},
	}
	if app.WindowMode() == string(lyn.SettingsWindowMode) {
		appOptions.Title = "Lyn Settings"
		appOptions.Width = 640
		appOptions.Height = 720
		appOptions.MinWidth = 560
		appOptions.MinHeight = 560
		appOptions.DisableResize = false
		appOptions.AlwaysOnTop = false
		appOptions.BackgroundColour = &options.RGBA{R: 19, G: 19, B: 19, A: 255}
		appOptions.Windows.WebviewIsTransparent = false
		appOptions.Windows.WindowIsTranslucent = false
		appOptions.Windows.WindowClassName = lyn.NativeSettingsWindowClassName
	}
	return appOptions
}

func startHiddenArg(args []string) bool {
	return slices.Contains(args, "--start-hidden")
}

func windowModeFromArgs(args []string) lyn.WindowMode {
	if slices.Contains(args, "--settings-window") {
		return lyn.SettingsWindowMode
	}
	return lyn.LauncherWindowMode
}
