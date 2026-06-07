package hotkey

import (
	"fmt"
	"strings"

	nativehotkey "golang.design/x/hotkey"
)

type Binding struct {
	Modifiers []nativehotkey.Modifier
	Key       nativehotkey.Key
}

func ParseHotkey(input string) (Binding, error) {
	parts := strings.Split(input, "+")
	if len(parts) < 2 {
		return Binding{}, fmt.Errorf("hotkey must include modifiers and key")
	}
	binding := Binding{}
	for i, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			return Binding{}, fmt.Errorf("empty hotkey segment")
		}
		if i == len(parts)-1 {
			key, ok := keys[name]
			if !ok {
				return Binding{}, fmt.Errorf("unsupported key %q", part)
			}
			binding.Key = key
			continue
		}
		mod, ok := modifiers[name]
		if !ok {
			return Binding{}, fmt.Errorf("unsupported modifier %q", part)
		}
		binding.Modifiers = append(binding.Modifiers, mod)
	}
	return binding, nil
}

type Registration interface {
	Unregister() error
}

func Register(input string, onPress func()) (Registration, error) {
	binding, err := ParseHotkey(input)
	if err != nil {
		return nil, err
	}
	return registerHotkeyBinding(binding, onPress)
}
