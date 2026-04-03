// Package imaging provides an ImageProcessor adapter backed by
// github.com/disintegration/imaging (pure Go, no CGO).
package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/disintegration/imaging"

	"github.com/polagonow/pola/imageprocessing"
)

type processor struct{}

// New returns an ImageProcessor backed by disintegration/imaging.
func New() imageprocessing.ImageProcessor { return &processor{} }

func (p *processor) Process(src []byte, opts imageprocessing.ProcessOptions) ([]byte, string, error) {
	img, srcFormat, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	// Resize / fit / crop.
	if opts.Width > 0 || opts.Height > 0 {
		img = applyResize(img, opts.Width, opts.Height, opts.Fit)
	}

	// Rotate.
	switch opts.Rotate {
	case 90:
		img = imaging.Rotate90(img)
	case 180:
		img = imaging.Rotate180(img)
	case 270:
		img = imaging.Rotate270(img)
	}

	// Blur.
	if opts.Blur > 0 {
		img = imaging.Blur(img, opts.Blur)
	}

	// Sharpen.
	if opts.Sharpen > 0 {
		img = imaging.Sharpen(img, opts.Sharpen)
	}

	// Determine output format.
	outFormat := strings.ToLower(opts.Format)
	if outFormat == "" {
		outFormat = srcFormat
	}

	data, contentType, err := encode(img, outFormat, opts.Quality)
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}

func applyResize(img image.Image, width, height int, fit string) image.Image {
	switch strings.ToLower(fit) {
	case "cover":
		return imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	case "contain":
		return imaging.Fit(img, width, height, imaging.Lanczos)
	default: // "fill" or empty
		return imaging.Resize(img, width, height, imaging.Lanczos)
	}
}

func encode(img image.Image, format string, quality int) ([]byte, string, error) {
	var buf bytes.Buffer

	switch format {
	case "jpeg", "jpg":
		q := quality
		if q <= 0 {
			q = 85
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, "", fmt.Errorf("encode jpeg: %w", err)
		}
		return buf.Bytes(), "image/jpeg", nil

	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("encode png: %w", err)
		}
		return buf.Bytes(), "image/png", nil

	case "gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, "", fmt.Errorf("encode gif: %w", err)
		}
		return buf.Bytes(), "image/gif", nil

	default:
		// Fall back to JPEG for unsupported formats.
		q := quality
		if q <= 0 {
			q = 85
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, "", fmt.Errorf("encode jpeg: %w", err)
		}
		return buf.Bytes(), "image/jpeg", nil
	}
}
