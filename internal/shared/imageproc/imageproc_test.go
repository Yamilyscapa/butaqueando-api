package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestOptimizeImageResizesFitInsideAndConvertsToWebP(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 2400, 1200))
	fillImage(src, color.RGBA{R: 120, G: 10, B: 200, A: 255})

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode source jpeg: %v", err)
	}

	result, err := OptimizeImage(buf.Bytes(), OptimizeOptions{Quality: 80, MaxWidth: 1080, MaxHeight: 1920, TargetContentType: "image/webp"})
	if err != nil {
		t.Fatalf("optimize image: %v", err)
	}

	if result.ContentType != "image/webp" {
		t.Fatalf("expected image/webp content type, got %q", result.ContentType)
	}

	decoded, _, err := image.Decode(bytes.NewReader(result.Content))
	if err != nil {
		t.Fatalf("decode optimized image: %v", err)
	}

	bounds := decoded.Bounds()
	if bounds.Dx() != 1080 || bounds.Dy() != 540 {
		t.Fatalf("expected 1080x540 result, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestOptimizeImageDoesNotUpscale(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 120, 100))
	fillImage(src, color.RGBA{R: 50, G: 120, B: 30, A: 255})

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode source jpeg: %v", err)
	}

	result, err := OptimizeImage(buf.Bytes(), OptimizeOptions{Quality: 80, MaxWidth: 200, MaxHeight: 200, TargetContentType: "image/webp"})
	if err != nil {
		t.Fatalf("optimize image: %v", err)
	}

	decoded, _, err := image.Decode(bytes.NewReader(result.Content))
	if err != nil {
		t.Fatalf("decode optimized image: %v", err)
	}

	bounds := decoded.Bounds()
	if bounds.Dx() != 120 || bounds.Dy() != 100 {
		t.Fatalf("expected 120x100 result, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func fillImage(img *image.RGBA, c color.RGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetRGBA(x, y, c)
		}
	}
}
