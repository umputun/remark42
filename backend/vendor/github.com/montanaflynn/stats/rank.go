package stats

import "math"

// Rank assigns fractional (average) ranks to the input values.
// Ranks are 1-based and tied values receive the average of the
// ranks they would have been assigned. Like SciPy's rankdata,
// if the input contains a NaN every rank is NaN.
func Rank(input Float64Data) ([]float64, error) {
	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}
	if hasNaN(input) {
		ranks := make([]float64, input.Len())
		for i := range ranks {
			ranks[i] = math.NaN()
		}
		return ranks, nil
	}
	return rankData(input), nil
}

// Rank assigns fractional (average) ranks to the input values
func (f Float64Data) Rank() ([]float64, error) {
	return Rank(f)
}
