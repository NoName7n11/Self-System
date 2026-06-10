package extractor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	// Register decoders so image.DecodeConfig can detect formats.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	thumbnailMaxDim = 200 // maximum width or height for generated thumbnails

	// Image type constants returned in ImageExtractResult.ImageType.
	ImageTypeScreenshot = "screenshot"
	ImageTypePhoto      = "photo"
	ImageTypeDiagram    = "diagram"
	ImageTypeUnknown    = "unknown"
)

// ImageExtractResult holds the data produced by WS3 image processing.
// OCRText is intentionally empty here — it is populated in Change 7 via AI vision.
type ImageExtractResult struct {
	Format       string // "png", "jpeg", "gif", etc. as reported by image.DecodeConfig
	Width        int
	Height       int
	ImageType    string // screenshot / photo / diagram / unknown
	ThumbnailPNG []byte // PNG-encoded thumbnail, max thumbnailMaxDim × thumbnailMaxDim
}

// ImageExtractor performs heuristic image classification and thumbnail generation.
// It is safe for concurrent use. OCR is deferred to Change 7 (AI vision layer).
type ImageExtractor struct{}

// NewImageExtractor returns an ImageExtractor ready to use.
func NewImageExtractor() *ImageExtractor { return &ImageExtractor{} }

// Extract processes image bytes and returns metadata + a thumbnail.
// filename is optional; if non-empty it is used as a classification hint.
// A non-nil error means the bytes could not be decoded as a supported image.
func (e *ImageExtractor) Extract(ctx context.Context, content []byte, filename string) (ImageExtractResult, error) {
	if err := ctx.Err(); err != nil {
		return ImageExtractResult{}, err
	}
	if len(content) == 0 {
		return ImageExtractResult{}, fmt.Errorf("image content is empty")
	}

	// Probe format and dimensions without decoding the full pixel data.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return ImageExtractResult{}, fmt.Errorf("decode image config: %w", err)
	}

	imgType := classifyImage(format, cfg.Width, cfg.Height, filename)

	thumb, err := generateThumbnail(content, format, thumbnailMaxDim)
	if err != nil {
		// Thumbnail failure is non-fatal — return the metadata without it.
		thumb = nil
	}

	return ImageExtractResult{
		Format:       format,
		Width:        cfg.Width,
		Height:       cfg.Height,
		ImageType:    imgType,
		ThumbnailPNG: thumb,
	}, nil
}

// ThumbnailBase64 returns the thumbnail PNG encoded as a base64 string,
// or an empty string if no thumbnail was generated.
func (r ImageExtractResult) ThumbnailBase64() string {
	if len(r.ThumbnailPNG) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(r.ThumbnailPNG)
}

// ---- Classification ---------------------------------------------------------

// classifyImage returns a coarse image type using heuristics.
// The rules are intentionally conservative — AI refinement happens in Change 7.
//
//   - screenshot: PNG format AND landscape-oriented AND wide (≥ 800 px wide or
//     aspect ratio ≥ 1.6). Most desktop/mobile screenshots are saved as PNG and
//     are wider than they are tall.
//   - photo: JPEG/JPG format AND not classified as screenshot.
//   - diagram: square-ish aspect ratio (0.75 ≤ w/h ≤ 1.33) AND PNG format AND
//     not already screenshot.
//   - unknown: everything else.
func classifyImage(format string, w, h int, filename string) string {
	// Filename hint takes precedence.
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "screenshot") || strings.Contains(lower, "screen_") ||
		strings.Contains(lower, "capture") || strings.Contains(lower, "snap") {
		return ImageTypeScreenshot
	}

	if h == 0 {
		return ImageTypeUnknown
	}
	aspect := float64(w) / float64(h)

	switch format {
	case "png":
		// Wide landscape PNG is almost certainly a screenshot.
		if aspect >= 1.6 && w >= 800 {
			return ImageTypeScreenshot
		}
		// Square-ish PNG is likely a diagram, logo, or chart.
		if aspect >= 0.75 && aspect <= 1.33 {
			return ImageTypeDiagram
		}
		return ImageTypeUnknown

	case "jpeg":
		return ImageTypePhoto

	default:
		return ImageTypeUnknown
	}
}

// ---- Thumbnail generation ---------------------------------------------------

// generateThumbnail decodes content and returns a PNG-encoded thumbnail
// scaled to fit within maxDim × maxDim, preserving aspect ratio.
func generateThumbnail(content []byte, format string, maxDim int) ([]byte, error) {
	var src image.Image
	var err error

	switch format {
	case "png":
		src, err = png.Decode(bytes.NewReader(content))
	case "jpeg":
		src, err = jpeg.Decode(bytes.NewReader(content))
	default:
		// Attempt generic decode for gif and others.
		src, _, err = image.Decode(bytes.NewReader(content))
	}
	if err != nil {
		return nil, fmt.Errorf("decode image pixels: %w", err)
	}

	thumb := scaleDown(src, maxDim)

	var buf bytes.Buffer
	if err := png.Encode(&buf, thumb); err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}
	return buf.Bytes(), nil
}

// scaleDown returns a new image scaled so its longest side is ≤ maxDim,
// using nearest-neighbour sampling (stdlib only, no external deps).
// If the image already fits, it is returned unchanged.
func scaleDown(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()

	if srcW <= maxDim && srcH <= maxDim {
		return src
	}

	// Compute target dimensions preserving aspect ratio.
	dstW, dstH := srcW, srcH
	if srcW > srcH {
		dstW = maxDim
		dstH = srcH * maxDim / srcW
	} else {
		dstH = maxDim
		dstW = srcW * maxDim / srcH
	}
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		srcY := y*srcH/dstH + b.Min.Y
		for x := 0; x < dstW; x++ {
			srcX := x*srcW/dstW + b.Min.X
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}
