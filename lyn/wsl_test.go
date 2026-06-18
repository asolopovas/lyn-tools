package lyn

import "testing"

func TestWSLUnixFromUNC(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantDistro string
		wantPath   string
		wantOK     bool
	}{
		{name: "localhost project", input: `\\wsl.localhost\Ubuntu\home\me\src\app`, wantDistro: "Ubuntu", wantPath: "/home/me/src/app", wantOK: true},
		{name: "legacy prefix", input: `\\wsl$\Ubuntu\home\me\src`, wantDistro: "Ubuntu", wantPath: "/home/me/src", wantOK: true},
		{name: "distro root", input: `\\wsl.localhost\Ubuntu`, wantDistro: "Ubuntu", wantPath: "/", wantOK: true},
		{name: "trailing slash", input: `\\wsl.localhost\Ubuntu\home\me\`, wantDistro: "Ubuntu", wantPath: "/home/me", wantOK: true},
		{name: "windows path", input: `C:\Users\me\src`, wantOK: false},
		{name: "unix path", input: "/home/me/src", wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			distro, path, ok := wslUnixFromUNC(tc.input)
			if ok != tc.wantOK || distro != tc.wantDistro || path != tc.wantPath {
				t.Fatalf("wslUnixFromUNC(%q) = (%q,%q,%v), want (%q,%q,%v)", tc.input, distro, path, ok, tc.wantDistro, tc.wantPath, tc.wantOK)
			}
		})
	}
}

func TestWSLWindowsRootRoundTrip(t *testing.T) {
	cases := []struct {
		distro string
		unix   string
		want   string
	}{
		{distro: "Ubuntu", unix: "/home/me/src", want: `\\wsl.localhost\Ubuntu\home\me\src`},
		{distro: "Ubuntu", unix: "/", want: `\\wsl.localhost\Ubuntu`},
	}
	for _, tc := range cases {
		got := wslWindowsRoot(tc.distro, tc.unix)
		if got != tc.want {
			t.Fatalf("wslWindowsRoot(%q,%q) = %q, want %q", tc.distro, tc.unix, got, tc.want)
		}
		distro, unix, ok := wslUnixFromUNC(got)
		if !ok || distro != tc.distro || unix != tc.unix {
			t.Fatalf("round trip of %q = (%q,%q,%v)", got, distro, unix, ok)
		}
	}
}

func TestIsWSLSystemDistro(t *testing.T) {
	for _, name := range []string{"docker-desktop", "Docker-Desktop", "rancher-desktop"} {
		if !isWSLSystemDistro(name) {
			t.Fatalf("expected %q to be a system distro", name)
		}
	}
	if isWSLSystemDistro("Ubuntu") {
		t.Fatal("Ubuntu must not be treated as a system distro")
	}
}
