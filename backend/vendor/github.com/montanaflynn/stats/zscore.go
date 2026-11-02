package stats

// ZScore standardizes the input values by subtracting the mean
// and dividing by the sample standard deviation, returning the
// number of standard deviations each value is from the mean.
func ZScore(input Float64Data) ([]float64, error) {
	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	m, _ := Mean(input)
	sd, _ := StandardDeviationSample(input)

	if sd == 0 {
		return nil, ErrZero
	}

	z := make([]float64, len(input))
	for i, v := range input {
		z[i] = (v - m) / sd
	}
	return z, nil
}

// ZScore standardizes the input values by subtracting the mean
// and dividing by the sample standard deviation
func (f Float64Data) ZScore() ([]float64, error) {
	return ZScore(f)
}
