// Package identicon is an open source avatar generator inspired by GitHub avatars.
//
// IdentIcon uses a deterministic algorithm that generates an image (using Golang's
// stdlib image encoders) based on a text (Generally Usernames, Emails or just
// random strings), by hashing it and iterating over the bytes of the digest to pick
// whether to draw a point, pick a color or choose where to go next.
//
// IdentIcon's Generator enables the creation of customized figures: (NxN size,
// points density, custom color palette) as well as multiple exporting formats in
// case the developers want to generate their own images.
package identicon

import (
	"errors"
	"image"
	"image/color"
	"math/rand"
	"strconv"
)

const (
	// Bits used to give continuity
	moveUp    = 0x80
	moveDown  = 0x40
	moveLeft  = 0x20
	moveRight = 0x10

	// Either 0x8 or 0x2 are active
	fillPoint = 0xA
)

// Constrains for the size of the IdentIcon.
const (
	// MinSize is the minimal number of blocks allowed, anything lower that this
	// wouldn't make sense.
	MinSize = 4
)

// IdentIcon represents a mirror-symmetry image generated from a string and a
// set of configurations.
type IdentIcon struct {
	// Text is the base string that will generate the canvas after being hashed.
	Text string
	// Namespace
	Namespace string

	// Size is the number of blocks of the figure.
	Size int
	// Density * Size = times to iterate over the hash of Text.
	Density int
	// Canvas is a map of maps that contains the points and values that has been
	// visited and filled.
	Canvas Canvas

	// FillColor is the color used to fill squares in the figure when encoding
	// to PNG or JPEG.
	FillColor color.Color
	// BackgroundColor is the background color of the figure when encoding it to
	// PNG or JPEG.
	BackgroundColor color.Color
	// fillColorFunction used to pick a color to fill the squares of the figure.
	fillColorFunction func([]byte) color.Color
	// backgroundColorFunction used to pick a background color for the figure.
	backgroundColorFunction func([]byte, color.Color) color.Color

	// drawableWidth represents the length of the left half of the canvas.
	drawableWidth int
	// hasBeenDrawn indicates whether the Draw() has been called before.
	hasBeenDrawn bool

	// hashFunction used to generate a fixed length array of bytes.
	hashFunction func([]byte) []byte

	// isRandom flag to decide whether the generated image will be randomized.
	isRandom bool
	// randomSeed
	randomSeed string
	// rand is the source of randomness.
	rand *rand.Rand
}

// Draw a figure in Canvas.
//  - If isRandom == true, the figure will redrawn everytime Draw() is called,
//  - If isRandom == false and Draw() was called before, it won't redraw.
func (ii *IdentIcon) Draw() {

	if ii.hasBeenDrawn && !ii.isRandom {
		// Don't redraw once twice unless isRandom is enabled.
		return
	} else if ii.isRandom {
		// Set a new randomSeed everytime Draw is executed to produce different
		// results on each execution.
		ii.randomSeed = strconv.Itoa(ii.rand.Int())
	}

	ii.hasBeenDrawn = true

	// Make sure that the canvas has been initialized.
	ii.initCanvas()

	// current index of the digested bytes array.
	var i int
	// Number of bytes readed.
	var readedBytes int
	// Flag to know whether it as completed a full cycle.
	var hasCompletedCycle bool
	// Position that represents a point in the canvas.
	var current image.Point

	// Text:Namespace:randomSeed
	generatingBytes := []byte(ii.GeneratorText())

	// Produce fixed-length array of bytes that will be used to control the
	// drawing process.
	hashBytes := ii.hashFunction(generatingBytes)
	hashBytesLen := len(hashBytes)

	ii.FillColor = ii.fillColorFunction(hashBytes)
	ii.BackgroundColor = ii.backgroundColorFunction(hashBytes, ii.FillColor)

	// Total number of iterations over the digested hash.
	bytesToRead := ii.Density * ii.Size

	for {
		if hasCompletedCycle {
			// If the number of bytes to read exceeds the length of the hash,
			// it will cycle through it. After it has completed a whole cycle,
			// altering the value will produce more varied figures.
			//
			// XOR pseudo-random produces interesting results.
			hashBytes[i] ^= byte(ii.rand.Intn(255))
		}

		if i == 0 {
			// Everytime a new cycle is starting, change the current point to
			// cover multiple areas of the canvas.
			current = initialPoint(
				hashBytes[0],
				ii.rand.Intn(ii.drawableWidth),
				ii.rand.Intn(ii.Size),
			)
		}

		// value to add in the current point, zeroes will be ignored.
		value := getFillValue(hashBytes[i])

		if value != 0 {
			// Initialize the map for Y-axis, making sure that the map that
			// contains X-axis values won't be nil.
			createMapIfDoesntExist(&ii.Canvas, current.Y)

			firstTimeFilled := false
			if ii.Canvas.PointsMap[current.Y][current.X] == 0 {
				// Increment FilledPoints the first time this point is visited.
				ii.Canvas.FilledPoints++
				firstTimeFilled = true
			}

			// Add the value to current position
			ii.Canvas.PointsMap[current.Y][current.X] += value

			// Mark Y value as visited. This will be helpful to determine big
			// blank spaces in the resulting figure.
			ii.Canvas.VisitedYPoints[current.Y] = true

			// Update the maximum and minimum Y-axis values, useful to
			// vertically center the figure at image creation.
			if current.Y < ii.Canvas.MinY {
				ii.Canvas.MinY = current.Y
			}
			if current.Y > ii.Canvas.MaxY {
				ii.Canvas.MaxY = current.Y
			}

			// When Size is an odd number, prevent points in the middle to be
			// added twice. By substrating oddDiff to drawableWidth we make sure
			// that it doesn't happens.
			oddDiff := ii.Size % 2
			if current.X < (ii.drawableWidth - oddDiff) {
				// Calculate the mirror position for X-axis
				mirror := mirrorSymmetric(current, ii.Size)
				// Add value to the mirrowed position
				ii.Canvas.PointsMap[mirror.Y][mirror.X] += value
				if firstTimeFilled {
					ii.Canvas.FilledPoints++
				}
			}
		}

		// Decide the next position relative to the current position.
		current = nextPoint(hashBytes[i], current, ii.drawableWidth, ii.Size)

		i++
		readedBytes++

		if readedBytes >= bytesToRead {
			// The total number of bytes to read has been reached, stop.
			break
		}

		if i == hashBytesLen-1 {
			// A full cycle has been completed, reset the index to prevent
			// getting out of bounds.
			i = 0
			// Further iterations will add a pesudo-random number to hashBytes.
			hasCompletedCycle = true
		}
	}

}

// GeneratorText returns the string later to be hashed using the format:
//  - Text[:Namespace][:randomSeed]
func (ii *IdentIcon) GeneratorText() string {
	gt := ii.Text

	if ii.Namespace != "" {
		gt += ":" + ii.Namespace
	}

	if ii.isRandom && ii.randomSeed != "" {
		gt += ":" + ii.randomSeed
	}

	return gt
}

// Array generates a two-dimensional array version of the IdentIcon figure.
func (ii *IdentIcon) Array() [][]int {
	return ii.Canvas.Array()
}

// ToString generates a string version of the IdentIcon figure.
func (ii *IdentIcon) String(separator string, fillEmptyWith string) string {
	return ii.Canvas.String(separator, fillEmptyWith)
}

// Points generates an array of points of a two-dimensional plane as [x, y]
// that correspond to all filled points in the IdentIcon figure.
func (ii *IdentIcon) Points() []image.Point {
	return ii.Canvas.Points()
}

// IntCoordinates generates an array of points of a two-dimensional plane as:
//  - [x, y] that correspond to all filled points in the IdentIcon figure.
func (ii *IdentIcon) IntCoordinates() [][]int {
	return ii.Canvas.IntCoordinates()
}

// New returns a pointer to IdentIcon.
func newIdentIcon(
	text string,
	namespace string,
	size int,
	density int,
	isRandom bool,
	rand *rand.Rand,
	hashFunction func([]byte) []byte,
	fillColorFunction func([]byte) color.Color,
	backgroundColorFunction func([]byte, color.Color) color.Color,
) (*IdentIcon, error) {

	if text == "" {
		// Text is the minimum requirement to generate an IdentIcon.
		return nil, errors.New("Text can't be empty")
	}

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

	identicon := IdentIcon{
		Text:                    text,
		Namespace:               namespace,
		Size:                    size,
		Density:                 density,
		isRandom:                isRandom,
		rand:                    rand,
		hashFunction:            hashFunction,
		fillColorFunction:       fillColorFunction,
		backgroundColorFunction: backgroundColorFunction,
	}

	// Reflection Line
	identicon.drawableWidth = identicon.Size / 2

	// Since the canvas is a symmetrical reflection make sure to:
	//  - Handle even and odd Canvas sizes
	if identicon.Size%2 == 1 {
		// Is odd, the vertical middle point exist.
		identicon.drawableWidth++
	}

	return &identicon, nil
}

// initCanvas initializes and erases everything that was in the Canvas map.
func (ii *IdentIcon) initCanvas() {
	ii.Canvas = Canvas{
		Size:           ii.Size,
		PointsMap:      make(map[int]map[int]int),
		MinY:           ii.Size,
		MaxY:           0,
		VisitedYPoints: make(map[int]bool),
	}
}

func nextPoint(control byte, p image.Point, width, heigth int) image.Point {
	// Active bits will decide the destination of the next point.
	//  - If two opposite bits are active, it will keep its current position.
	if control&moveUp == moveUp {
		p.Y--
	}
	if control&moveDown == moveDown {
		p.Y++
	}
	if control&moveLeft == moveLeft {
		p.X--
	}
	if control&moveRight == moveRight {
		p.X++
	}

	// Transform to 0-based indices.
	width--
	heigth--

	// Teleport to opposite bounds when the limit has been reached.
	if p.X > width {
		p.X = 0
	} else if p.X < 0 {
		p.X = width
	}
	if p.Y > heigth {
		p.Y = 0
	} else if p.Y < 0 {
		p.Y = heigth
	}

	return p
}

func initialPoint(control byte, width, heigth int) image.Point {
	return image.Point{
		Y: heigth,
		X: width,
	}
}

func mirrorSymmetric(p image.Point, size int) image.Point {
	return image.Point{
		Y: p.Y,
		X: size - p.X - 1,
	}
}

func getFillValue(control byte) int {
	if control&fillPoint > 0 {
		return 1
	}
	return 0
}

func createMapIfDoesntExist(canvas *Canvas, y int) {
	_, exist := canvas.PointsMap[y]
	if !exist {
		canvas.PointsMap[y] = make(map[int]int)
	}
}
