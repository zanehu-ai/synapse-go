package imaging

import (
	"errors"
	"image"
	"io"
	"strings"
)

// ValidateConfig defines constraints for image validation.
type ValidateConfig struct {
	MaxFileSize  int64    // max file size in bytes (0 = no limit)
	MaxWidth     int      // max pixel width (0 = no limit)
	MaxHeight    int      // max pixel height (0 = no limit)
	AllowedTypes []string // allowed MIME sub-types: "jpeg", "png", "gif" (empty = allow all)
}

// Validate checks an image against the given constraints without fully decoding.
// It reads the image header to determine format and dimensions.
func Validate(r io.Reader, size int64, cfg ValidateConfig) error {
	if cfg.MaxFileSize > 0 && size > cfg.MaxFileSize {
		return errors.New("image exceeds maximum file size")
	}

	config, format, err := image.DecodeConfig(r)
	if err != nil {
		return errors.New("invalid image format")
	}

	if len(cfg.AllowedTypes) > 0 {
		allowed := false
		for _, t := range cfg.AllowedTypes {
			if strings.EqualFold(format, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.New("image format not allowed: " + format)
		}
	}

	if cfg.MaxWidth > 0 && config.Width > cfg.MaxWidth {
		return errors.New("image width exceeds maximum")
	}
	if cfg.MaxHeight > 0 && config.Height > cfg.MaxHeight {
		return errors.New("image height exceeds maximum")
	}

	return nil
}
