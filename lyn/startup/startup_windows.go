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
	if machineStartupRegistered() {
		return disableWindowsStartup()
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsStartupRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(startupName, windowsStartupValue(exe))
}

func machineStartupRegistered() bool {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, windowsStartupRunKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(startupName)
	return err == nil
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
