package stats

import (
	"math"
)

// Validate data for distance calculation
func validateData(dataPointX, dataPointY Float64Data) error {
	if len(dataPointX) == 0 || len(dataPointY) == 0 {
		return EmptyInputErr
	}

	if len(dataPointX) != len(dataPointY) {
		return SizeErr
	}
	return nil
}

// ChebyshevDistance computes the Chebyshev distance between two data sets.
// If any coordinate difference is NaN the distance is NaN.
func ChebyshevDistance(dataPointX, dataPointY Float64Data) (distance float64, err error) {
	err = validateData(dataPointX, dataPointY)
	if err != nil {
		return math.NaN(), err
	}
	var tempDistance float64
	for i := 0; i < len(dataPointY); i++ {
		tempDistance = math.Abs(dataPointX[i] - dataPointY[i])
		// Keep a NaN difference, since NaN never compares greater
		if distance < tempDistance || math.IsNaN(tempDistance) {
			distance = tempDistance
		}
	}
	return distance, nil
}

// EuclideanDistance computes the Euclidean distance between two data sets
func EuclideanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error) {

	err = validateData(dataPointX, dataPointY)
	if err != nil {
		return math.NaN(), err
	}
	x, y := dataPointX, dataPointY[:len(dataPointX)]

	// The leftover len%8 squares are summed first, while the sum is small.
	// The rest are summed in blocks of eight, and each block is added to sum
	// with TwoSum, which collects the exact rounding error in compensation so
	// small squares aren't lost next to a large sum.
	var sum, compensation float64
	i := len(x) % 8
	for j := 0; j < i; j++ {
		delta := x[j] - y[j]
		sum = sum + delta*delta
	}
	for ; i < len(x); i += 8 {
		a, b := x[i:i+8:i+8], y[i:i+8:i+8]
		d0, d1, d2, d3 := a[0]-b[0], a[1]-b[1], a[2]-b[2], a[3]-b[3]
		d4, d5, d6, d7 := a[4]-b[4], a[5]-b[5], a[6]-b[6], a[7]-b[7]
		block := ((d0*d0 + d1*d1) + (d2*d2 + d3*d3)) + ((d4*d4 + d5*d5) + (d6*d6 + d7*d7))
		total := sum + block
		z := total - sum
		compensation = compensation + (sum - (total - z)) + (block - z)
		sum = total
	}

	// Squares can only overflow or underflow when the sum is infinite or
	// tiny, so only those sums (and NaN) need to be recomputed with scaling.
	if sum >= 0x1p-900 && sum <= math.MaxFloat64 {
		return math.Sqrt(sum + compensation), nil
	}
	sum, scale := scaledSumSquares(x, y, sum)
	return math.Sqrt(sum) / scale, nil
}

// scaledSumSquares recomputes a sum of squares of x[i]-y[i], or of x[i]
// when y is nil, with every value multiplied by scale: 2^-600 after an
// overflow or 2^600 after an underflow. Scaling by a power of two is exact,
// so the squares stay in range without losing accuracy. The true sum of
// squares is sum / scale².
func scaledSumSquares(x, y Float64Data, unscaledSum float64) (sum, scale float64) {
	scale = 0x1p600
	if unscaledSum > math.MaxFloat64 {
		scale = 0x1p-600
	}
	var compensation float64
	for i := 0; i < len(x); i++ {
		delta := x[i]
		if y != nil {
			delta = delta - y[i]
		}
		delta = delta * scale
		square := float64(delta * delta) // rounded, so it isn't fused into the TwoSum below
		total := sum + square
		z := total - sum
		compensation = compensation + (sum - (total - z)) + (square - z)
		sum = total
	}
	if sum > math.MaxFloat64 { // an input is infinite
		return sum, scale
	}
	return sum + compensation, scale
}

// ManhattanDistance computes the Manhattan distance between two data sets
func ManhattanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error) {
	err = validateData(dataPointX, dataPointY)
	if err != nil {
		return math.NaN(), err
	}
	distance = 0
	for i := 0; i < len(dataPointX); i++ {
		distance = distance + math.Abs(dataPointX[i]-dataPointY[i])
	}
	return distance, nil
}

// MinkowskiDistance computes the Minkowski distance between two data sets
//
// Arguments:
//
//	dataPointX: First set of data points
//	dataPointY: Second set of data points. Length of both data
//	            sets must be equal.
//	lambda:     aka p or city blocks; With lambda = 1
//	            returned distance is manhattan distance and
//	            lambda = 2; it is euclidean distance. Lambda
//	            reaching to infinite - distance would be chebysev
//	            distance. A negative or NaN lambda returns
//	            ErrBounds.
//
// Return:
//
//	Distance or error
func MinkowskiDistance(dataPointX, dataPointY Float64Data, lambda float64) (distance float64, err error) {
	err = validateData(dataPointX, dataPointY)
	if err != nil {
		return math.NaN(), err
	}
	if math.IsNaN(lambda) || lambda < 0 {
		return math.NaN(), ErrBounds
	}
	if lambda == 2 {
		// The same distance, without overflowing or underflowing the squares.
		distance, _ = EuclideanDistance(dataPointX, dataPointY)
	} else {
		for i := 0; i < len(dataPointY); i++ {
			distance = distance + math.Pow(math.Abs(dataPointX[i]-dataPointY[i]), lambda)
		}
		distance = math.Pow(distance, 1/lambda)

		// The powers overflow or underflow for differences far from unit
		// magnitude, so compute them again from scaled differences
		if (math.IsInf(distance, 1) || distance == 0) && lambda > 0 {
			distance = scaledMinkowskiDistance(dataPointX, dataPointY, lambda)
		}
	}

	if math.IsInf(distance, 1) {
		return math.NaN(), InfValue
	}
	return distance, nil
}

// scaledMinkowskiDistance computes the Minkowski distance from differences
// scaled by powers of two, first so the differences cannot overflow and then
// so the largest is in [0.5, 1), and scales the result back. Scaling by a
// power of two is exact.
func scaledMinkowskiDistance(dataPointX, dataPointY Float64Data, lambda float64) float64 {
	e := unitExponent(append(append(Float64Data{}, dataPointX...), dataPointY...))
	f := math.Ldexp(1, -e)
	diffs := make(Float64Data, len(dataPointX))
	for i := range diffs {
		diffs[i] = math.Abs(dataPointX[i]*f - dataPointY[i]*f)
	}
	de := unitExponent(diffs)
	var sum float64
	for _, d := range diffs {
		sum += math.Pow(math.Ldexp(d, -de), lambda)
	}
	return math.Ldexp(math.Pow(sum, 1/lambda), e+de)
}
