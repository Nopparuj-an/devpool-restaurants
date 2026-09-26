package imageproc

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"testing"
)

// photo makes a noisy image, which compresses badly, like a real photo.
func photo(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewPCG(1, 2))
	for i := range img.Pix {
		img.Pix[i] = uint8(r.IntN(256))
	}
	return img
}

func encodePNG(img image.Image) []byte {
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func TestNormalize(t *testing.T) {
	t.Run("large photo is resized and re-encoded as JPEG", func(t *testing.T) {
		in := encodePNG(photo(3200, 2000))
		out, err := Normalize(in)
		if err != nil {
			t.Fatal(err)
		}
		cfg, format, _ := image.DecodeConfig(bytes.NewReader(out.Data))
		if format != "jpeg" || out.ContentType != "image/jpeg" || cfg.Width != 1600 || cfg.Height != 1000 {
			t.Fatalf("got %s %dx%d", format, cfg.Width, cfg.Height)
		}
		if len(out.Data) >= len(in)/4 {
			t.Errorf("%d bytes → %d bytes; expected a big saving", len(in), len(out.Data))
		}
	})

	t.Run("small photo is stored untouched", func(t *testing.T) {
		var b bytes.Buffer
		jpeg.Encode(&b, photo(800, 600), &jpeg.Options{Quality: 80})
		out, err := Normalize(b.Bytes())
		if err != nil || !bytes.Equal(out.Data, b.Bytes()) || out.Ext != ".jpg" {
			t.Fatalf("expected passthrough, got %d bytes %s err=%v", len(out.Data), out.Ext, err)
		}
	})

	t.Run("transparent PNG becomes JPEG on white", func(t *testing.T) {
		img := image.NewNRGBA(image.Rect(0, 0, 2000, 10))
		img.Set(0, 0, color.NRGBA{255, 0, 0, 0}) // fully transparent pixel
		out, err := Normalize(encodePNG(img))
		if err != nil {
			t.Fatal(err)
		}
		dec, _ := jpeg.Decode(bytes.NewReader(out.Data))
		if r, g, b, _ := dec.At(0, 0).RGBA(); r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
			t.Errorf("transparent pixel = %d,%d,%d, want white", r>>8, g>>8, b>>8)
		}
	})

	t.Run("decompression bomb is rejected before decoding", func(t *testing.T) {
		// A 10000x10000 PNG of one color is tiny on disk but 400 MB in memory.
		in := encodePNG(image.NewGray(image.Rect(0, 0, 10000, 10000)))
		if _, err := Normalize(in); !errors.Is(err, ErrTooManyPx) {
			t.Fatalf("err = %v (input %d bytes)", err, len(in))
		}
	})

	t.Run("not an image", func(t *testing.T) {
		if _, err := Normalize([]byte("hello")); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("err = %v", err)
		}
	})
}
