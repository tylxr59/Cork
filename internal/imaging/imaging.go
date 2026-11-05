// Package imaging validates uploaded images and produces grid thumbnails.
package imaging

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ThumbWidth is the display width thumbnails are scaled down to. The grid shows
// pins at ~240 CSS px, so this covers 2x displays.
const ThumbWidth = 600

// MaxPixels guards against decompression bombs.
const MaxPixels = 100_000_000

var ErrUnsupported = errors.New("unsupported image format (use JPEG, PNG, GIF or WebP)")

type Result struct {
	// Width and Height are the display dimensions, after EXIF orientation.
	Width, Height int
	// Ext is the file extension for the original, including the dot.
	Ext string
	// Thumb is the encoded thumbnail (JPEG, or PNG when the image has transparency).
	// It is nil for animated GIFs, which are shown as-is.
	Thumb []byte
	// Color is the average color as #rrggbb, used as a loading placeholder.
	Color string
}

// Process validates data as an image and builds its thumbnail.
func Process(data []byte) (*Result, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return nil, fmt.Errorf("image dimensions %dx%d are not allowed", cfg.Width, cfg.Height)
	}

	res := &Result{Width: cfg.Width, Height: cfg.Height}
	switch format {
	case "jpeg":
		res.Ext = ".jpg"
	case "png":
		res.Ext = ".png"
	case "webp":
		res.Ext = ".webp"
	case "gif":
		res.Ext = ".gif"
	default:
		return nil, ErrUnsupported
	}

	if format == "gif" {
		g, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil || len(g.Image) == 0 {
			return nil, ErrUnsupported
		}
		res.Color = averageColor(g.Image[0])
		if len(g.Image) > 1 {
			return res, nil // keep animation: no thumbnail
		}
	}

	orientation := 1
	if format == "jpeg" {
		orientation = jpegOrientation(data)
	}
	if orientation >= 5 {
		res.Width, res.Height = res.Height, res.Width
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}

	// Scale so the *displayed* width is at most ThumbWidth.
	scale := 1.0
	if res.Width > ThumbWidth {
		scale = float64(ThumbWidth) / float64(res.Width)
	}
	b := src.Bounds()
	tw := max(1, int(float64(b.Dx())*scale+0.5))
	th := max(1, int(float64(b.Dy())*scale+0.5))
	scaled := image.NewNRGBA(image.Rect(0, 0, tw, th))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, draw.Src, nil)
	thumb := orient(scaled, orientation)

	res.Color = averageColor(thumb)

	var buf bytes.Buffer
	if thumb.Opaque() {
		err = jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 82})
	} else {
		err = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, thumb)
	}
	if err != nil {
		return nil, err
	}
	res.Thumb = buf.Bytes()
	return res, nil
}

func averageColor(img image.Image) string {
	b := img.Bounds()
	step := max(1, min(b.Dx(), b.Dy())/32)
	var r, g, bl, n uint64
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A < 32 {
				continue
			}
			r += uint64(c.R)
			g += uint64(c.G)
			bl += uint64(c.B)
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", r/n, g/n, bl/n)
}

// orient applies an EXIF orientation (1-8) so the image displays upright.
func orient(src *image.NRGBA, o int) *image.NRGBA {
	if o < 2 || o > 8 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			var sx, sy int
			switch o {
			case 2:
				sx, sy = w-1-dx, dy
			case 3:
				sx, sy = w-1-dx, h-1-dy
			case 4:
				sx, sy = dx, h-1-dy
			case 5:
				sx, sy = dy, dx
			case 6:
				sx, sy = dy, h-1-dx
			case 7:
				sx, sy = w-1-dy, h-1-dx
			case 8:
				sx, sy = w-1-dy, dx
			}
			si := src.PixOffset(sx, sy)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// jpegOrientation reads the EXIF orientation tag from a JPEG, returning 1 if absent.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // start of scan / end: no EXIF found
			return 1
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		if segLen < 2 || i+2+segLen > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+segLen]
		if marker == 0xE1 && len(seg) > 6 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + segLen
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var u16 func([]byte) int
	var u32 func([]byte) int
	switch string(t[:2]) {
	case "II":
		u16 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 }
		u32 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) int { return int(b[1]) | int(b[0])<<8 }
		u32 = func(b []byte) int { return int(b[3]) | int(b[2])<<8 | int(b[1])<<16 | int(b[0])<<24 }
	default:
		return 1
	}
	off := u32(t[4:8])
	if off < 8 || off+2 > len(t) {
		return 1
	}
	n := u16(t[off : off+2])
	for k := 0; k < n; k++ {
		e := off + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if u16(t[e:e+2]) == 0x0112 {
			v := u16(t[e+8 : e+10])
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}
