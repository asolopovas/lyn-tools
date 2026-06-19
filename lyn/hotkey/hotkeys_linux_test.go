//go:build linux

package hotkey

import (
	"reflect"
	"testing"

	nativehotkey "golang.design/x/hotkey"
)

func TestHotkeyLockVariantsCoverNumAndCapsLock(t *testing.T) {
	got := hotkeyLockVariants([]nativehotkey.Modifier{nativehotkey.Mod4})
	want := [][]nativehotkey.Modifier{
		{nativehotkey.Mod4},
		{nativehotkey.Mod4, nativehotkey.Mod2},
		{nativehotkey.Mod4, capsLockMask},
		{nativehotkey.Mod4, nativehotkey.Mod2, capsLockMask},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lock variants %#v, want %#v", got, want)
	}
}

func TestHotkeyLockVariantsDoesNotMutateBase(t *testing.T) {
	base := []nativehotkey.Modifier{nativehotkey.Mod4}
	hotkeyLockVariants(base)
	if len(base) != 1 || base[0] != nativehotkey.Mod4 {
		t.Fatalf("base modifiers were mutated: %#v", base)
	}
}
