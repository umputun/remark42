package stats

import "math"

// Quartiles holds the three quartile points
type Quartiles struct {
	Q1 float64
	Q2 float64
	Q3 float64
}

// Quartile returns the three quartile points from a slice of data.
//
// The input must contain at least two elements: Q1 and Q3 are the medians of
// the lower and upper halves, which are empty for a single element.
func Quartile(input Float64Data) (Quartiles, error) {

	il := input.Len()
	if il < 2 {
		return Quartiles{}, EmptyInputErr
	}

	// Start by sorting a copy of the slice
	copy := sortedCopy(input)

	// Find the cutoff places depeding on if
	// the input slice length is even or odd
	var c1 int
	var c2 int
	if il%2 == 0 {
		c1 = il / 2
		c2 = il / 2
	} else {
		c1 = (il - 1) / 2
		c2 = c1 + 1
	}

	// Find the Medians with the cutoff points. Both halves have at least
	// one element once il >= 2, so Median cannot fail here.
	Q1, _ := Median(copy[:c1])
	Q2, _ := Median(copy)
	Q3, _ := Median(copy[c2:])

	return Quartiles{Q1, Q2, Q3}, nil

}

// InterQuartileRange finds the range between Q1 and Q3
func InterQuartileRange(input Float64Data) (float64, error) {
	qs, err := Quartile(input)
	if err != nil {
		return math.NaN(), err
	}
	iqr := qs.Q3 - qs.Q1
	return iqr, nil
}

// Midhinge finds the average of the first and third quartiles
func Midhinge(input Float64Data) (float64, error) {
	qs, err := Quartile(input)
	if err != nil {
		return math.NaN(), err
	}
	mh := (qs.Q1 + qs.Q3) / 2

	// Halve before adding when the sum of the quartiles overflows
	if math.IsInf(mh, 0) {
		mh = qs.Q1/2 + qs.Q3/2
	}

	return mh, nil
}

// Trimean finds the average of the median and the midhinge
func Trimean(input Float64Data) (float64, error) {
	q, err := Quartile(input)
	if err != nil {
		return math.NaN(), err
	}

	tm := (q.Q1 + (q.Q2 * 2) + q.Q3) / 4

	// Divide before adding when the weighted sum overflows
	if math.IsInf(tm, 0) {
		tm = q.Q1/4 + q.Q2/2 + q.Q3/4
	}

	return tm, nil
}
