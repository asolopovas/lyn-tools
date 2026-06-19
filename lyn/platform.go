package lyn

import "runtime"

func (a *App) Platform() string {
	return runtime.GOOS
}
