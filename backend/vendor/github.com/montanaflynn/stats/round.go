package stats

import "math"

// Round a float to a specific decimal place or precision
func Round(input float64, places int) (rounded float64, err error) {
	if math.IsNaN(input) {
		return math.NaN(), NaNErr
	}
	precision := math.Pow(10, float64(places))
	return math.Round(input*precision) / precision, nil
}
