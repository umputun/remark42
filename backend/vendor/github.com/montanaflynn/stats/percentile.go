package stats

import (
	"math"
)

// Percentile finds the relative standing in a slice of floats.
//
// The function uses the Linear Interpolation Between Closest Ranks method
// (Hyndman & Fan type 7), the default in Excel (PERCENTILE, PERCENTILE.INC),
// Google Sheets, NumPy, and R. Note that NIST [1] describes the closely
// related p(N+1) variant (Hyndman & Fan type 6, Excel PERCENTILE.EXC),
// which gives different results for the same input.
//
// Algorithm (for percent p and sorted data of length n):
//
//  1. Compute the rank: rank = (p / 100) * (n - 1)
//  2. Split into integer part k and fractional part f
//  3. Result = data[k] + f * (data[k+1] - data[k]), or data[k] when f is 0
//     or data[k] equals data[k+1]
//
// When data[k+1] - data[k] is infinite, because it overflows or an endpoint
// is infinite, the endpoints are weighted separately as
// (1 - f) * data[k] + f * data[k+1] so large finite inputs stay finite.
//
// If the input contains a NaN the result is NaN.
//
// [1] https://www.itl.nist.gov/div898/handbook/prc/section2/prc262.htm
func Percentile(input Float64Data, percent float64) (percentile float64, err error) {
	length := input.Len()
	if length == 0 {
		return math.NaN(), EmptyInputErr
	}

	if math.IsNaN(percent) || percent <= 0 || percent > 100 {
		return math.NaN(), BoundsErr
	}

	// A single value is its own percentile for every valid percent. The
	// general path below gives the same answer (rank 0, no interpolation),
	// so this only skips the sortedCopy allocation. It must stay after the
	// bounds check so invalid percents are rejected at every length.
	if length == 1 {
		return input[0], nil
	}

	// Sorting places a NaN below every number, so it has no rank
	if hasNaN(input) {
		return math.NaN(), nil
	}

	// Start by sorting a copy of the slice
	c := sortedCopy(input)

	// Use the standard linear interpolation method:
	// rank = (percent / 100) * (n - 1)
	// result = c[k] + f * (c[k+1] - c[k]), or c[k] for an exact rank or
	// when c[k] and c[k+1] are equal
	rank := (percent / 100) * float64(length-1)
	k := int(rank)
	f := rank - float64(k)

	if k+1 < length && f != 0 && c[k+1] != c[k] {
		delta := c[k+1] - c[k]
		if math.IsInf(delta, 1) {
			// Weight the endpoints separately when their difference is
			// infinite, from overflow or from an infinite endpoint.
			percentile = (1-f)*c[k] + f*c[k+1]
		} else {
			percentile = c[k] + f*delta
		}
	} else {
		percentile = c[k]
	}

	return percentile, nil

}

// PercentileNearestRank finds the relative standing in a slice of floats using the Nearest Rank method.
// If the input contains a NaN the result is NaN.
func PercentileNearestRank(input Float64Data, percent float64) (percentile float64, err error) {

	// Find the length of items in the slice
	il := input.Len()

	// Return an error for empty slices
	if il == 0 {
		return math.NaN(), EmptyInputErr
	}

	// Return error for NaN, less than 0, or greater than 100 percentages
	if math.IsNaN(percent) || percent < 0 || percent > 100 {
		return math.NaN(), BoundsErr
	}

	// Sorting places a NaN below every number, so it has no rank
	if hasNaN(input) {
		return math.NaN(), nil
	}

	// Start by sorting a copy of the slice
	c := sortedCopy(input)

	// Return the last item
	if percent == 100.0 {
		return c[il-1], nil
	}

	// Find ordinal ranking
	or := int(math.Ceil(float64(il) * percent / 100))

	// Return the item that is in the place of the ordinal rank
	if or == 0 {
		return c[0], nil
	}
	return c[or-1], nil

}
