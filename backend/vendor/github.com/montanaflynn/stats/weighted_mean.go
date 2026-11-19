package stats

import "math"

// WeightedMean finds the weighted mean of a slice of floats, defined as
// the sum of each data value multiplied by its weight divided by the sum
// of all the weights. This matches the behavior of Python's
// numpy.average with the weights argument.
//
// The data and weights slices must be the same length. Weights must be
// non-negative and at least one weight must be positive.
func WeightedMean(data, weights Float64Data) (float64, error) {
	l := data.Len()
	if l == 0 {
		return math.NaN(), ErrEmptyInput
	}

	if weights.Len() != l {
		return math.NaN(), ErrSize
	}

	weightedSum := 0.0
	totalWeight := 0.0
	for i := 0; i < l; i++ {
		if weights[i] < 0 {
			return math.NaN(), ErrNegative
		}
		weightedSum += data[i] * weights[i]
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return math.NaN(), ErrZero
	}

	mean := weightedSum / totalWeight

	// Products or sums of finite values near math.MaxFloat64 can overflow
	// even though the mean cannot. Weight half of each value by its share
	// of the total instead, with the weights scaled down by a power of two
	// at least the length so their sum stays finite.
	if math.IsInf(mean, 0) || math.IsNaN(mean) {
		_, e := math.Frexp(float64(l))
		total := 0.0
		for _, w := range weights {
			total += math.Ldexp(w, -e)
		}
		half, lo, hi := 0.0, math.Inf(1), math.Inf(-1)
		for i := 0; i < l; i++ {
			if weights[i] > 0 {
				lo = math.Min(lo, data[i]/2)
				hi = math.Max(hi, data[i]/2)
			}
			half += data[i] / 2 * (math.Ldexp(weights[i], -e) / total)
		}
		// Rounding the shares can carry the sum just past the weighted
		// values, and so past math.MaxFloat64 once doubled.
		mean = 2 * math.Max(lo, math.Min(half, hi))
	}

	return mean, nil
}

// WeightedMean finds the weighted mean of the data using the given weights
func (f Float64Data) WeightedMean(weights Float64Data) (float64, error) {
	return WeightedMean(f, weights)
}
