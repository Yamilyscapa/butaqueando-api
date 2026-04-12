package imageproc

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"math"
	"strings"

	gowebp "github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const defaultQuality = 80

type OptimizeOptions struct {
	Quality           int
	MaxWidth          int
	MaxHeight         int
	TargetContentType string
}

type OptimizeResult struct {
	Content     []byte
	ContentType string
	Optimized   bool
}

func OptimizeImage(src []byte, opts OptimizeOptions) (OptimizeResult, error) {
	if opts.Quality <= 0 || opts.Quality > 100 {
		opts.Quality = defaultQuality
	}

	targetContentType := strings.ToLower(strings.TrimSpace(opts.TargetContentType))
	if targetContentType == "" {
		targetContentType = "image/webp"
	}

	img, format, err := decodeImage(src)
	if err != nil {
		return OptimizeResult{}, fmt.Errorf("decode image: %w", err)
	}

	img = resizeFitInside(img, opts.MaxWidth, opts.MaxHeight)

	buf := &bytes.Buffer{}
	switch targetContentType {
	case "image/webp":
		if err := gowebp.Encode(buf, img, gowebp.Options{Quality: opts.Quality}); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode webp: %w", err)
		}
		optimized := format != "webp" || !bytes.Equal(src, buf.Bytes())
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/webp", Optimized: optimized}, nil
	case "image/jpeg":
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: opts.Quality}); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode jpeg: %w", err)
		}
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/jpeg", Optimized: true}, nil
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(buf, img); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode png: %w", err)
		}
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/png", Optimized: true}, nil
	default:
		return OptimizeResult{}, fmt.Errorf("unsupported target content type")
	}
}

func decodeImage(src []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(src))
	if err == nil {
		return img, format, nil
	}

	return nil, "", fmt.Errorf("unsupported image format")
}

func resizeFitInside(src image.Image, maxWidth int, maxHeight int) image.Image {
	if src == nil {
		return nil
	}

	if maxWidth <= 0 || maxHeight <= 0 {
		return src
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}

	scale := math.Min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	if scale >= 1 {
		return src
	}

	targetWidth := int(math.Round(float64(width) * scale))
	targetHeight := int(math.Round(float64(height) * scale))
	if targetWidth < 1 {
		targetWidth = 1
	}
	if targetHeight < 1 {
		targetHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	return dst
}
