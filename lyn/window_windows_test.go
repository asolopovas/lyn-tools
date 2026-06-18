//go:build windows

package lyn

import "testing"

func TestTaskbarHiddenExStyle(t *testing.T) {
	style := uintptr(wsExAppwindow)
	updated := taskbarHiddenExStyle(style)
	if updated&wsExAppwindow != 0 {
		t.Fatal("expected app-window taskbar style to be removed")
	}
	if updated&wsExToolwindow == 0 {
		t.Fatal("expected tool-window style to be added")
	}
}

func TestTaskbarHiddenExStylePreservesOtherFlags(t *testing.T) {
	const otherFlag = uintptr(0x20)
	updated := taskbarHiddenExStyle(uintptr(wsExAppwindow) | otherFlag)
	if updated&otherFlag == 0 {
		t.Fatal("expected unrelated extended styles to be preserved")
	}
}

func TestCloseToTrayShouldHide(t *testing.T) {
	originalQuitting := closeToTrayQuitting
	originalHide := closeToTrayHide
	defer func() {
		closeToTrayQuitting = originalQuitting
		closeToTrayHide = originalHide
	}()
	closeToTrayHide = func() {}
	quitting := false
	closeToTrayQuitting = func() bool { return quitting }

	cases := []struct {
		name   string
		msg    uintptr
		wparam uintptr
		quit   bool
		want   bool
	}{
		{name: "win+q sc_close hides", msg: wmSyscommand, wparam: scClose, want: true},
		{name: "sc_close with low bits hides", msg: wmSyscommand, wparam: scClose | 0x07, want: true},
		{name: "wm_close hides", msg: wmClose, want: true},
		{name: "other syscommand forwards", msg: wmSyscommand, wparam: 0xF120, want: false},
		{name: "unrelated message forwards", msg: 0x0200, want: false},
		{name: "quitting allows close", msg: wmClose, quit: true, want: false},
		{name: "quitting allows sc_close", msg: wmSyscommand, wparam: scClose, quit: true, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quitting = tc.quit
			if got := closeToTrayShouldHide(tc.msg, tc.wparam); got != tc.want {
				t.Fatalf("closeToTrayShouldHide(%#x, %#x) = %v, want %v", tc.msg, tc.wparam, got, tc.want)
			}
		})
	}
}

func TestPointInWindowRect(t *testing.T) {
	rect := windowsRect{Left: 10, Top: 20, Right: 30, Bottom: 40}
	inside := []windowsPoint{{X: 10, Y: 20}, {X: 29, Y: 39}, {X: 20, Y: 30}}
	for _, point := range inside {
		if !pointInWindowRect(point, rect) {
			t.Fatalf("expected point %+v inside rect %+v", point, rect)
		}
	}
	outside := []windowsPoint{{X: 9, Y: 20}, {X: 30, Y: 20}, {X: 10, Y: 19}, {X: 10, Y: 40}}
	for _, point := range outside {
		if pointInWindowRect(point, rect) {
			t.Fatalf("expected point %+v outside rect %+v", point, rect)
		}
	}
}
