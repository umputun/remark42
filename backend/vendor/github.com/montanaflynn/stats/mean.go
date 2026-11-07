package stats

import "math"

// Mean gets the average of a slice of numbers
func Mean(input Float64Data) (float64, error) {

	if input.Len() == 0 {
		return math.NaN(), EmptyInputErr
	}

	sum, _ := input.Sum()
	l := float64(input.Len())

	// The sum of finite values near math.MaxFloat64 can overflow even
	// though the mean cannot, so sum again scaled down by a power of two
	// at least the length, which is exact, and scale the mean back up.
	if math.IsInf(sum, 0) {
		_, e := math.Frexp(l)
		sum = 0
		for _, n := range input {
			sum += math.Ldexp(n, -e)
		}
		return math.Ldexp(sum/l, e), nil
	}

	return sum / l, nil
}

// GeometricMean gets the geometric mean for a slice of numbers
func GeometricMean(input Float64Data) (float64, error) {

	l := input.Len()
	if l == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Get the sum of all the numbers natural logs and return an
	// error for values that cannot be included in geometric mean
	var p float64
	for _, n := range input {
		if n < 0 {
			return math.NaN(), NegativeErr
		} else if n == 0 {
			return math.NaN(), ZeroErr
		}
		p += math.Log(n)
	}

	return math.Exp(p / float64(l)), nil
}

// HarmonicMean gets the harmonic mean for a slice of numbers
func HarmonicMean(input Float64Data) (float64, error) {

	l := input.Len()
	if l == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Get the sum of all the numbers reciprocals and return an
	// error for values that cannot be included in harmonic mean
	var p float64
	for _, n := range input {
		if n < 0 {
			return math.NaN(), NegativeErr
		} else if n == 0 {
			return math.NaN(), ZeroErr
		}
		p += (1 / n)
	}

	return float64(l) / p, nil
}
