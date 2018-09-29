package identicon

import (
	"image/color"
	"io"
	"strconv"
	"text/template"
)

type svgRect struct {
	X         int
	Y         int
	Width     int
	Height    int
	FillColor string
}

type svgTmpl struct {
	Pixels          int
	BackgroundColor string
	FillColor       string
	Rects           []svgRect
}

const svgTemplate = `
<!--   <?xml version="1.0"?> -->
<svg version="1.1" baseprofile="full" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:ev="http://www.w3.org/2001/xml-events" width="{{.Pixels}}" height="{{.Pixels}}" viewBox="0 0 {{.Pixels}} {{.Pixels}}" >
	<rect width="100%" height="100%" fill="{{.BackgroundColor}}"/>
{{range $index, $r := .Rects}}
	<rect x="{{$r.X}}" y="{{$r.Y}}" width="{{$r.Width}}" height="{{$r.Height}}" fill="{{$r.FillColor}}"/>{{end}}
</svg>`

func colorToRGBAString(c color.Color) string {
	r, g, b, a := c.RGBA()
	r >>= 8
	g >>= 8
	b >>= 8
	a >>= 8

	rs := strconv.Itoa(int(r))
	gs := strconv.Itoa(int(g))
	bs := strconv.Itoa(int(b))
	as := strconv.Itoa(int(a))

	return "rgba(" + rs + "," + gs + "," + bs + "," + as + ")"
}

// Encode an IdentIcon to SVG
func svgEncode(w io.Writer, ii *IdentIcon, pixels int) error {

	// Padding is relative to the number of blocks.
	padding := pixels / (ii.Size * MinSize)
	drawableArea := pixels - (padding * 2)
	blockSize := drawableArea / ii.Size

	// Add the residue (pixels that won't be filled) to the padding.
	// Try to center the figure regardless when the drawable area is not
	// divisible by the block pixels.
	padding += (drawableArea % ii.Size) / 2

	fillColor := colorToRGBAString(ii.FillColor)
	backgroundColor := colorToRGBAString(ii.BackgroundColor)

	b, err := template.New("svg").Parse(svgTemplate)

	if err != nil {
		return err
	}

	t := svgTmpl{
		Pixels:          pixels,
		BackgroundColor: backgroundColor,
		FillColor:       fillColor,
		Rects:           make([]svgRect, ii.Canvas.FilledPoints),
	}

	i := 0
	for y, mapX := range ii.Canvas.PointsMap {
		for x := range mapX {
			t.Rects[i] = svgRect{
				blockSize*x + padding,
				blockSize*y + padding,
				blockSize,
				blockSize,
				fillColor,
			}
			i++
		}
	}

	return b.Execute(w, t)
}
