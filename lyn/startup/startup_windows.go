//go:build windows

package startup

import "golang.org/x/sys/windows/registry"

const windowsStartupRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const startupName = "Lyn"

func init() {
	configureStartup = configureWindowsStartup
}

func configureWindowsStartup(enabled bool, exe string) error {
	if enabled {
		return enableWindowsStartup(exe)
	}
	return disableWindowsStartup()
}

func enableWindowsStartup(exe string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsStartupRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(startupName, windowsStartupValue(exe))
}

func disableWindowsStartup() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsStartupRunKey, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer key.Close()
	err = key.DeleteValue(startupName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
