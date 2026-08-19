package stats

// CumulativeProduct calculates the cumulative product of the input slice
func CumulativeProduct(input Float64Data) ([]float64, error) {

	if input.Len() == 0 {
		return Float64Data{}, ErrEmptyInput
	}

	cumProduct := make([]float64, input.Len())

	for i, val := range input {
		if i == 0 {
			cumProduct[i] = val
		} else {
			cumProduct[i] = cumProduct[i-1] * val
		}
	}

	return cumProduct, nil
}

// CumulativeMax calculates the cumulative maximum of the input slice
func CumulativeMax(input Float64Data) ([]float64, error) {

	if input.Len() == 0 {
		return Float64Data{}, ErrEmptyInput
	}

	cumMax := make([]float64, input.Len())

	for i, val := range input {
		if i == 0 || val > cumMax[i-1] {
			cumMax[i] = val
		} else {
			cumMax[i] = cumMax[i-1]
		}
	}

	return cumMax, nil
}

// CumulativeMin calculates the cumulative minimum of the input slice
func CumulativeMin(input Float64Data) ([]float64, error) {

	if input.Len() == 0 {
		return Float64Data{}, ErrEmptyInput
	}

	cumMin := make([]float64, input.Len())

	for i, val := range input {
		if i == 0 || val < cumMin[i-1] {
			cumMin[i] = val
		} else {
			cumMin[i] = cumMin[i-1]
		}
	}

	return cumMin, nil
}

// CumulativeProduct calculates the cumulative product of the data
func (f Float64Data) CumulativeProduct() ([]float64, error) {
	return CumulativeProduct(f)
}

// CumulativeMax calculates the cumulative maximum of the data
func (f Float64Data) CumulativeMax() ([]float64, error) {
	return CumulativeMax(f)
}

// CumulativeMin calculates the cumulative minimum of the data
func (f Float64Data) CumulativeMin() ([]float64, error) {
	return CumulativeMin(f)
}
