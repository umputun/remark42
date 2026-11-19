package stats

import "math"

// SEM calculates the standard error of the mean of a slice
// of floats, defined as the sample standard deviation divided
// by the square root of the sample size. This matches the
// behavior of Python's scipy.stats.sem with ddof=1, except that a
// single element returns ErrTooFew instead of NaN.
func SEM(input Float64Data) (float64, error) {
	sd, err := StandardDeviationSample(input)
	if err != nil {
		return math.NaN(), err
	}

	return sd / math.Sqrt(float64(input.Len())), nil
}

// SEM calculates the standard error of the mean of the data
func (f Float64Data) SEM() (float64, error) {
	return SEM(f)
}
