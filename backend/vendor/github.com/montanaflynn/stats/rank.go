package stats

// Rank assigns fractional (average) ranks to the input values.
// Ranks are 1-based and tied values receive the average of the
// ranks they would have been assigned.
func Rank(input Float64Data) ([]float64, error) {
	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}
	return rankData(input), nil
}

// Rank assigns fractional (average) ranks to the input values
func (f Float64Data) Rank() ([]float64, error) {
	return Rank(f)
}
