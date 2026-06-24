//go:build !windows

package lyn

func MaybeRunHotkeyBroker(args []string) bool {
	return false
}
