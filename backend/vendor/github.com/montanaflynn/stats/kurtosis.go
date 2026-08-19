package stats

import "math"

// Kurtosis computes the population excess kurtosis of the dataset
func Kurtosis(input Float64Data) (float64, error) {
	return PopulationKurtosis(input)
}

// PopulationKurtosis computes the population excess kurtosis (Fisher
// definition) using the fourth central moment normalized by the squared
// variance, so a normal distribution has a kurtosis of zero.
func PopulationKurtosis(input Float64Data) (float64, error) {
	if input.Len() < 2 {
		return math.NaN(), ErrEmptyInput
	}

	mean, _ := Mean(input)

	// Compute sums of squared and fourth-power differences from the mean
	var sumOfSquares, sumOfFourths float64
	for _, v := range input {
		d := v - mean
		d2 := d * d
		sumOfSquares += d2
		sumOfFourths += d2 * d2
	}

	if sumOfSquares == 0 {
		return math.NaN(), ErrZero
	}

	n := float64(input.Len())
	variance := sumOfSquares / n

	return (sumOfFourths/n)/(variance*variance) - 3.0, nil
}

// SampleKurtosis computes the bias-corrected sample excess kurtosis,
// matching pandas .kurt() and scipy.stats.kurtosis with bias=False.
func SampleKurtosis(input Float64Data) (float64, error) {
	n := input.Len()
	if n < 4 {
		return math.NaN(), ErrEmptyInput
	}

	g2, err := PopulationKurtosis(input)
	if err != nil {
		return math.NaN(), err
	}

	// Bias-corrected: G2 = ((n+1)*g2 + 6) * (n-1) / ((n-2)*(n-3))
	nf := float64(n)
	return ((nf+1)*g2 + 6) * (nf - 1) / ((nf - 2) * (nf - 3)), nil
}

// Kurtosis finds the population excess kurtosis of a slice of floats
func (f Float64Data) Kurtosis() (float64, error) {
	return Kurtosis(f)
}

// PopulationKurtosis finds the population excess kurtosis of a slice of floats
func (f Float64Data) PopulationKurtosis() (float64, error) {
	return PopulationKurtosis(f)
}

// SampleKurtosis finds the bias-corrected sample excess kurtosis of a slice of floats
func (f Float64Data) SampleKurtosis() (float64, error) {
	return SampleKurtosis(f)
}
