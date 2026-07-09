//go:build linux

package main

/*
#cgo pkg-config: gtk4 gtk4-layer-shell-0
#include <stdio.h>
#include <stdlib.h>
#include <gtk/gtk.h>
#include <gtk4-layer-shell.h>
#include <gdk/gdk.h>
#include <cairo.h>

static GtkWindow *main_window = NULL;
static GtkWidget *draw_area = NULL;
static gboolean should_quit = FALSE;
static int screen_width = 1920;
static int screen_height = 1080;

// sprite data
static unsigned char *sprite_pixels = NULL;
static int sprite_width = 0;
static int sprite_height = 0;
static int sprite_stride = 0;
static int src_x = 0, src_y = 0, src_w = 32, src_h = 32;
static int draw_scale = 8;
static int flip_x = 0;
static int draw_x = 0, draw_y = 0; // cat position within the full-screen surface

// pointer state, fed from GTK event controllers (Wayland gives us no global
// query, so we listen to the cat window's own pointer events instead)
static int ptr_inside = 0;
static int ptr_button = 0; // 1=left, 3=right, 0=none held
static double ptr_x = 0, ptr_y = 0;

static void on_motion(GtkEventControllerMotion *c, double x, double y, gpointer d) {
    ptr_x = x;
    ptr_y = y;
    ptr_inside = 1;
}
static void on_leave(GtkEventControllerMotion *c, gpointer d) {
    ptr_inside = 0;
}
static void on_pressed(GtkGestureClick *g, int n, double x, double y, gpointer d) {
    ptr_button = gtk_gesture_single_get_current_button(GTK_GESTURE_SINGLE(g));
    ptr_x = x;
    ptr_y = y;
}
static void on_released(GtkGestureClick *g, int n, double x, double y, gpointer d) {
    ptr_button = 0;
}

static void on_draw(GtkDrawingArea *area, cairo_t *cr, int width, int height, gpointer data) {
    cairo_set_operator(cr, CAIRO_OPERATOR_SOURCE);
    cairo_set_source_rgba(cr, 0, 0, 0, 0);
    cairo_paint(cr);

    if (!sprite_pixels) return;

    cairo_surface_t *surface = cairo_image_surface_create_for_data(
        sprite_pixels,
        CAIRO_FORMAT_ARGB32,
        sprite_width,
        sprite_height,
        sprite_stride
    );

    cairo_set_operator(cr, CAIRO_OPERATOR_OVER);

    // draw the cat at its position within the (stationary, full-screen) surface
    cairo_save(cr);
    cairo_translate(cr, draw_x, draw_y);
    if (flip_x) {
        cairo_translate(cr, src_w * draw_scale, 0);
        cairo_scale(cr, -draw_scale, draw_scale);
    } else {
        cairo_scale(cr, draw_scale, draw_scale);
    }

    cairo_set_source_surface(cr, surface, -src_x, -src_y);
    cairo_pattern_set_filter(cairo_get_source(cr), CAIRO_FILTER_NEAREST);
    cairo_rectangle(cr, 0, 0, src_w, src_h);
    cairo_fill(cr);
    cairo_restore(cr);

    cairo_surface_destroy(surface);
}

static gboolean on_close(GtkWindow *window, gpointer data) {
    should_quit = TRUE;
    return FALSE;
}

// on_realize intentionally left minimal: the window now accepts pointer input
// over its (cat-sized) bounds so it can be dragged and clicked. It's no longer
// fully click-through — clicks inside the cat's box land on the cat.

void wayland_create_window(int width, int height) {
    main_window = GTK_WINDOW(gtk_window_new());
    gtk_window_set_title(main_window, "goob - lil vro");

    gtk_layer_init_for_window(main_window);
    gtk_layer_set_layer(main_window, GTK_LAYER_SHELL_LAYER_OVERLAY);
    // full-screen: anchor all four edges so the surface covers the whole output
    // and never moves — the cat is drawn at an offset inside it instead.
    gtk_layer_set_anchor(main_window, GTK_LAYER_SHELL_EDGE_LEFT, TRUE);
    gtk_layer_set_anchor(main_window, GTK_LAYER_SHELL_EDGE_TOP, TRUE);
    gtk_layer_set_anchor(main_window, GTK_LAYER_SHELL_EDGE_RIGHT, TRUE);
    gtk_layer_set_anchor(main_window, GTK_LAYER_SHELL_EDGE_BOTTOM, TRUE);
    gtk_layer_set_keyboard_mode(main_window, GTK_LAYER_SHELL_KEYBOARD_MODE_NONE);
    gtk_layer_set_exclusive_zone(main_window, -1);

    GtkCssProvider *css = gtk_css_provider_new();
    gtk_css_provider_load_from_string(css, "window, * { background: transparent; background-color: transparent; }");
    gtk_style_context_add_provider_for_display(
        gdk_display_get_default(),
        GTK_STYLE_PROVIDER(css),
        GTK_STYLE_PROVIDER_PRIORITY_APPLICATION
    );

    draw_area = gtk_drawing_area_new();
    gtk_drawing_area_set_draw_func(GTK_DRAWING_AREA(draw_area), on_draw, NULL, NULL);
    gtk_widget_set_size_request(draw_area, screen_width, screen_height);
    gtk_widget_set_hexpand(draw_area, TRUE);
    gtk_widget_set_vexpand(draw_area, TRUE);
    gtk_window_set_child(main_window, draw_area);

    // pointer input: motion for drag tracking, click for grab/release + scare
    GtkEventController *motion = gtk_event_controller_motion_new();
    g_signal_connect(motion, "motion", G_CALLBACK(on_motion), NULL);
    g_signal_connect(motion, "leave", G_CALLBACK(on_leave), NULL);
    gtk_widget_add_controller(draw_area, motion);

    GtkGesture *click = gtk_gesture_click_new();
    gtk_gesture_single_set_button(GTK_GESTURE_SINGLE(click), 0); // any button
    g_signal_connect(click, "pressed", G_CALLBACK(on_pressed), NULL);
    g_signal_connect(click, "released", G_CALLBACK(on_released), NULL);
    gtk_widget_add_controller(draw_area, GTK_EVENT_CONTROLLER(click));

    g_signal_connect(main_window, "close-request", G_CALLBACK(on_close), NULL);

    gtk_window_present(main_window);
}

void wayland_set_sprite(unsigned char *pixels, int w, int h, int stride) {
    sprite_pixels = pixels;
    sprite_width = w;
    sprite_height = h;
    sprite_stride = stride;
}

void wayland_set_frame(int x, int y, int w, int h, int scale, int flip) {
    src_x = x;
    src_y = y;
    src_w = w;
    src_h = h;
    draw_scale = scale;
    flip_x = flip;
}

void wayland_set_position(int x, int y) {
    if (!main_window) return;
    draw_x = x;
    draw_y = y;
    // Keep the surface itself full-screen but clip pointer input to just the
    // cat's rectangle, so every other pixel stays click-through to the desktop.
    GdkSurface *s = gtk_native_get_surface(GTK_NATIVE(main_window));
    if (s) {
        cairo_rectangle_int_t r = { x, y, src_w * draw_scale, src_h * draw_scale };
        cairo_region_t *region = cairo_region_create_rectangle(&r);
        gdk_surface_set_input_region(s, region);
        cairo_region_destroy(region);
    }
}

// wayland_pointer reports the pointer in absolute screen coords (the surface is
// full-screen and stationary). Returns 1 when the pointer is over the cat.
int wayland_pointer(int *button, int *px, int *py) {
    *button = ptr_button;
    *px = (int)ptr_x;
    *py = (int)ptr_y;
    return ptr_inside;
}

void wayland_set_size(int w, int h) {
    if (draw_area) gtk_widget_set_size_request(draw_area, w, h);
    if (main_window) gtk_window_set_default_size(main_window, w, h);
}

void wayland_queue_draw() {
    if (draw_area) gtk_widget_queue_draw(draw_area);
}

int wayland_iterate() {
    return g_main_context_iteration(NULL, FALSE);
}

int wayland_should_quit() { return should_quit ? 1 : 0; }
int wayland_screen_width() { return screen_width; }
int wayland_screen_height() { return screen_height; }

void wayland_init_display() {
    GdkDisplay *display = gdk_display_get_default();
    if (display) {
        GListModel *monitors = gdk_display_get_monitors(display);
        if (g_list_model_get_n_items(monitors) > 0) {
            GdkMonitor *mon = g_list_model_get_item(monitors, 0);
            GdkRectangle geom;
            gdk_monitor_get_geometry(mon, &geom);
            screen_width = geom.width;
            screen_height = geom.height;
            g_object_unref(mon);
        }
    }
}
*/
import "C"

import (
	"encoding/json"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/NovusEdge/goob/internal/pet"
	"github.com/NovusEdge/goob/internal/sprite"
	"github.com/NovusEdge/goob/internal/sysmon"
)

type manifest struct {
	Sheet     string          `json:"sheet"`
	FrameSize [2]int          `json:"frameSize"`
	States    map[string]anim `json:"states"`
}

type anim struct {
	Row    int `json:"row"`
	Frames int `json:"frames"`
	FPS    int `json:"fps"`
}

func runWayland(manifestPath string, scale int, newPet func(int, int, int, int) *pet.Pet) {
	C.gtk_init()
	C.wayland_init_display()

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatal(err)
	}

	imgPath := filepath.Join(filepath.Dir(manifestPath), m.Sheet)
	imgFile, err := os.Open(imgPath)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(imgFile)
	imgFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			i := (y*w + x) * 4
			pixels[i+0] = byte(b >> 8)
			pixels[i+1] = byte(g >> 8)
			pixels[i+2] = byte(r >> 8)
			pixels[i+3] = byte(a >> 8)
		}
	}

	frameW, frameH := m.FrameSize[0], m.FrameSize[1]
	scaledW, scaledH := frameW*scale, frameH*scale

	C.wayland_create_window(C.int(scaledW), C.int(scaledH))

	C.wayland_set_sprite(
		(*C.uchar)(unsafe.Pointer(&pixels[0])),
		C.int(w), C.int(h), C.int(w*4),
	)

	screenW := int(C.wayland_screen_width())
	screenH := int(C.wayland_screen_height())

	p := newPet(screenW, screenH, scaledW, scaledH)
	has := func(n string) bool { _, ok := m.States[n]; return ok }
	p.SetLoopFn(func(n string) int {
		a := m.States[sprite.Resolve(n, has)]
		if a.FPS <= 0 {
			return 0
		}
		return a.Frames * (60 / a.FPS)
	})

	currentAnim := "idle"
	animFrame := 0
	animTick := 0

	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	grabbing := false
	grabLX, grabLY := 0, 0
	for C.wayland_should_quit() == 0 {
		<-ticker.C

		if frame%120 == 0 { // ~every 2s: sample the machine's mood
			p.SetMood(moodFrom(sysmon.Read()))
		}
		frame++

		for C.wayland_iterate() != 0 {
		}

		// The full-screen stationary surface gives us absolute screen coords from
		// GTK directly — no X11, no feedback. ptr is valid while over the cat or
		// while a button is held (implicit grab keeps events flowing during drag).
		var btn, px, py C.int
		inside := C.wayland_pointer(&btn, &px, &py) != 0
		cursorX, cursorY := -1, -1
		if inside || int(btn) != 0 {
			cursorX, cursorY = int(px), int(py)
		}

		switch {
		case int(btn) == 1: // left held: drag
			if !grabbing {
				grabbing = true
				grabLX, grabLY = cursorX-p.X, cursorY-p.Y // pin where you grabbed
			}
			p.Hold(cursorX-grabLX+scaledW/2, cursorY-grabLY+scaledH/2)
		case p.Held(): // released while carrying -> drop
			grabbing = false
			p.Release()
		case int(btn) == 3: // right: startle
			grabbing = false
			p.Scare()
		default:
			grabbing = false
		}

		p.Update(cursorX, cursorY)

		newAnim := p.Anim()
		if newAnim != currentAnim {
			currentAnim = newAnim
			animFrame = 0
			animTick = 0
		}

		a := m.States[sprite.Resolve(currentAnim, has)]

		animTick++
		if a.FPS > 0 && animTick >= 60/a.FPS {
			animTick = 0
			animFrame = (animFrame + 1) % a.Frames
		}

		srcX := animFrame * frameW
		srcY := a.Row * frameH
		flip := 0
		if p.FacingLeft {
			flip = 1
		}
		C.wayland_set_frame(C.int(srcX), C.int(srcY), C.int(frameW), C.int(frameH), C.int(scale), C.int(flip))

		C.wayland_set_position(C.int(p.X), C.int(p.Y))
		C.wayland_queue_draw()
	}
}
