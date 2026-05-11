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

	"github.com/bbrks/go-blurhash"
	gowebp "github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const defaultQuality = 75

type OptimizeOptions struct {
	Quality           int
	MaxWidth          int
	MaxHeight         int
	TargetContentType string
}

type OptimizeResult struct {
	Content     []byte
	ContentType string
	Width       int
	Height      int
	Optimized   bool
}

// VariantSpec declares a single output variant for OptimizeImageVariants.
type VariantSpec struct {
	MaxWidth          int
	MaxHeight         int
	Quality           int
	TargetContentType string
}

// PlayMediaVariants is the default variant plan for play media (portrait-oriented).
// Widths chosen to map to mobile surfaces: cards/avatars (320), carousels (720), full poster (1080).
var PlayMediaVariants = []VariantSpec{
	{MaxWidth: 320, MaxHeight: 568, TargetContentType: "image/webp"},
	{MaxWidth: 720, MaxHeight: 1280, TargetContentType: "image/webp"},
	{MaxWidth: 1080, MaxHeight: 1920, TargetContentType: "image/webp"},
}

// AvatarVariants is the default plan for square user avatars.
var AvatarVariants = []VariantSpec{
	{MaxWidth: 96, MaxHeight: 96, TargetContentType: "image/webp"},
	{MaxWidth: 240, MaxHeight: 240, TargetContentType: "image/webp"},
	{MaxWidth: 512, MaxHeight: 512, TargetContentType: "image/webp"},
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

	return encodeFromImage(img, format, opts.Quality, targetContentType, opts.MaxWidth, opts.MaxHeight, src)
}

// OptimizeImageVariants decodes the source ONCE and produces one OptimizeResult per spec.
// Resize + encode happens per variant; decode work is amortized.
func OptimizeImageVariants(src []byte, plan []VariantSpec) ([]OptimizeResult, error) {
	if len(plan) == 0 {
		return nil, fmt.Errorf("empty variant plan")
	}

	img, format, err := decodeImage(src)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	results := make([]OptimizeResult, 0, len(plan))
	for _, spec := range plan {
		quality := spec.Quality
		if quality <= 0 || quality > 100 {
			quality = defaultQuality
		}
		ct := strings.ToLower(strings.TrimSpace(spec.TargetContentType))
		if ct == "" {
			ct = "image/webp"
		}

		res, err := encodeFromImage(img, format, quality, ct, spec.MaxWidth, spec.MaxHeight, src)
		if err != nil {
			return nil, fmt.Errorf("variant %dx%d: %w", spec.MaxWidth, spec.MaxHeight, err)
		}
		results = append(results, res)
	}
	return results, nil
}

// GenerateBlurhash returns a compact blurhash string for the given image.
// Uses 4x3 components which yields ~20-character hashes, well-suited for portrait posters.
func GenerateBlurhash(src image.Image) (string, error) {
	if src == nil {
		return "", fmt.Errorf("nil image")
	}
	preview := downscaleForHash(src)
	hash, err := blurhash.Encode(4, 3, preview)
	if err != nil {
		return "", fmt.Errorf("blurhash encode: %w", err)
	}
	return hash, nil
}

// GenerateBlurhashFromBytes is a convenience wrapper that decodes src first.
func GenerateBlurhashFromBytes(src []byte) (string, error) {
	img, _, err := decodeImage(src)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	return GenerateBlurhash(img)
}

func encodeFromImage(
	img image.Image,
	format string,
	quality int,
	targetContentType string,
	maxWidth int,
	maxHeight int,
	src []byte,
) (OptimizeResult, error) {
	resized := resizeFitInside(img, maxWidth, maxHeight)
	bounds := resized.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	buf := &bytes.Buffer{}
	switch targetContentType {
	case "image/webp":
		if err := gowebp.Encode(buf, resized, gowebp.Options{Quality: quality}); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode webp: %w", err)
		}
		optimized := format != "webp" || src == nil || !bytes.Equal(src, buf.Bytes())
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/webp", Width: width, Height: height, Optimized: optimized}, nil
	case "image/jpeg":
		if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: quality}); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode jpeg: %w", err)
		}
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/jpeg", Width: width, Height: height, Optimized: true}, nil
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(buf, resized); err != nil {
			return OptimizeResult{}, fmt.Errorf("encode png: %w", err)
		}
		return OptimizeResult{Content: buf.Bytes(), ContentType: "image/png", Width: width, Height: height, Optimized: true}, nil
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

// downscaleForHash returns a small copy of src capped at 64px long-edge to keep blurhash encoding cheap.
func downscaleForHash(src image.Image) image.Image {
	const maxEdge = 64
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	scale := math.Min(float64(maxEdge)/float64(w), float64(maxEdge)/float64(h))
	tw := int(math.Round(float64(w) * scale))
	th := int(math.Round(float64(h) * scale))
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
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
