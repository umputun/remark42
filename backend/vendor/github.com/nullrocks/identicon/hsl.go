package identicon

// Identicon WebColor maxium values.
const (
	// hueMax is the maximum allowed value for Hue in the HSL color model.
	hueMax = 360
	// saturationMax is the maximum allowed value for Saturation in the HSL
	// color model.
	saturationMax = 100
	// lightnessMax is the maximum allowed value for lightnessMax in the HSL
	// color model.
	lightnessMax = 100
	// rgbaMax is the maximum allowed value for any R, G, B, A property value.
	rgbaMax = 255
)

// HSL is a color model representation based on RGB. HSL facilitates the
// generation of colors that look similar between themselves by changing the
// value of Hue H while keeping Saturation S and Lightness L the same.
type HSL struct {
	// Hue [0, 360]
	H uint32
	// Saturation [0, 100]
	S uint32
	// Lightness [0, 100]
	L uint32
}

// RGBA conversion
func (hsl HSL) RGBA() (r, g, b, a uint32) {
	h := 1.0 / float64(hueMax) * float64(hsl.H)
	s := float64(hsl.S) / float64(saturationMax)
	l := float64(hsl.L) / float64(lightnessMax)
	r, g, b = hslToRgb(h, s, l)
	a = rgbaMax
	r |= r << 8
	g |= g << 8
	b |= b << 8
	a |= a << 8
	return
}

// Golang port of Mohen's code in Stack Overflow.
// https://stackoverflow.com/questions/2353211/hsl-to-rgb-color-conversion
func hslToRgb(h, s, l float64) (uint32, uint32, uint32) {
	var q, p float64
	var r, g, b float64

	if s == 0 {
		r = l
		g = l
		b = l
	} else {
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = (l + s) - (l * s)
		}
		p = (2 * l) - q
		r = hueToRgb(p, q, h+(1.0/3.0))
		g = hueToRgb(p, q, h)
		b = hueToRgb(p, q, h-(1.0/3.0))
	}

	return uint32(r * rgbaMax), uint32(g * rgbaMax), uint32(b * rgbaMax)
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t++
	} else if t > 1 {
		t--
	}
	switch {
	case 6*t < 1:
		return (p + (q-p)*6*t)
	case 2*t < 1:
		return q
	case 3*t < 2:
		return p + (q-p)*((2.0/3.0)-t)*6
	}
	return p
}
