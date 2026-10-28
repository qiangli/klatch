package databag

import (
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestTransformImage(t *testing.T) {
	encoders := map[string]func(*os.File, image.Image) error{
		"jpeg": func(f *os.File, img image.Image) error { return jpeg.Encode(f, img, nil) },
		"png":  func(f *os.File, img image.Image) error { return png.Encode(f, img) },
		"gif":  func(f *os.File, img image.Image) error { return gif.Encode(f, img, nil) },
	}
	for format, encode := range encoders {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input")
			f, err := os.Create(input)
			if err != nil {
				t.Fatal(err)
			}
			img := image.NewRGBA(image.Rect(0, 0, 300, 150))
			for y := 0; y < 150; y++ {
				for x := 0; x < 300; x++ {
					img.Set(x, y, color.RGBA{200, 50, 20, 255})
				}
			}
			if err := encode(f, img); err != nil {
				t.Fatal(err)
			}
			f.Close()
			for kind, want := range map[string]image.Point{"icopy": {300, 150}, "ithumb": {192, 96}, "ilg": {300, 150}} {
				output := filepath.Join(dir, kind)
				if err := transformImage(kind, input, output); err != nil {
					t.Fatal(err)
				}
				out, err := os.Open(output)
				if err != nil {
					t.Fatal(err)
				}
				result, gotFormat, err := image.Decode(out)
				out.Close()
				if err != nil {
					t.Fatal(err)
				}
				if gotFormat != format || result.Bounds().Size() != want {
					t.Errorf("%s: format %s size %v, want %s %v", kind, gotFormat, result.Bounds().Size(), format, want)
				}
			}
		})
	}
}
