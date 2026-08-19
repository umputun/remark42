package stats

import "math"

// RMS calculates the root mean square of a slice of floats,
// defined as the square root of the mean of the squared values.
func RMS(input Float64Data) (float64, error) {
	if input.Len() == 0 {
		return math.NaN(), ErrEmptyInput
	}

	var sumSquares float64
	for _, v := range input {
		sumSquares += v * v
	}

	return math.Sqrt(sumSquares / float64(input.Len())), nil
}

// RMS calculates the root mean square of the data
func (f Float64Data) RMS() (float64, error) {
	return RMS(f)
}
