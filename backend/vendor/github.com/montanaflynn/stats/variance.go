package stats

import "math"

// _variance finds the variance for both population and sample data
func _variance(input Float64Data, sample int) (variance float64, err error) {

	if input.Len() == 0 {
		return math.NaN(), EmptyInputErr
	}

	v, e := scaledVariance(input, sample)
	return math.Ldexp(v, 2*e), nil
}

// scaledVariance returns the variance of the input divided by 2^(2e), where
// 2^e is near its largest magnitude. Squaring the scaled deviations cannot
// overflow or underflow, and scaling by a power of two is exact, so the
// variance is math.Ldexp(v, 2*e) and the standard deviation is
// math.Ldexp(math.Sqrt(v), e) even when the variance itself overflows.
func scaledVariance(input Float64Data, sample int) (variance float64, e int) {
	e = unitExponent(input)
	f := math.Ldexp(1, -e)

	// Sum the square of the mean subtracted from each number
	m, _ := Mean(input)
	m *= f

	for _, n := range input {
		d := n*f - m
		variance += d * d
	}

	// When getting the mean of the squared differences
	// "sample" will allow us to know if it's a sample
	// or population and wether to subtract by one or not
	return variance / float64((input.Len() - (1 * sample))), e
}

// Variance the amount of variation in the dataset
func Variance(input Float64Data) (sdev float64, err error) {
	return PopulationVariance(input)
}

// PopulationVariance finds the amount of variance within a population
func PopulationVariance(input Float64Data) (pvar float64, err error) {

	v, err := _variance(input, 0)
	if err != nil {
		return math.NaN(), err
	}

	return v, nil
}

// SampleVariance finds the amount of variance within a sample. It returns
// ErrTooFew for a single element, where the n-1 divisor is zero.
func SampleVariance(input Float64Data) (svar float64, err error) {

	if input.Len() == 1 {
		return math.NaN(), ErrTooFew
	}

	v, err := _variance(input, 1)
	if err != nil {
		return math.NaN(), err
	}

	return v, nil
}

// Covariance is a measure of how much two sets of data change. It is the
// sample covariance, so it returns ErrTooFew for a single pair.
func Covariance(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), EmptyInputErr
	}

	if l1 != l2 {
		return math.NaN(), SizeErr
	}

	if l1 == 1 {
		return math.NaN(), ErrTooFew
	}

	m1, _ := Mean(data1)
	m2, _ := Mean(data2)

	// Calculate sum of squares
	var ss float64
	for i := 0; i < l1; i++ {
		delta1 := (data1.Get(i) - m1)
		delta2 := (data2.Get(i) - m2)
		ss += (delta1*delta2 - ss) / float64(i+1)
	}

	return ss * float64(l1) / float64(l1-1), nil
}

// CovariancePopulation computes covariance for entire population between two variables.
func CovariancePopulation(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), EmptyInputErr
	}

	if l1 != l2 {
		return math.NaN(), SizeErr
	}

	m1, _ := Mean(data1)
	m2, _ := Mean(data2)

	var s float64
	for i := 0; i < l1; i++ {
		delta1 := (data1.Get(i) - m1)
		delta2 := (data2.Get(i) - m2)
		s += delta1 * delta2
	}

	return s / float64(l1), nil
}
