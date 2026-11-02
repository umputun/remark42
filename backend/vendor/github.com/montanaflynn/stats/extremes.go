package stats

// ArgMax finds the index of the highest number in a slice,
// returning the first occurrence in the case of ties
func ArgMax(input Float64Data) (int, error) {

	// Return an error if there are no numbers
	if input.Len() == 0 {
		return -1, ErrEmptyInput
	}

	// Track the index of the highest value seen so far
	index := 0

	// Loop and replace with strictly higher values only,
	// which keeps the first occurrence on ties
	for i := 1; i < input.Len(); i++ {
		if input.Get(i) > input.Get(index) {
			index = i
		}
	}

	return index, nil
}

// ArgMin finds the index of the lowest number in a slice,
// returning the first occurrence in the case of ties
func ArgMin(input Float64Data) (int, error) {

	// Return an error if there are no numbers
	if input.Len() == 0 {
		return -1, ErrEmptyInput
	}

	// Track the index of the lowest value seen so far
	index := 0

	// Loop and replace with strictly lower values only,
	// which keeps the first occurrence on ties
	for i := 1; i < input.Len(); i++ {
		if input.Get(i) < input.Get(index) {
			index = i
		}
	}

	return index, nil
}

// Range finds the difference between the highest and
// lowest numbers in a slice
func Range(input Float64Data) (float64, error) {

	// Return an error if there are no numbers
	max, err := Max(input)
	if err != nil {
		return max, ErrEmptyInput
	}

	// Disregard error, since Max would have already returned it
	min, _ := Min(input)

	return max - min, nil
}

// ArgMax returns the index of the highest number in the data
func (f Float64Data) ArgMax() (int, error) { return ArgMax(f) }

// ArgMin returns the index of the lowest number in the data
func (f Float64Data) ArgMin() (int, error) { return ArgMin(f) }

// Range returns the difference between the highest and lowest numbers in the data
func (f Float64Data) Range() (float64, error) { return Range(f) }
