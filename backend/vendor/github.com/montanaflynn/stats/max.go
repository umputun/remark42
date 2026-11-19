package stats

import (
	"math"
)

// Max finds the highest number in a slice. If the slice contains a NaN
// the result is NaN.
func Max(input Float64Data) (max float64, err error) {

	// Return an error if there are no numbers
	if input.Len() == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Get the first value as the starting point
	max = input.Get(0)

	// Loop and replace higher values, stopping at a NaN
	for i := 0; i < input.Len(); i++ {
		if math.IsNaN(input.Get(i)) {
			return math.NaN(), nil
		}
		if input.Get(i) > max {
			max = input.Get(i)
		}
	}

	return max, nil
}
