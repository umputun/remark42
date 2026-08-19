package stats

import "math"

// ZTest performs a one-sample or two-sample Z-test.
//
// For a one-sample Z-test, pass the sample data as data1, nil for data2,
// the known population mean as populationMean, and the known population
// standard deviation as populationStdDev.
//
// For a two-sample Z-test, pass both sample datasets and the known population
// standard deviations. The populationMean parameter is ignored in this case.
//
// Returns the Z statistic and the two-tailed p-value.
//
// https://en.wikipedia.org/wiki/Z-test
func ZTest(data1, data2 Float64Data, populationMean, populationStdDev float64) (z float64, pvalue float64, err error) {

	n1 := data1.Len()
	if n1 == 0 {
		return math.NaN(), math.NaN(), ErrEmptyInput
	}

	mean1, _ := Mean(data1)

	// Two-sample Z-test
	if data2 != nil && data2.Len() > 0 {
		n2 := data2.Len()
		mean2, _ := Mean(data2)

		if populationStdDev <= 0 {
			return math.NaN(), math.NaN(), ErrBounds
		}

		se := populationStdDev * math.Sqrt(1.0/float64(n1)+1.0/float64(n2))
		z = (mean1 - mean2) / se
	} else {
		// One-sample Z-test
		if populationStdDev <= 0 {
			return math.NaN(), math.NaN(), ErrBounds
		}

		se := populationStdDev / math.Sqrt(float64(n1))
		z = (mean1 - populationMean) / se
	}

	pvalue = 2 * NormSf(math.Abs(z), 0, 1)

	return z, pvalue, nil
}
