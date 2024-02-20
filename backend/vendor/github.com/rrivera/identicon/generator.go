package identicon

import (
	"crypto/sha256"
	"errors"
	"image/color"
	"math/rand"
	"strconv"
	"time"
)

// Generator represents a predefined set of configurations that can be reused to
// create multiple icons by passing a Text string only.
type Generator struct {
	// Namespace that will be concatenated previous to the icon generation.
	Namespace string
	// Size is the number of blocks of the figure.
	Size int
	// Density * Size = times to iterate over the hash of Text:Namespace:Seed.
	Density int
	// hashFunction used to generate a fixed length array of bytes.
	hashFunction func([]byte) []byte
	// fillColorFunction used to pick a color to fill the squares of the figure.
	fillColorFunction func([]byte) color.Color
	// backgroundColorFunction used to pick a background color for the figure.
	backgroundColorFunction func([]byte, color.Color) color.Color
	// isRandom flag to decide whether the generated image will be randomized.
	isRandom bool
	// rand is the source of randomness.
	rand *rand.Rand
}

// option Configuration functional approach
type option func(*Generator)

// New returns a pointer to a Generator with the desired configuration.
func New(
	namespace string,
	size int,
	density int,
	opts ...option,
) (*Generator, error) {

	if size < MinSize {
		// Smaller values will generate a meaningless Generator.
		return nil, errors.New(
			"Size cannot be less than " + strconv.Itoa(MinSize),
		)
	}

	if density < 1 {
		return nil, errors.New(
			"Density cannot be less than 1",
		)
	}

	g := Generator{
		Size:                    size,
		Namespace:               namespace,
		Density:                 density,
		isRandom:                false,
		hashFunction:            _sha256,
		fillColorFunction:       _fillColor,
		backgroundColorFunction: _backgroundColor,
	}

	g.Option(opts...)

	return &g, nil
}

// Draw returns a pointer to an IdentIcon with a generated figure and a color.
func (g Generator) Draw(text string) (*IdentIcon, error) {

	var randomGenerator *rand.Rand

	if g.isRandom {
		// In order to generate a randomized Canvas, use UnixNano as the source
		// of randomess. By doing this, random numbers won't be consistent.
		randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		// rand will generate consistent values since the source is Size.
		randomGenerator = rand.New(rand.NewSource(int64(g.Size)))
	}

	ii, err := newIdentIcon(
		text,
		g.Namespace,
		g.Size,
		g.Density,
		g.isRandom,
		randomGenerator,
		g.hashFunction,
		g.fillColorFunction,
		g.backgroundColorFunction,
	)

	if err != nil {
		return nil, err
	}

	// Generate Canvas
	ii.Draw()

	return ii, nil
}

// Option sets the options specified.
func (g *Generator) Option(opts ...option) {
	for _, opt := range opts {
		opt(g)
	}
}

// SetHashFunction replaces the default hash function (Sha256).
func SetHashFunction(hf func([]byte) []byte) option {
	return func(g *Generator) {
		g.hashFunction = hf
	}
}

// SetFillColorFunction replaces the default color generation function (HSL).
func SetFillColorFunction(fcf func([]byte) color.Color) option {
	return func(g *Generator) {
		g.fillColorFunction = fcf
	}
}

// SetBackgroundColorFunction replaces the default background's color generation
// function (HSL).
func SetBackgroundColorFunction(bcf func([]byte, color.Color) color.Color) option {
	return func(g *Generator) {
		g.backgroundColorFunction = bcf
	}
}

// SetRandom to append a random string to the generator text everytime Draw is
// called.
func SetRandom(r bool) option {
	return func(g *Generator) {
		g.isRandom = r
	}
}

func _sha256(b []byte) []byte {
	digest := sha256.Sum256(b)
	return digest[:]
}

func _fillColor(hashBytes []byte) color.Color {
	cb1, cb2 := uint32(hashBytes[0]), uint32(hashBytes[1])
	h := (cb1 + cb2) % 360
	s := (cb1 % 30) + 60
	l := (cb2 % 20) + 40

	// Some colors in the HSL color model are too bright and don't play well
	// with the default background color. This is a naïve normalization method.
	if (h >= 50 && h <= 85) || (h >= 170 && h <= 190) {
		s = 80
		l -= 20
	} else if h > 85 && h < 170 {
		l -= 10
	}

	return HSL{h, s, l}
}

func _backgroundColor(hashBytes []byte, fill color.Color) color.Color {
	return color.NRGBA{R: 240, G: 240, B: 240, A: 255}
}
