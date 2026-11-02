package stats

// EWMA calculates the exponentially weighted moving average of the input
// with smoothing factor alpha. The first output equals the first input and
// each subsequent entry is alpha*input[i] + (1-alpha)*output[i-1], so the
// result has the same length as the input. The alpha must satisfy
// 0 < alpha <= 1 or ErrBounds is returned. An empty input returns
// ErrEmptyInput.
func EWMA(input Float64Data, alpha float64) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if alpha <= 0 || alpha > 1 {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len())

	output[0] = input[0]
	for i := 1; i < input.Len(); i++ {
		output[i] = alpha*input[i] + (1-alpha)*output[i-1]
	}

	return output, nil
}

// EWMA returns the exponentially weighted moving average of the data with smoothing factor alpha
func (f Float64Data) EWMA(alpha float64) ([]float64, error) {
	return EWMA(f, alpha)
}
