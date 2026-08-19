package stats

import "math"

// CoefficientOfVariation finds the coefficient of variation of a slice
// of floats, defined as the sample standard deviation divided by the
// mean. This matches the behavior of Python's scipy.stats.variation
// with ddof=1.
//
// The input must not be empty and its mean must not be zero.
func CoefficientOfVariation(input Float64Data) (float64, error) {
	if input.Len() == 0 {
		return math.NaN(), ErrEmptyInput
	}

	// Input is known to be non-empty so the mean and sample
	// standard deviation cannot return an error
	m, _ := Mean(input)
	if m == 0 {
		return math.NaN(), ErrZero
	}

	sd, _ := StandardDeviationSample(input)

	return sd / m, nil
}

// CoefficientOfVariation finds the sample standard deviation divided by the mean
func (f Float64Data) CoefficientOfVariation() (float64, error) {
	return CoefficientOfVariation(f)
}
