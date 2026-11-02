package stats

import "math"

// TrimmedMean finds the mean of a slice of floats after removing a
// fraction of the smallest and largest values. This matches the
// behavior of Python's scipy.stats.trim_mean.
//
// The percent parameter is the fraction removed from each tail and
// must be in the range [0, 0.5). The number of elements trimmed from
// each tail is floor(len(input) * percent). A percent of zero returns
// the same result as Mean.
func TrimmedMean(input Float64Data, percent float64) (float64, error) {
	l := input.Len()
	if l == 0 {
		return math.NaN(), ErrEmptyInput
	}

	// Reject percents outside [0, 0.5) including NaN. Since percent is
	// strictly below 0.5 at least one element always remains after
	// trimming floor(l * percent) elements from each tail.
	if !(percent >= 0 && percent < 0.5) {
		return math.NaN(), ErrBounds
	}

	sorted := sortedCopy(input)

	// Number of elements removed from each tail
	k := int(math.Floor(float64(l) * percent))

	return Mean(sorted[k : l-k])
}

// TrimmedMean finds the mean of the data after removing a fraction of
// the smallest and largest values from each tail
func (f Float64Data) TrimmedMean(percent float64) (float64, error) {
	return TrimmedMean(f, percent)
}
