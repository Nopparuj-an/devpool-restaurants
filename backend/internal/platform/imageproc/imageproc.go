// Package imageproc shrinks uploaded photos before they are stored, so
// storage stays small whatever the client sends. The browser already does
// the same before upload (frontend lib/image.ts); this is the server-side
// guarantee for any other client.
package imageproc

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // register decoder
	"math"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register decoder
)

const (
	// MaxSide is the longest edge a stored photo keeps.
	MaxSide = 1600
	// MaxPixels rejects "decompression bombs": small files that decode to
	// enormous images and would exhaust memory.
	MaxPixels = 40_000_000
	// keepBelow: photos already within MaxSide and this size are stored untouched.
	keepBelow = 1 << 20
	quality   = 82
)

var (
	ErrUnsupported = errors.New("not a JPEG, PNG or WebP image")
	ErrTooManyPx   = errors.New("image has too many pixels")
)

// Result is the photo to store.
type Result struct {
	Data        []byte
	ContentType string
	Ext         string
}

// Normalize returns the photo unchanged if it is already small, otherwise
// resized to fit MaxSide and re-encoded as JPEG (transparency becomes white).
func Normalize(data []byte) (Result, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Result{}, ErrUnsupported
	}
	if cfg.Width*cfg.Height > MaxPixels {
		return Result{}, ErrTooManyPx
	}
	if cfg.Width <= MaxSide && cfg.Height <= MaxSide && len(data) <= keepBelow {
		ct, ext, ok := original(format)
		if ok {
			return Result{Data: data, ContentType: ct, Ext: ext}, nil
		}
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Result{}, ErrUnsupported
	}
	w, h := fit(cfg.Width, cfg.Height)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: quality}); err != nil {
		return Result{}, err
	}
	// Re-encoding a small, already-compressed photo can make it bigger; keep the smaller one.
	if ct, ext, ok := original(format); ok && out.Len() >= len(data) && w == cfg.Width && h == cfg.Height {
		return Result{Data: data, ContentType: ct, Ext: ext}, nil
	}
	return Result{Data: out.Bytes(), ContentType: "image/jpeg", Ext: ".jpg"}, nil
}

func original(format string) (contentType, ext string, ok bool) {
	switch format {
	case "jpeg":
		return "image/jpeg", ".jpg", true
	case "png":
		return "image/png", ".png", true
	case "webp":
		return "image/webp", ".webp", true
	}
	return "", "", false
}

// fit scales (w, h) down so neither side exceeds MaxSide, keeping the ratio.
func fit(w, h int) (int, int) {
	if w <= MaxSide && h <= MaxSide {
		return w, h
	}
	scale := float64(MaxSide) / float64(max(w, h))
	return max(1, int(math.Round(float64(w)*scale))), max(1, int(math.Round(float64(h)*scale)))
}
