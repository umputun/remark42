package stats

import "math"

// Entropy provides calculation of the entropy
func Entropy(input Float64Data) (float64, error) {
	input, err := normalize(input)
	if err != nil {
		return math.NaN(), err
	}
	var result float64
	for i := 0; i < input.Len(); i++ {
		v := input.Get(i)
		if v == 0 {
			continue
		}
		result += (v * math.Log(v))
	}
	return -result, nil
}

// normalize divides each value by the sum of all values,
// leaving the input itself unchanged.
func normalize(input Float64Data) (Float64Data, error) {
	sum, err := input.Sum()
	if err != nil {
		return Float64Data{}, err
	}
	c := copyslice(input)
	for i := 0; i < c.Len(); i++ {
		c[i] = c[i] / sum
	}
	return c, nil
}
