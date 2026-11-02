package stats

// MovingAverage calculates the rolling mean of the input over a trailing
// window. Only fully-populated windows produce output, so the result has
// len(input)-window+1 entries and entry i is the mean of input[i : i+window].
// The window must satisfy 1 <= window <= len(input) or ErrBounds is
// returned. An empty input returns ErrEmptyInput.
func MovingAverage(input Float64Data, window int) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if window < 1 || window > input.Len() {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len()-window+1)

	for i := range output {
		// Mean cannot fail here since every window is non-empty
		mean, _ := Mean(input[i : i+window])
		output[i] = mean
	}

	return output, nil
}

// MovingStdDev calculates the rolling sample standard deviation of the input
// over a trailing window. Only fully-populated windows produce output, so the
// result has len(input)-window+1 entries and entry i is the sample standard
// deviation of input[i : i+window]. The window must satisfy
// 2 <= window <= len(input) or ErrBounds is returned, since the sample
// standard deviation of a single value is undefined. An empty input returns
// ErrEmptyInput.
func MovingStdDev(input Float64Data, window int) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if window < 2 || window > input.Len() {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len()-window+1)

	for i := range output {
		// StandardDeviationSample cannot fail here since every window is non-empty
		sdev, _ := StandardDeviationSample(input[i : i+window])
		output[i] = sdev
	}

	return output, nil
}

// MovingAverage returns the rolling mean of the data over a trailing window
func (f Float64Data) MovingAverage(window int) ([]float64, error) {
	return MovingAverage(f, window)
}

// MovingStdDev returns the rolling sample standard deviation of the data over a trailing window
func (f Float64Data) MovingStdDev(window int) ([]float64, error) {
	return MovingStdDev(f, window)
}
