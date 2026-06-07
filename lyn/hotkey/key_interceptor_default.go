//go:build !windows

package hotkey

func SetKeyInterceptor(func(uint32) bool) func() {
	return func() {}
}
