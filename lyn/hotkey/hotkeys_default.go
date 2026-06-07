//go:build linux || darwin

package hotkey

import nativehotkey "golang.design/x/hotkey"

func registerHotkeyBinding(binding Binding, onPress func()) (Registration, error) {
	hk := nativehotkey.New(binding.Modifiers, binding.Key)
	if err := hk.Register(); err != nil {
		return nil, err
	}
	go func() {
		for range hk.Keydown() {
			onPress()
		}
	}()
	return hk, nil
}
