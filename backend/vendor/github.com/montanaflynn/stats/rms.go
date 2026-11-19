package stats

import "math"

// RMS calculates the root mean square of a slice of floats,
// defined as the square root of the mean of the squared values.
func RMS(input Float64Data) (float64, error) {
	if input.Len() == 0 {
		return math.NaN(), ErrEmptyInput
	}

	// Summed like EuclideanDistance: blocks of eight added with TwoSum, and
	// recomputed with scaling only when the squares overflow or underflow.
	var sum, compensation float64
	head := len(input) % 8
	for _, v := range input[:head] {
		sum = sum + v*v
	}
	for rest := input[head:]; len(rest) >= 8; rest = rest[8:] {
		v := rest[:8:8]
		block := ((v[0]*v[0] + v[1]*v[1]) + (v[2]*v[2] + v[3]*v[3])) + ((v[4]*v[4] + v[5]*v[5]) + (v[6]*v[6] + v[7]*v[7]))
		total := sum + block
		z := total - sum
		compensation = compensation + (sum - (total - z)) + (block - z)
		sum = total
	}
	n := float64(len(input))
	if sum >= 0x1p-900 && sum <= math.MaxFloat64 {
		return math.Sqrt((sum + compensation) / n), nil
	}
	sum, scale := scaledSumSquares(input, nil, sum)
	return math.Sqrt(sum/n) / scale, nil
}

// RMS calculates the root mean square of the data
func (f Float64Data) RMS() (float64, error) {
	return RMS(f)
}
