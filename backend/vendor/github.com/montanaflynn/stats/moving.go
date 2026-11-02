package stats

// MovingMedian calculates the rolling median of the input over a trailing
// window. Only fully-populated windows produce output, so the result has
// len(input)-window+1 entries and entry i is the median of input[i : i+window].
// The window must satisfy 1 <= window <= len(input) or ErrBounds is
// returned. An empty input returns ErrEmptyInput.
func MovingMedian(input Float64Data, window int) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if window < 1 || window > input.Len() {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len()-window+1)

	for i := range output {
		// Median cannot fail here since every window is non-empty
		median, _ := Median(input[i : i+window])
		output[i] = median
	}

	return output, nil
}

// MovingMin calculates the rolling minimum of the input over a trailing
// window. Only fully-populated windows produce output, so the result has
// len(input)-window+1 entries and entry i is the minimum of
// input[i : i+window]. The window must satisfy 1 <= window <= len(input) or
// ErrBounds is returned. An empty input returns ErrEmptyInput.
func MovingMin(input Float64Data, window int) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if window < 1 || window > input.Len() {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len()-window+1)

	for i := range output {
		// Min cannot fail here since every window is non-empty
		min, _ := Min(input[i : i+window])
		output[i] = min
	}

	return output, nil
}

// MovingMax calculates the rolling maximum of the input over a trailing
// window. Only fully-populated windows produce output, so the result has
// len(input)-window+1 entries and entry i is the maximum of
// input[i : i+window]. The window must satisfy 1 <= window <= len(input) or
// ErrBounds is returned. An empty input returns ErrEmptyInput.
func MovingMax(input Float64Data, window int) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if window < 1 || window > input.Len() {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len()-window+1)

	for i := range output {
		// Max cannot fail here since every window is non-empty
		max, _ := Max(input[i : i+window])
		output[i] = max
	}

	return output, nil
}

// MovingSum calculates the rolling sum of the input over a trailing
// window. Only fully-populated windows produce output, so the result has
// len(input)-window+1 entries and entry i is the sum of input[i : i+window].
// The window must satisfy 1 <= window <= len(input) or ErrBounds is
// returned. An empty input returns ErrEmptyInput.
func MovingSum(input Float64Data, window int) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if window < 1 || window > input.Len() {
		return nil, ErrBounds
	}

	output := make([]float64, input.Len()-window+1)

	for i := range output {
		// Sum cannot fail here since every window is non-empty
		sum, _ := Sum(input[i : i+window])
		output[i] = sum
	}

	return output, nil
}

// MovingMedian returns the rolling median of the data over a trailing window
func (f Float64Data) MovingMedian(window int) ([]float64, error) {
	return MovingMedian(f, window)
}

// MovingMin returns the rolling minimum of the data over a trailing window
func (f Float64Data) MovingMin(window int) ([]float64, error) {
	return MovingMin(f, window)
}

// MovingMax returns the rolling maximum of the data over a trailing window
func (f Float64Data) MovingMax(window int) ([]float64, error) {
	return MovingMax(f, window)
}

// MovingSum returns the rolling sum of the data over a trailing window
func (f Float64Data) MovingSum(window int) ([]float64, error) {
	return MovingSum(f, window)
}
