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

	return weightedSum / totalWeight, nil
}

// WeightedMean finds the weighted mean of the data using the given weights
func (f Float64Data) WeightedMean(weights Float64Data) (float64, error) {
	return WeightedMean(f, weights)
}
