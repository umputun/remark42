package identicon

import (
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
)

// Image genetares an image.Image of size
func (ii *IdentIcon) Image(pixels int) image.Image {

	// Padding is relative to the number of blocks.
	padding := pixels / (ii.Size * MinSize)
	drawableArea := pixels - (padding * 2)
	blockSize := drawableArea / ii.Size

	// Add the residue (pixels that won't be filled) to the padding.
	// Try to center the figure regardless when the drawable area is not
	// divisible by the block pixels.
	padding += (drawableArea % ii.Size) / 2

	img := image.NewNRGBA(image.Rect(0, 0, pixels, pixels))

	// Background
	draw.Draw(
		img,
		img.Bounds(),
		&image.Uniform{ii.BackgroundColor},
		image.ZP,
		draw.Src,
	)

	for y, mapX := range ii.Canvas.PointsMap {
		for x := range mapX {

			ix := blockSize*x + padding
			iy := blockSize*y + padding

			draw.Draw(img,
				image.Rect(
					ix,
					iy,
					ix+blockSize,
					iy+blockSize,
				),
				&image.Uniform{ii.FillColor},
				image.ZP,
				draw.Src,
			)
		}
	}

	return img
}

// Png writes an image of pixels
func (ii *IdentIcon) Png(pixels int, w io.Writer) error {
	img := ii.Image(pixels)
	return png.Encode(w, img)
}

// Jpeg writes an image of pixels and quality
func (ii *IdentIcon) Jpeg(pixels int, quality int, w io.Writer) error {
	img := ii.Image(pixels)
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

// Svg writes an image of pixels
func (ii *IdentIcon) Svg(pixels int, w io.Writer) error {
	return svgEncode(w, ii, pixels)
}
