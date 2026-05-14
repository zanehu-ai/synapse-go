package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// createTestImage generates a solid-color test image.
func createTestImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	return img
}

func encodeJPEG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		panic("jpeg.Encode: " + err.Error())
	}
	return buf.Bytes()
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("png.Encode: " + err.Error())
	}
	return buf.Bytes()
}

func TestResize_ScaleDown(t *testing.T) {
	img := createTestImage(800, 600)

	data, err := Resize(img, ResizeOption{MaxWidth: 400, MaxHeight: 300, Format: JPEG})
	if err != nil {
		t.Fatalf("Resize() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Resize() returned empty data")
	}

	// Decode result to verify dimensions
	result, _, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode resized: %v", err)
	}
	bounds := result.Bounds()
	if bounds.Dx() > 400 || bounds.Dy() > 300 {
		t.Errorf("resized dimensions %dx%d exceed max 400x300", bounds.Dx(), bounds.Dy())
	}
}

func TestResize_NoScaleIfSmaller(t *testing.T) {
	img := createTestImage(200, 100)

	data, err := Resize(img, ResizeOption{MaxWidth: 400, MaxHeight: 300, Format: JPEG})
	if err != nil {
		t.Fatalf("Resize() error: %v", err)
	}

	result, _, _ := Decode(bytes.NewReader(data))
	bounds := result.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 100 {
		t.Errorf("should not resize: got %dx%d, want 200x100", bounds.Dx(), bounds.Dy())
	}
}

func TestResize_AspectRatio(t *testing.T) {
	img := createTestImage(1000, 500) // 2:1 ratio

	data, _ := Resize(img, ResizeOption{MaxWidth: 400, Format: JPEG})
	result, _, _ := Decode(bytes.NewReader(data))
	bounds := result.Bounds()

	// Should maintain ~2:1 ratio
	if bounds.Dx() != 400 {
		t.Errorf("width = %d, want 400", bounds.Dx())
	}
	if bounds.Dy() != 200 {
		t.Errorf("height = %d, want 200", bounds.Dy())
	}
}

func TestResize_PNG(t *testing.T) {
	img := createTestImage(800, 600)

	data, err := Resize(img, ResizeOption{MaxWidth: 400, Format: PNG})
	if err != nil {
		t.Fatalf("Resize() PNG error: %v", err)
	}

	_, format, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want png", format)
	}
}

func TestCompress_JPEG(t *testing.T) {
	img := createTestImage(400, 300)

	highQ, _ := Compress(img, 95, JPEG)
	lowQ, _ := Compress(img, 30, JPEG)

	if len(lowQ) >= len(highQ) {
		t.Errorf("low quality (%d bytes) should be smaller than high quality (%d bytes)", len(lowQ), len(highQ))
	}
}

func TestThumbnail(t *testing.T) {
	img := createTestImage(800, 600) // non-square

	data, err := Thumbnail(img, 100, 80)
	if err != nil {
		t.Fatalf("Thumbnail() error: %v", err)
	}

	result, _, _ := Decode(bytes.NewReader(data))
	bounds := result.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("thumbnail = %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}
}

func TestDecode_JPEG(t *testing.T) {
	img := createTestImage(100, 100)
	data := encodeJPEG(img)

	_, format, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("format = %q, want jpeg", format)
	}
}

func TestDecode_PNG(t *testing.T) {
	img := createTestImage(100, 100)
	data := encodePNG(img)

	_, format, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want png", format)
	}
}

func TestValidate_Pass(t *testing.T) {
	img := createTestImage(400, 300)
	data := encodeJPEG(img)

	err := Validate(bytes.NewReader(data), int64(len(data)), ValidateConfig{
		MaxFileSize:  1024 * 1024,
		MaxWidth:     1920,
		MaxHeight:    1080,
		AllowedTypes: []string{"jpeg", "png"},
	})
	if err != nil {
		t.Errorf("Validate() should pass: %v", err)
	}
}

func TestValidate_FileSize(t *testing.T) {
	img := createTestImage(100, 100)
	data := encodeJPEG(img)

	err := Validate(bytes.NewReader(data), int64(len(data)), ValidateConfig{
		MaxFileSize: 10, // 10 bytes — way too small
	})
	if err == nil {
		t.Error("Validate() should fail for oversized file")
	}
}

func TestValidate_Dimensions(t *testing.T) {
	img := createTestImage(2000, 1500)
	data := encodeJPEG(img)

	err := Validate(bytes.NewReader(data), int64(len(data)), ValidateConfig{
		MaxWidth:  1920,
		MaxHeight: 1080,
	})
	if err == nil {
		t.Error("Validate() should fail for oversized dimensions")
	}
}

func TestValidate_FormatNotAllowed(t *testing.T) {
	img := createTestImage(100, 100)
	data := encodePNG(img)

	err := Validate(bytes.NewReader(data), int64(len(data)), ValidateConfig{
		AllowedTypes: []string{"jpeg"}, // only JPEG allowed
	})
	if err == nil {
		t.Error("Validate() should fail for disallowed format")
	}
}

func TestValidate_InvalidData(t *testing.T) {
	err := Validate(bytes.NewReader([]byte("not an image")), 12, ValidateConfig{})
	if err == nil {
		t.Error("Validate() should fail for invalid data")
	}
}

func TestFitDimensions(t *testing.T) {
	tests := []struct {
		srcW, srcH, maxW, maxH int
		wantW, wantH           int
	}{
		{1000, 500, 400, 0, 400, 200},  // constrain by width
		{500, 1000, 0, 400, 200, 400},  // constrain by height
		{800, 600, 400, 300, 400, 300}, // constrain by both (same ratio)
		{800, 600, 400, 400, 400, 300}, // constrain by width (narrower limit)
		{200, 100, 400, 300, 200, 100}, // no upscale (both within limits)
		{100, 100, 0, 0, 100, 100},     // no limits
	}
	for _, tt := range tests {
		w, h := fitDimensions(tt.srcW, tt.srcH, tt.maxW, tt.maxH)
		if w != tt.wantW || h != tt.wantH {
			t.Errorf("fitDimensions(%d,%d,%d,%d) = %d,%d, want %d,%d",
				tt.srcW, tt.srcH, tt.maxW, tt.maxH, w, h, tt.wantW, tt.wantH)
		}
	}
}
