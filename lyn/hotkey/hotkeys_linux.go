//go:build linux

package hotkey

import nativehotkey "golang.design/x/hotkey"

const capsLockMask = nativehotkey.Modifier(1 << 1)

func hotkeyLockVariants(base []nativehotkey.Modifier) [][]nativehotkey.Modifier {
	extras := [][]nativehotkey.Modifier{
		nil,
		{nativehotkey.Mod2},
		{capsLockMask},
		{nativehotkey.Mod2, capsLockMask},
	}
	variants := make([][]nativehotkey.Modifier, 0, len(extras))
	for _, extra := range extras {
		mods := append(append([]nativehotkey.Modifier{}, base...), extra...)
		variants = append(variants, mods)
	}
	return variants
}

type multiRegistration []*nativehotkey.Hotkey

func (m multiRegistration) Unregister() error {
	var firstErr error
	for _, hk := range m {
		if err := hk.Unregister(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func registerHotkeyBinding(binding Binding, onPress func()) (Registration, error) {
	registered := make(multiRegistration, 0, 4)
	for _, mods := range hotkeyLockVariants(binding.Modifiers) {
		hk := nativehotkey.New(mods, binding.Key)
		if err := hk.Register(); err != nil {
			_ = registered.Unregister()
			return nil, err
		}
		registered = append(registered, hk)
		go func() {
			for range hk.Keydown() {
				onPress()
			}
		}()
	}
	return registered, nil
}
