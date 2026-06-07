//go:build linux

package hotkey

import nativehotkey "golang.design/x/hotkey"

func linuxKey(value uint16) nativehotkey.Key {
	return nativehotkey.Key(value)
}

var modifiers = map[string]nativehotkey.Modifier{
	"alt":     nativehotkey.Modifier(1 << 3),
	"ctrl":    nativehotkey.Modifier(1 << 2),
	"control": nativehotkey.Modifier(1 << 2),
	"shift":   nativehotkey.Modifier(1 << 0),
	"super":   nativehotkey.Modifier(1 << 6),
	"win":     nativehotkey.Modifier(1 << 6),
}

var keys = map[string]nativehotkey.Key{
	"space":  nativehotkey.Key(0x0020),
	"enter":  linuxKey(0xff0d),
	"return": linuxKey(0xff0d),
	"0":      nativehotkey.Key('0'),
	"1":      nativehotkey.Key('1'),
	"2":      nativehotkey.Key('2'),
	"3":      nativehotkey.Key('3'),
	"4":      nativehotkey.Key('4'),
	"5":      nativehotkey.Key('5'),
	"6":      nativehotkey.Key('6'),
	"7":      nativehotkey.Key('7'),
	"8":      nativehotkey.Key('8'),
	"9":      nativehotkey.Key('9'),
	"a":      nativehotkey.Key('a'),
	"b":      nativehotkey.Key('b'),
	"c":      nativehotkey.Key('c'),
	"d":      nativehotkey.Key('d'),
	"e":      nativehotkey.Key('e'),
	"f":      nativehotkey.Key('f'),
	"g":      nativehotkey.Key('g'),
	"h":      nativehotkey.Key('h'),
	"i":      nativehotkey.Key('i'),
	"j":      nativehotkey.Key('j'),
	"k":      nativehotkey.Key('k'),
	"l":      nativehotkey.Key('l'),
	"m":      nativehotkey.Key('m'),
	"n":      nativehotkey.Key('n'),
	"o":      nativehotkey.Key('o'),
	"p":      nativehotkey.Key('p'),
	"q":      nativehotkey.Key('q'),
	"r":      nativehotkey.Key('r'),
	"s":      nativehotkey.Key('s'),
	"t":      nativehotkey.Key('t'),
	"u":      nativehotkey.Key('u'),
	"v":      nativehotkey.Key('v'),
	"w":      nativehotkey.Key('w'),
	"x":      nativehotkey.Key('x'),
	"y":      nativehotkey.Key('y'),
	"z":      nativehotkey.Key('z'),
	"f1":     linuxKey(0xffbe),
	"f2":     linuxKey(0xffbf),
	"f3":     linuxKey(0xffc0),
	"f4":     linuxKey(0xffc1),
	"f5":     linuxKey(0xffc2),
	"f6":     linuxKey(0xffc3),
	"f7":     linuxKey(0xffc4),
	"f8":     linuxKey(0xffc5),
	"f9":     linuxKey(0xffc6),
	"f10":    linuxKey(0xffc7),
	"f11":    linuxKey(0xffc8),
	"f12":    linuxKey(0xffc9),
}
