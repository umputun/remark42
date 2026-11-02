package stats

import "math"

// KendallTau calculates Kendall's tau-b rank correlation coefficient
// between two variables. Tau-b corrects for ties, matching the values
// produced by SciPy's kendalltau and pandas' corr(method="kendall").
// Pairs are compared with a simple O(n^2) loop for clarity.
func KendallTau(data1, data2 Float64Data) (float64, error) {

	l1 := data1.Len()
	l2 := data2.Len()

	if l1 == 0 || l2 == 0 {
		return math.NaN(), ErrEmptyInput
	}

	if l1 != l2 {
		return math.NaN(), ErrSize
	}

	// Count concordant and discordant pairs along with pairs
	// tied only in the first or only in the second variable.
	var concordant, discordant, tiedX, tiedY float64
	for i := 0; i < l1; i++ {
		for j := i + 1; j < l1; j++ {
			dx := data1[i] - data1[j]
			dy := data2[i] - data2[j]
			switch {
			case dx == 0 && dy == 0:
				// Pairs tied in both variables are excluded
			case dx == 0:
				tiedX++
			case dy == 0:
				tiedY++
			case (dx > 0) == (dy > 0):
				concordant++
			default:
				discordant++
			}
		}
	}

	// tau-b = (C - D) / sqrt((C + D + Tx) * (C + D + Ty))
	denominator := math.Sqrt((concordant + discordant + tiedX) * (concordant + discordant + tiedY))
	if denominator == 0 {
		return 0, nil
	}

	return (concordant - discordant) / denominator, nil
}

// KendallTau calculates Kendall's tau-b rank correlation coefficient
// between two variables.
func (f Float64Data) KendallTau(d Float64Data) (float64, error) {
	return KendallTau(f, d)
}
