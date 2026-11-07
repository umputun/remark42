package stats

import (
	"math"
	"sort"
)

// Correlation describes the degree of relationship between two sets of data.
// The correlation is undefined for a single pair or when either set is
// constant, so it returns 0 with ErrTooFew or ErrZero respectively.
func Correlation(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), EmptyInputErr
	}

	if l1 != l2 {
		return math.NaN(), SizeErr
	}

	if l1 == 1 {
		return 0, ErrTooFew
	}

	// Correlation is scale-invariant, so scale each input to unit magnitude
	// first to keep the squared deviations from overflowing or underflowing.
	data1, data2 = scaleToUnit(data1), scaleToUnit(data2)

	sdev1, _ := StandardDeviationPopulation(data1)
	sdev2, _ := StandardDeviationPopulation(data2)

	if sdev1 == 0 || sdev2 == 0 {
		return 0, ErrZero
	}

	covp, _ := CovariancePopulation(data1, data2)
	return covp / (sdev1 * sdev2), nil
}

// scaleToUnit returns a copy of input multiplied by a power of two so its
// largest magnitude is in [0.5, 1). Scaling by a power of two is exact, so
// scale-invariant statistics computed from the copy are unchanged.
func scaleToUnit(input Float64Data) Float64Data {
	exp := unitExponent(input)
	scaled := make(Float64Data, len(input))
	for i, v := range input {
		scaled[i] = math.Ldexp(v, -exp)
	}
	return scaled
}

// Pearson calculates the Pearson product-moment correlation coefficient between two variables
func Pearson(data1, data2 Float64Data) (float64, error) {
	return Correlation(data1, data2)
}

// Spearman calculates the Spearman rank correlation coefficient between two variables.
// It works by ranking the data and then computing the Pearson correlation of the ranks.
// This method handles tied values using fractional (average) ranking.
// Like Correlation, it returns 0 with ErrTooFew for a single pair and 0 with
// ErrZero when either set is constant. If either variable contains a NaN the
// result is NaN.
func Spearman(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), EmptyInputErr
	}

	if l1 != l2 {
		return math.NaN(), SizeErr
	}

	// A NaN has no rank
	if hasNaN(data1) || hasNaN(data2) {
		return math.NaN(), nil
	}

	ranks1 := rankData(data1)
	ranks2 := rankData(data2)

	return Correlation(ranks1, ranks2)
}

// rankData assigns fractional (average) ranks to the data values.
// Tied values receive the average of the ranks they would have been assigned.
func rankData(data Float64Data) Float64Data {
	n := len(data)

	// Create index-value pairs and sort by value
	type indexedValue struct {
		index int
		value float64
	}

	sorted := make([]indexedValue, n)
	for i, v := range data {
		sorted[i] = indexedValue{i, v}
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].value < sorted[j].value
	})

	ranks := make(Float64Data, n)

	// Assign fractional ranks handling ties
	for i := 0; i < n; {
		j := i + 1
		for j < n && sorted[j].value == sorted[i].value {
			j++
		}

		// Average rank for tied values (ranks are 1-based)
		avgRank := float64(i+j+1) / 2.0
		for k := i; k < j; k++ {
			ranks[sorted[k].index] = avgRank
		}

		i = j
	}

	return ranks
}

// AutoCorrelation is the correlation of a signal with a delayed copy of itself as a function of delay.
// It returns 0 with ErrZero for constant data, where the correlation is undefined.
func AutoCorrelation(data Float64Data, lags int) (float64, error) {
	if len(data) < 1 {
		return 0, EmptyInputErr
	}

	if lags < 0 || lags >= len(data) {
		return 0, BoundsErr
	}

	// Autocorrelation is scale-invariant, so scale to unit magnitude first
	// to keep the squared deviations from overflowing or underflowing.
	data = scaleToUnit(data)

	mean, _ := Mean(data)

	var variance float64
	for _, v := range data {
		delta := v - mean
		variance += delta * delta
	}

	if variance == 0 {
		return 0, ErrZero
	}

	var covariance float64
	for i := lags; i < len(data); i++ {
		covariance += (data[i] - mean) * (data[i-lags] - mean)
	}

	return covariance / variance, nil
}
