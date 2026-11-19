package stats

import "math"

// PercentileOfScore calculates the percentile rank of a score
// relative to a slice of floats, defined as the percentage of
// values strictly below the score plus half the percentage of
// values equal to the score. The result is between 0 and 100.
// This matches the behavior of Python's
// scipy.stats.percentileofscore with kind="mean". A NaN score
// returns ErrBounds, and if the input contains a NaN the result
// is NaN.
func PercentileOfScore(input Float64Data, score float64) (float64, error) {
	if input.Len() == 0 {
		return math.NaN(), ErrEmptyInput
	}

	if math.IsNaN(score) {
		return math.NaN(), ErrBounds
	}

	// A NaN is neither below, equal to, nor above the score
	if hasNaN(input) {
		return math.NaN(), nil
	}

	var below, equal float64
	for _, v := range input {
		if v < score {
			below++
		} else if v == score {
			equal++
		}
	}

	return 100 * (below + 0.5*equal) / float64(input.Len()), nil
}

// PercentileOfScore calculates the percentile rank of a score relative to the data
func (f Float64Data) PercentileOfScore(score float64) (float64, error) {
	return PercentileOfScore(f, score)
}
