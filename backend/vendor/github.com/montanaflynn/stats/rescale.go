package stats

// Rescale normalizes the input values to the range of 0 to 1
// by subtracting the minimum and dividing by the range,
// also known as min-max normalization.
func Rescale(input Float64Data) ([]float64, error) {
	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	min, _ := Min(input)
	max, _ := Max(input)

	if max == min {
		return nil, ErrZero
	}

	r := make([]float64, len(input))
	for i, v := range input {
		r[i] = (v - min) / (max - min)
	}
	return r, nil
}

// Rescale normalizes the input values to the range of 0 to 1
// by subtracting the minimum and dividing by the range
func (f Float64Data) Rescale() ([]float64, error) {
	return Rescale(f)
}
