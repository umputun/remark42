package stats

import (
	"math"
	"sort"
)

// Correlation describes the degree of relationship between two sets of data
func Correlation(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), EmptyInputErr
	}

	if l1 != l2 {
		return math.NaN(), SizeErr
	}

	sdev1, _ := StandardDeviationPopulation(data1)
	sdev2, _ := StandardDeviationPopulation(data2)

	if sdev1 == 0 || sdev2 == 0 {
		return 0, nil
	}

	covp, _ := CovariancePopulation(data1, data2)
	return covp / (sdev1 * sdev2), nil
}

// Pearson calculates the Pearson product-moment correlation coefficient between two variables
func Pearson(data1, data2 Float64Data) (float64, error) {
	return Correlation(data1, data2)
}

// Spearman calculates the Spearman rank correlation coefficient between two variables.
// It works by ranking the data and then computing the Pearson correlation of the ranks.
// This method handles tied values using fractional (average) ranking.
func Spearman(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), EmptyInputErr
	}

	if l1 != l2 {
		return math.NaN(), SizeErr
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

// AutoCorrelation is the correlation of a signal with a delayed copy of itself as a function of delay
func AutoCorrelation(data Float64Data, lags int) (float64, error) {
	if len(data) < 1 {
		return 0, EmptyInputErr
	}

	if lags < 0 || lags >= len(data) {
		return 0, BoundsErr
	}

	mean, _ := Mean(data)

	var variance float64
	for _, v := range data {
		delta := v - mean
		variance += delta * delta
	}

	if variance == 0 {
		return 0, nil
	}

	var covariance float64
	for i := lags; i < len(data); i++ {
		covariance += (data[i] - mean) * (data[i-lags] - mean)
	}

	return covariance / variance, nil
}
