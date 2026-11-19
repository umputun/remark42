package stats

import "math"

// Skewness computes the population skewness of the dataset
func Skewness(input Float64Data) (float64, error) {
	return PopulationSkewness(input)
}

// PopulationSkewness computes the population skewness using the third
// central moment normalized by the cube of the standard deviation.
func PopulationSkewness(input Float64Data) (float64, error) {
	if input.Len() < 2 {
		return math.NaN(), ErrEmptyInput
	}

	// Skewness is scale-invariant, so scale the deviations by a power of two
	// near the largest magnitude to keep their cubes from overflowing or
	// underflowing. Scaling by a power of two is exact.
	f := math.Ldexp(1, -unitExponent(input))
	mean, _ := Mean(input)
	mean *= f

	// Compute sum of squared and cubed differences from the mean
	var sumOfSquares, sumOfCubes float64
	for _, v := range input {
		d := v*f - mean
		sumOfSquares += d * d
		sumOfCubes += d * d * d
	}

	if sumOfSquares == 0 {
		return math.NaN(), ErrZero
	}

	if sumOfCubes == 0 {
		return 0.0, nil
	}

	n := float64(input.Len())
	variance := sumOfSquares / n
	// variance^1.5 with Sqrt is accurate to about an ulp and scales
	// exactly, where math.Pow can be off by tens of ulps
	stdDevCubed := variance * math.Sqrt(variance)

	return (sumOfCubes / n) / stdDevCubed, nil
}

// SampleSkewness computes the adjusted Fisher-Pearson standardized moment
// coefficient, correcting for bias in small samples.
func SampleSkewness(input Float64Data) (float64, error) {
	n := input.Len()
	if n < 3 {
		return math.NaN(), ErrEmptyInput
	}

	g1, err := PopulationSkewness(input)
	if err != nil {
		return math.NaN(), err
	}

	if g1 == 0 {
		return 0.0, nil
	}

	// Adjusted Fisher-Pearson: G1 = g1 * sqrt(n*(n-1)) / (n-2)
	nf := float64(n)
	return g1 * math.Sqrt(nf*(nf-1)) / (nf - 2), nil
}
