package stats

// ZScore standardizes the input values by subtracting the mean
// and dividing by the sample standard deviation, returning the
// number of standard deviations each value is from the mean. A
// single element returns ErrTooFew and constant input returns ErrZero.
func ZScore(input Float64Data) ([]float64, error) {
	sd, err := StandardDeviationSample(input)
	if err != nil {
		return nil, err
	}

	m, _ := Mean(input)

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
