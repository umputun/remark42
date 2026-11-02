package stats

import "math"

// Product calculates the product of a slice of floats by
// multiplying the values from left to right. It is the scalar
// counterpart of CumulativeProduct. Large inputs can overflow
// to Inf; use GeometricMean for an overflow-safe summary of
// multiplicative data.
func Product(input Float64Data) (float64, error) {
	if input.Len() == 0 {
		return math.NaN(), ErrEmptyInput
	}

	product := 1.0
	for _, v := range input {
		product *= v
	}

	return product, nil
}

// Product calculates the product of the data
func (f Float64Data) Product() (float64, error) {
	return Product(f)
}
