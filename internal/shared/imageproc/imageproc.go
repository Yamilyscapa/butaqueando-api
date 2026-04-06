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
)

const defaultQuality = 80

type OptimizeOptions struct {
	Quality int
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

	img, format, err := decodeImage(src)
	if err != nil {
		return OptimizeResult{}, fmt.Errorf("decode image: %w", err)
	}

	if format == "webp" {
		return OptimizeResult{Content: src, ContentType: "image/webp", Optimized: false}, nil
	}

	buf := &bytes.Buffer{}
	switch format {
	case "jpeg":
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: opts.Quality}); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode jpeg: %w", err)
		}
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/jpeg", Optimized: true}, nil
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(buf, img); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode png: %w", err)
		}
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/png", Optimized: true}, nil
	default:
		return OptimizeResult{}, fmt.Errorf("unsupported image format")
	}
}

func decodeImage(src []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(src))
	if err == nil {
		return img, format, nil
	}

	return nil, "", fmt.Errorf("unsupported image format")
}
