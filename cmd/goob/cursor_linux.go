//go:build linux

package main

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>

static void get_cursor_pos(int *x, int *y) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        *x = -1;
        *y = -1;
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
}
*/
import "C"

func getGlobalCursor() (int, int) {
	var x, y C.int
	C.get_cursor_pos(&x, &y)
	return int(x), int(y)
}
