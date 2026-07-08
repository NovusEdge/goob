//go:build linux

package main

/*
#cgo pkg-config: gtk4 gtk4-layer-shell-0
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

    if (flip_x) {
        cairo_translate(cr, width, 0);
        cairo_scale(cr, -draw_scale, draw_scale);
    } else {
        cairo_scale(cr, draw_scale, draw_scale);
    }

    cairo_set_source_surface(cr, surface, -src_x, -src_y);
    cairo_pattern_set_filter(cairo_get_source(cr), CAIRO_FILTER_NEAREST);
    cairo_rectangle(cr, 0, 0, src_w, src_h);
    cairo_fill(cr);

    cairo_surface_destroy(surface);
}

static gboolean on_close(GtkWindow *window, gpointer data) {
    should_quit = TRUE;
    return FALSE;
}

static void on_realize(GtkWidget *widget, gpointer data) {
    // try to make click-through by setting empty input region
    GdkSurface *surface = gtk_native_get_surface(GTK_NATIVE(widget));
    if (surface) {
        cairo_region_t *region = cairo_region_create();
        gdk_surface_set_input_region(surface, region);
        cairo_region_destroy(region);
    }
}

void wayland_create_window(int width, int height) {
    main_window = GTK_WINDOW(gtk_window_new());
    gtk_window_set_title(main_window, "goob - lil vro");
    gtk_window_set_default_size(main_window, width, height);

    gtk_layer_init_for_window(main_window);
    gtk_layer_set_layer(main_window, GTK_LAYER_SHELL_LAYER_OVERLAY);
    gtk_layer_set_anchor(main_window, GTK_LAYER_SHELL_EDGE_LEFT, TRUE);
    gtk_layer_set_anchor(main_window, GTK_LAYER_SHELL_EDGE_TOP, TRUE);
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
    gtk_widget_set_size_request(draw_area, width, height);
    gtk_window_set_child(main_window, draw_area);

    g_signal_connect(main_window, "close-request", G_CALLBACK(on_close), NULL);
    g_signal_connect(main_window, "realize", G_CALLBACK(on_realize), NULL);

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
    gtk_layer_set_margin(main_window, GTK_LAYER_SHELL_EDGE_LEFT, x);
    gtk_layer_set_margin(main_window, GTK_LAYER_SHELL_EDGE_TOP, y);
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

	currentAnim := "idle"
	animFrame := 0
	animTick := 0

	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	for C.wayland_should_quit() == 0 {
		<-ticker.C

		for C.wayland_iterate() != 0 {
		}

		cursorX, cursorY := getGlobalCursor()
		p.Update(cursorX, cursorY)

		newAnim := p.Anim()
		if newAnim != currentAnim {
			currentAnim = newAnim
			animFrame = 0
			animTick = 0
		}

		a, ok := m.States[currentAnim]
		if !ok {
			a = m.States["idle"]
		}

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
