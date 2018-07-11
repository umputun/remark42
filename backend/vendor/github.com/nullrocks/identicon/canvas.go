package identicon

import (
	"image"
	"strconv"
)

// Canvas contains what is needed to generate an image. It contains properties
// that could be useful when rendering the image.
//  - Having MinY and MaxY allows you to vertically center the figure.
//  - VisitedYPoints could be useful to determine whether there is a big empty
//  vertical space in the figure.
type Canvas struct {
	// Size same value specified in identicon.New(...).
	Size int
	// PointsMap contains all coordinates and it's values that form the figure.
	PointsMap map[int]map[int]int
	// MinY is the upper Y-axis that has at least one point drawn.
	MinY int
	// MaxY is the lower Y-axis that has at least one point drawn.
	MaxY int
	// VisitedYPoints contains all Y-axis that had been visited. Helpful to
	// determine big blank spaces in the resulting figure.
	VisitedYPoints map[int]bool
	// FilledPoints is the number of points filled at least once.
	FilledPoints int
}

// Array generates a two-dimensional array version of the IdentIcon figure.
func (c *Canvas) Array() [][]int {
	canvasArray := make([][]int, c.Size)
	for i := range canvasArray {
		canvasArray[i] = make([]int, c.Size)
	}

	for y := range c.PointsMap {
		for x := range c.PointsMap[y] {
			canvasArray[y][x] = c.PointsMap[y][x]
		}
	}

	return canvasArray
}

// ToString generates a string version of the IdentIcon figure.
func (c *Canvas) String(separator string, fillEmptyWith string) string {
	tp := c.Size * c.Size
	// Total number of characters considering:
	strLen := c.Size - 1                                 // Line Breaks
	strLen += (tp - c.Size) * len(separator)             // Separators
	strLen += c.FilledPoints                             // Points
	strLen += (tp - c.FilledPoints) * len(fillEmptyWith) // Fill Empty

	// Concatenating strings with the `+` is slow and uses a lot of memory,
	// using `copy` in a slice of bytes has been proved to be a better approach.
	bs := make([]byte, strLen)
	// Keep track of the length of the bytes array, concatenations will occur
	// using `bl` as the right-most index.
	bl := 0

	for y := 0; y < c.Size; y++ {
		if mapY, exists := c.PointsMap[y]; exists {
			for x := 0; x < c.Size; x++ {
				if value, exists := mapY[x]; exists {
					if value > 9 {
						value = 9
					}
					bl += copy(bs[bl:], []byte(strconv.Itoa(value)))
				} else {
					bl += copy(bs[bl:], []byte(fillEmptyWith))
				}
				if x < c.Size-1 {
					bl += copy(bs[bl:], []byte(separator))
				}
			}
		} else {
			// There aren't any values in this row, fill it the row anyway.
			for x := 0; x < c.Size; x++ {
				bl += copy(bs[bl:], []byte(fillEmptyWith))
				if x < c.Size-1 {
					bl += copy(bs[bl:], []byte(separator))
				}
			}
		}

		if y < c.Size-1 {
			// Append a line break except when it's the last line.
			bl += copy(bs[bl:], "\n")
		}
	}

	return string(bs)
}

// Points generates an array of points of a two-dimensional plane as [x, y]
// that correspond to all filled points in the IdentIcon figure.
func (c *Canvas) Points() []image.Point {
	points := []image.Point{}

	for y, value := range c.PointsMap {
		for x := range value {
			points = append(points, image.Point{X: x, Y: y})
		}
	}

	return points
}

// IntCoordinates generates an array of points of a two-dimensional plane as:
//  - [x, y] that correspond to all filled points in the IdentIcon figure.
func (c *Canvas) IntCoordinates() [][]int {
	points := [][]int{}

	for y, value := range c.PointsMap {
		for x := range value {
			points = append(points, []int{x, y})
		}
	}

	return points
}
