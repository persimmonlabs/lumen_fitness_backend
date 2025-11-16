package storage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// CreateTestPNG creates a simple test image in PNG format
func CreateTestPNG() []byte {
	// Create a simple 100x100 PNG image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Fill with a simple pattern
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 2),
				G: uint8(y * 2),
				B: 100,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}
