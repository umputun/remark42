package stats

import (
	"math"
	"sort"
)

// Histogram calculates the histogram of a slice using the given
// number of equal-width bins over [min, max], returning the count
// of values in each bin along with the bins+1 bin edges. Each bin
// is half-open [edges[i], edges[i+1]) except the last, which also
// includes the maximum value. A NaN cannot be placed in any
// bin, so input containing a NaN returns ErrNaN.
func Histogram(input Float64Data, bins int) ([]int, []float64, error) {

	if input.Len() == 0 {
		return nil, nil, ErrEmptyInput
	}

	if bins < 1 {
		return nil, nil, ErrBounds
	}

	if hasNaN(input) {
		return nil, nil, ErrNaN
	}

	// Disregard errors, since input is not empty
	min, _ := Min(input)
	max, _ := Max(input)

	// Expand the range by 0.5 on each side like
	// numpy when all of the values are equal
	if min == max {
		min -= 0.5
		max += 0.5
	}

	// Build bins+1 equal-width bin edges from min to max
	width := (max - min) / float64(bins)
	edges := make([]float64, bins+1)
	for i := range edges {
		edges[i] = min + width*float64(i)
	}

	// When max - min overflows, build the edges from halved
	// bounds, which is exact for values this large, and double
	// them back so the edges stay finite
	if math.IsInf(width, 1) {
		width = (max/2 - min/2) / float64(bins)
		for i := range edges {
			edges[i] = 2 * (min/2 + width*float64(i))
		}
	}
	edges[bins] = max

	// Count each value into the last bin whose left
	// edge does not exceed it, so the maximum value
	// lands in the final closed bin
	counts := make([]int, bins)
	for _, v := range input {
		i := sort.Search(len(edges), func(i int) bool { return edges[i] > v }) - 1
		if i == bins {
			i--
		}
		counts[i]++
	}

	return counts, edges, nil
}

// Histogram returns the counts and equal-width bin edges of the data
func (f Float64Data) Histogram(bins int) ([]int, []float64, error) {
	return Histogram(f, bins)
}
