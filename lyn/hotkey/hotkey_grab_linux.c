#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <sys/select.h>
#include <errno.h>
#include <stdint.h>
#include "_cgo_export.h"

void lynRunHotkey(uintptr_t handle, int keysym, unsigned int baseMod, unsigned int *grabMods, int nMods, int stopFD) {
	Display *d = XOpenDisplay(0);
	if (d == NULL) {
		return;
	}

	Window root = DefaultRootWindow(d);
	KeyCode keycode = XKeysymToKeycode(d, keysym);
	if (keycode == 0) {
		XCloseDisplay(d);
		return;
	}

	for (int i = 0; i < nMods; i++) {
		XGrabKey(d, keycode, grabMods[i], root, False, GrabModeAsync, GrabModeAsync);
	}
	XSync(d, False);

	int xfd = ConnectionNumber(d);
	unsigned int relevant = ShiftMask | ControlMask | Mod1Mask | Mod3Mask | Mod4Mask | Mod5Mask;

	for (;;) {
		fd_set fds;
		FD_ZERO(&fds);
		FD_SET(xfd, &fds);
		FD_SET(stopFD, &fds);
		int maxfd = xfd > stopFD ? xfd : stopFD;

		int n = select(maxfd + 1, &fds, NULL, NULL, NULL);
		if (n < 0) {
			if (errno == EINTR) {
				continue;
			}
			break;
		}
		if (FD_ISSET(stopFD, &fds)) {
			break;
		}

		while (XPending(d) > 0) {
			XEvent ev;
			XNextEvent(d, &ev);
			if (ev.type == KeyPress &&
				ev.xkey.keycode == keycode &&
				(ev.xkey.state & relevant) == baseMod) {
				goHotkeyFire(handle);
			}
		}
	}

	for (int i = 0; i < nMods; i++) {
		XUngrabKey(d, keycode, grabMods[i], root);
	}
	XCloseDisplay(d);
}
