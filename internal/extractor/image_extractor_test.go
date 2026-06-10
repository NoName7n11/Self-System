package extractor_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"selfsystems/internal/extractor"
)

// ---------------------------------------------------------------------------
// Image fixture helpers
// ---------------------------------------------------------------------------

// makePNG returns PNG-encoded bytes for a solid-colour image of the given size.
func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	blue := color.RGBA{R: 50, G: 100, B: 200, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, blue)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// makeJPEG returns JPEG-encoded bytes for a solid-colour image of the given size.
func makeJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	green := color.RGBA{R: 50, G: 180, B: 80, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, green)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestImageExtractor_PNGLandscapeIsScreenshot(t *testing.T) {
	// 1920×1080 PNG, aspect ≈ 1.78 → screenshot
	content := makePNG(1920, 1080)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "png" {
		t.Errorf("format = %q, want png", result.Format)
	}
	if result.Width != 1920 || result.Height != 1080 {
		t.Errorf("dimensions = %dx%d, want 1920x1080", result.Width, result.Height)
	}
	if result.ImageType != extractor.ImageTypeScreenshot {
		t.Errorf("image_type = %q, want %q", result.ImageType, extractor.ImageTypeScreenshot)
	}
	if len(result.ThumbnailPNG) == 0 {
		t.Error("thumbnail not generated")
	}
}

func TestImageExtractor_FilenameHintOverridesHeuristic(t *testing.T) {
	// Even a portrait PNG should be classified as screenshot if filename says so
	content := makePNG(400, 800)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "screenshot_2026.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ImageType != extractor.ImageTypeScreenshot {
		t.Errorf("image_type = %q, want %q (filename hint)", result.ImageType, extractor.ImageTypeScreenshot)
	}
}

func TestImageExtractor_SquarePNGIsDiagram(t *testing.T) {
	// 512×512 PNG, aspect = 1.0 → diagram
	content := makePNG(512, 512)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ImageType != extractor.ImageTypeDiagram {
		t.Errorf("image_type = %q, want %q", result.ImageType, extractor.ImageTypeDiagram)
	}
}

func TestImageExtractor_JPEGIsPhoto(t *testing.T) {
	content := makeJPEG(1200, 800)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "holiday.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "jpeg" {
		t.Errorf("format = %q, want jpeg", result.Format)
	}
	if result.ImageType != extractor.ImageTypePhoto {
		t.Errorf("image_type = %q, want %q", result.ImageType, extractor.ImageTypePhoto)
	}
}

func TestImageExtractor_ThumbnailScalesDown(t *testing.T) {
	// 800×600 PNG — larger than thumbnailMaxDim (200), must be scaled
	content := makePNG(800, 600)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ThumbnailPNG) == 0 {
		t.Fatal("thumbnail not generated")
	}

	// Decode thumbnail and verify dimensions are ≤ 200
	thumbImg, _, err := image.Decode(bytes.NewReader(result.ThumbnailPNG))
	if err != nil {
		t.Fatalf("thumbnail decode error: %v", err)
	}
	b := thumbImg.Bounds()
	if b.Dx() > 200 || b.Dy() > 200 {
		t.Errorf("thumbnail too large: %dx%d, want ≤ 200×200", b.Dx(), b.Dy())
	}
	// Verify aspect ratio preserved (800:600 = 4:3 → 200:150)
	if b.Dx() != 200 || b.Dy() != 150 {
		t.Errorf("thumbnail dimensions = %dx%d, want 200x150 (aspect preserved)", b.Dx(), b.Dy())
	}
}

func TestImageExtractor_SmallImageNotUpscaled(t *testing.T) {
	// 50×50 PNG — already fits in 200×200, should not be upscaled
	content := makePNG(50, 50)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	thumbImg, _, err := image.Decode(bytes.NewReader(result.ThumbnailPNG))
	if err != nil {
		t.Fatalf("thumbnail decode error: %v", err)
	}
	b := thumbImg.Bounds()
	if b.Dx() != 50 || b.Dy() != 50 {
		t.Errorf("small image was upscaled to %dx%d, want 50×50", b.Dx(), b.Dy())
	}
}

func TestImageExtractor_ThumbnailBase64NonEmpty(t *testing.T) {
	content := makePNG(400, 300)

	ex := extractor.NewImageExtractor()
	result, err := ex.Extract(context.Background(), content, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b64 := result.ThumbnailBase64()
	if b64 == "" {
		t.Error("ThumbnailBase64 returned empty string")
	}
	// Verify it is valid base64
	if _, err := base64.StdEncoding.DecodeString(b64); err != nil {
		t.Errorf("ThumbnailBase64 is not valid base64: %v", err)
	}
}

func TestImageExtractor_EmptyContent(t *testing.T) {
	ex := extractor.NewImageExtractor()
	_, err := ex.Extract(context.Background(), []byte{}, "")
	if err == nil {
		t.Error("expected error for empty content, got nil")
	}
}

func TestImageExtractor_InvalidBytes(t *testing.T) {
	ex := extractor.NewImageExtractor()
	_, err := ex.Extract(context.Background(), []byte("not an image"), "")
	if err == nil {
		t.Error("expected error for invalid image bytes, got nil")
	}
}

func TestImageExtractor_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	content := makePNG(100, 100)
	ex := extractor.NewImageExtractor()
	_, err := ex.Extract(ctx, content, "")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}
