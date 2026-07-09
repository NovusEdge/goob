//go:build linux

package main

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>

static void get_cursor_pos(int *x, int *y, int *buttons) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        *x = -1;
        *y = -1;
        *buttons = 0;
        return;
    }

    Window root = DefaultRootWindow(dpy);
    Window child;
    int root_x, root_y, win_x, win_y;
    unsigned int mask;

    XQueryPointer(dpy, root, &root, &child, &root_x, &root_y, &win_x, &win_y, &mask);
    XCloseDisplay(dpy);

    *x = root_x;
    *y = root_y;
    *buttons = 0;
    if (mask & Button1Mask) *buttons |= 1; // left
    if (mask & Button3Mask) *buttons |= 2; // right
}
*/
import "C"

// getGlobalCursor returns pointer position and a button bitmask (1=left, 2=right).
func getGlobalCursor() (x, y, buttons int) {
	var cx, cy, cb C.int
	C.get_cursor_pos(&cx, &cy, &cb)
	return int(cx), int(cy), int(cb)
}
