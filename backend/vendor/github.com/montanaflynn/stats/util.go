package stats

import (
	"math"
	"sort"
	"time"
)

// float64ToInt rounds a float64 to an int
func float64ToInt(input float64) (output int) {
	r, _ := Round(input, 0)
	return int(r)
}

// unixnano returns nanoseconds from UTC epoch
func unixnano() int64 {
	return time.Now().UTC().UnixNano()
}

// copyslice copies a slice of float64s
func copyslice(input Float64Data) Float64Data {
	s := make(Float64Data, input.Len())
	copy(s, input)
	return s
}

// hasNaN reports whether the input contains a NaN, which sorting
// places below every number and comparisons silently skip
func hasNaN(input Float64Data) bool {
	for _, v := range input {
		if math.IsNaN(v) {
			return true
		}
	}
	return false
}

// unitExponent returns e such that the largest magnitude in the input
// times 2^-e is in [0.5, 1), or 0 when that magnitude is 0, Inf or NaN.
// Multiplying by 2^-e is exact, so powers of the scaled values cannot
// overflow or underflow and results can be scaled back. e is at least
// -1021 so that 2^-e is finite.
func unitExponent(input Float64Data) int {
	var max float64
	for _, v := range input {
		max = math.Max(max, math.Abs(v))
	}
	_, e := math.Frexp(max)
	if e < -1021 {
		e = -1021
	}
	return e
}

// sortedCopy returns a sorted copy of float64s
func sortedCopy(input Float64Data) (copy Float64Data) {
	copy = copyslice(input)
	sort.Float64s(copy)
	return
}

// sortedCopyDif returns a sorted copy of float64s
// only if the original data isn't sorted.
// Only use this if returned slice won't be manipulated!
func sortedCopyDif(input Float64Data) (copy Float64Data) {
	if sort.Float64sAreSorted(input) {
		return input
	}
	copy = copyslice(input)
	sort.Float64s(copy)
	return
}
