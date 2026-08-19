package stats

import (
	"math"
	"sort"
)

// PercentileWeighted finds the weighted percentile of a slice of floats
// using the weighted empirical CDF (inverse CDF / nearest-rank method).
//
// For a given percent p, it returns the smallest data value x such that
// the cumulative weight of all values <= x is at least p% of the total
// weight. This matches the behavior of Python's statsmodels
// DescrStatsW.quantile.
//
// The data and weights slices must be the same length. Weights must be
// non-negative and at least one weight must be positive. The percent
// parameter must be between 0 and 100 (exclusive).
func PercentileWeighted(data, weights Float64Data, percent float64) (percentile float64, err error) {
	l := data.Len()
	if l == 0 {
		return math.NaN(), ErrEmptyInput
	}

	if weights.Len() != l {
		return math.NaN(), ErrSize
	}

	if math.IsNaN(percent) || percent <= 0 || percent > 100 {
		return math.NaN(), ErrBounds
	}

	// Build sorted pairs by data value
	type pair struct {
		value  float64
		weight float64
	}
	pairs := make([]pair, l)
	totalWeight := 0.0
	for i := 0; i < l; i++ {
		if weights[i] < 0 {
			return math.NaN(), ErrNegative
		}
		pairs[i] = pair{data[i], weights[i]}
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return math.NaN(), ErrBounds
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].value < pairs[j].value
	})

	// Find the smallest value where cumulative weight >= target
	target := (percent / 100) * totalWeight
	cumWeight := 0.0
	result := pairs[l-1].value
	for i := 0; i < l; i++ {
		cumWeight += pairs[i].weight
		if cumWeight >= target {
			result = pairs[i].value
			break
		}
	}

	return result, nil
}
