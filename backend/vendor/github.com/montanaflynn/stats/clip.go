package stats

import "math"

// Clip clamps each value in the input slice into the
// inclusive range between min and max. A min greater than max,
// or a min or max that is NaN, returns ErrBounds.
func Clip(input Float64Data, min, max float64) ([]float64, error) {
	if input.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if math.IsNaN(min) || math.IsNaN(max) || min > max {
		return nil, ErrBounds
	}

	c := make([]float64, len(input))
	for i, v := range input {
		switch {
		case v < min:
			c[i] = min
		case v > max:
			c[i] = max
		default:
			c[i] = v
		}
	}
	return c, nil
}

// Clip clamps each value in the input slice into the
// inclusive range between min and max.
func (f Float64Data) Clip(min, max float64) ([]float64, error) {
	return Clip(f, min, max)
}
