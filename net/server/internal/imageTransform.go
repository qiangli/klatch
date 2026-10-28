package databag

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// transformImage decodes and rewrites an image without requiring ImageMagick.
// Re-encoding also removes metadata from the result.
func transformImage(kind, input, output string) error {
	source, err := os.Open(input)
	if err != nil {
		return err
	}
	defer source.Close()
	img, format, err := image.Decode(source)
	if err != nil {
		return err
	}
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width == 0 || height == 0 {
		return fmt.Errorf("empty image")
	}
	limit := 0
	switch kind {
	case "icopy":
	case "ithumb":
		limit = 192
	case "ilg":
		limit = 1024
	default:
		return fmt.Errorf("unsupported image transform: %s", kind)
	}
	if limit > 0 && (kind == "ithumb" || width > limit || height > limit) {
		scale := float64(limit) / float64(width)
		if hscale := float64(limit) / float64(height); hscale < scale {
			scale = hscale
		}
		if kind == "ilg" && scale > 1 {
			scale = 1
		}
		w, h := int(float64(width)*scale+0.5), int(float64(height)*scale+0.5)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		resized := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = resized
	}
	dest, err := os.Create(output)
	if err != nil {
		return err
	}
	defer dest.Close()
	switch format {
	case "jpeg":
		err = jpeg.Encode(dest, img, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(dest, img)
	case "gif":
		err = gif.Encode(dest, img, nil)
	default:
		err = fmt.Errorf("unsupported image format: %s", format)
	}
	if err != nil {
		os.Remove(output)
	}
	return err
}
