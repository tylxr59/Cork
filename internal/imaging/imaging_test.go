package imaging

import (
	"image"
	"image/color"
	"testing"
)

func TestOrient(t *testing.T) {
	// 3x2 image with a marked top-left pixel
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.Set(0, 0, color.NRGBA{255, 0, 0, 255})

	cases := map[int]struct{ w, h, x, y int }{
		1: {3, 2, 0, 0},
		2: {3, 2, 2, 0},
		3: {3, 2, 2, 1},
		4: {3, 2, 0, 1},
		5: {2, 3, 0, 0},
		6: {2, 3, 1, 0}, // rotate 90° clockwise: top-left goes top-right
		7: {2, 3, 1, 2},
		8: {2, 3, 0, 2}, // rotate 90° counter-clockwise: top-left goes bottom-left
	}
	for o, want := range cases {
		got := orient(src, o)
		if b := got.Bounds(); b.Dx() != want.w || b.Dy() != want.h {
			t.Errorf("orientation %d: size %dx%d, want %dx%d", o, b.Dx(), b.Dy(), want.w, want.h)
			continue
		}
		if c := got.NRGBAAt(want.x, want.y); c.R != 255 {
			t.Errorf("orientation %d: marked pixel not at (%d,%d)", o, want.x, want.y)
		}
	}
}

func TestTiffOrientation(t *testing.T) {
	// little-endian TIFF header + one IFD entry: orientation = 6
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0}
	if got := tiffOrientation(tiff); got != 6 {
		t.Fatalf("got %d, want 6", got)
	}
}
