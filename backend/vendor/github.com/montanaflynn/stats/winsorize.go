package stats

import "math"

// Winsorize limits the effect of outliers in a slice of floats by
// clamping a fraction of the smallest and largest values. This matches
// the behavior of Python's scipy.stats.mstats.winsorize with symmetric
// limits.
//
// The percent parameter is the fraction clamped in each tail and must
// be in the range [0, 0.5). With k = floor(len(input) * percent),
// values below the k-th smallest value are set to it and values above
// the k-th largest value are set to it. The returned slice preserves
// the original element order and a percent of zero returns a copy of
// the input.
func Winsorize(input Float64Data, percent float64) ([]float64, error) {
	l := input.Len()
	if l == 0 {
		return nil, ErrEmptyInput
	}

	// Reject percents outside [0, 0.5) including NaN
	if !(percent >= 0 && percent < 0.5) {
		return nil, ErrBounds
	}

	sorted := sortedCopy(input)

	// Number of elements clamped in each tail
	k := int(math.Floor(float64(l) * percent))
	lower := sorted[k]
	upper := sorted[l-1-k]

	output := copyslice(input)
	for i, v := range output {
		if v < lower {
			output[i] = lower
		} else if v > upper {
			output[i] = upper
		}
	}

	return output, nil
}

// Winsorize returns a copy of the data with a fraction of the smallest
// and largest values in each tail clamped
func (f Float64Data) Winsorize(percent float64) ([]float64, error) {
	return Winsorize(f, percent)
}
