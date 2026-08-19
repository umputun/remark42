package stats

// Diff calculates the successive differences of the input slice,
// returning input[i] - input[i-1] for each i in 1..len(input)-1.
// The output has length len(input) - 1; a single-element input
// returns an empty slice.
func Diff(input Float64Data) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	diff := make([]float64, input.Len()-1)

	for i := 1; i < input.Len(); i++ {
		diff[i-1] = input[i] - input[i-1]
	}

	return diff, nil
}

// PercentChange calculates the fractional change between successive
// elements of the input slice, returning
// (input[i] - input[i-1]) / input[i-1] for each i in 1..len(input)-1.
// The output has length len(input) - 1; a single-element input
// returns an empty slice. A zero denominator follows IEEE 754
// semantics, yielding +Inf, -Inf, or NaN (for 0/0), matching the
// behavior of pandas pct_change.
func PercentChange(input Float64Data) ([]float64, error) {

	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	change := make([]float64, input.Len()-1)

	for i := 1; i < input.Len(); i++ {
		change[i-1] = (input[i] - input[i-1]) / input[i-1]
	}

	return change, nil
}

// Diff returns the successive differences of the data
func (f Float64Data) Diff() ([]float64, error) { return Diff(f) }

// PercentChange returns the fractional change between successive elements of the data
func (f Float64Data) PercentChange() ([]float64, error) { return PercentChange(f) }
