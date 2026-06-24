//go:build windows

package lyn

import "lyn.tools/launcher/lyn/hotkey"

func MaybeRunHotkeyBroker(args []string) bool {
	isBroker, parent := hotkey.ParseBrokerArgs(args)
	if !isBroker {
		return false
	}
	debug := NewDebugLogger(args)
	defer debug.Close()
	debug.Log("broker.start", "parent", parent)
	if err := hotkey.RunBroker(parent); err != nil {
		debug.Log("broker.error", "error", err)
	}
	debug.Log("broker.stop")
	return true
}
