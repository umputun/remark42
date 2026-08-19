package stats

import "math"

// TTest performs a one-sample or two-sample (independent) Student's t-test.
//
// For a one-sample t-test, pass the sample data as data1, nil for data2,
// and the expected population mean as populationMean.
//
// For a two-sample independent t-test (assuming equal variance), pass both
// sample datasets. The populationMean parameter is ignored in this case.
//
// Returns the t statistic and the two-tailed p-value.
//
// https://en.wikipedia.org/wiki/Student%27s_t-test
func TTest(data1, data2 Float64Data, populationMean float64) (t float64, pvalue float64, err error) {

	n1 := data1.Len()
	if n1 == 0 {
		return math.NaN(), math.NaN(), ErrEmptyInput
	}

	mean1, _ := Mean(data1)

	// Two-sample independent t-test (equal variance)
	if data2 != nil && data2.Len() > 0 {
		n2 := data2.Len()

		if n1+n2 < 3 {
			return math.NaN(), math.NaN(), ErrBounds
		}

		mean2, _ := Mean(data2)
		var1, _ := SampleVariance(data1)
		var2, _ := SampleVariance(data2)

		df := float64(n1 + n2 - 2)
		pooledVar := (float64(n1-1)*var1 + float64(n2-1)*var2) / df
		se := math.Sqrt(pooledVar * (1.0/float64(n1) + 1.0/float64(n2)))
		t = (mean1 - mean2) / se
		pvalue = 2 * tSf(math.Abs(t), df)
	} else {
		// One-sample t-test
		if n1 < 2 {
			return math.NaN(), math.NaN(), ErrBounds
		}

		sd, _ := StandardDeviationSample(data1)
		if sd == 0 {
			if mean1 == populationMean {
				return 0, 1.0, nil
			}
			return math.NaN(), math.NaN(), ErrBounds
		}
		se := sd / math.Sqrt(float64(n1))
		t = (mean1 - populationMean) / se
		df := float64(n1 - 1)
		pvalue = 2 * tSf(math.Abs(t), df)
	}

	return t, pvalue, nil
}

// tSf is the survival function for Student's t-distribution.
// It computes 1 - CDF(t, df) using the regularized incomplete beta function.
func tSf(t float64, df float64) float64 {
	x := df / (df + t*t)
	return 0.5 * regIncBeta(df/2.0, 0.5, x)
}

// regIncBeta computes the regularized incomplete beta function I_x(a, b)
// using a continued fraction approximation (Lentz's algorithm).
func regIncBeta(a, b, x float64) float64 {
	if x == 0 || x == 1 {
		return x
	}

	lbeta := lgammaBeta(a, b)
	front := math.Exp(math.Log(x)*a+math.Log(1-x)*b-lbeta) / a

	// Use Lentz's continued fraction algorithm
	f := 1.0
	c := 1.0
	d := clampTiny(1.0 - (a+b)*x/(a+1))
	d = 1.0 / d
	f = d

	for i := 1; i <= 200; i++ {
		m := float64(i)
		// Numerator for even step
		num := m * (b - m) * x / ((a + 2*m - 1) * (a + 2*m))
		d = clampTiny(1.0 + num*d)
		c = clampTiny(1.0 + num/c)
		d = 1.0 / d
		f *= c * d

		// Numerator for odd step
		num = -(a + m) * (a + b + m) * x / ((a + 2*m) * (a + 2*m + 1))
		d = clampTiny(1.0 + num*d)
		c = clampTiny(1.0 + num/c)
		d = 1.0 / d
		delta := c * d
		f *= delta

		if math.Abs(delta-1.0) < 1e-10 {
			break
		}
	}

	return front * f
}

// clampTiny prevents division by zero in Lentz's continued fraction
// algorithm by replacing near-zero values with a small constant.
func clampTiny(v float64) float64 {
	if math.Abs(v) < 1e-30 {
		return 1e-30
	}
	return v
}

// lgammaBeta computes log(Beta(a, b)) = log(Gamma(a)) + log(Gamma(b)) - log(Gamma(a+b))
func lgammaBeta(a, b float64) float64 {
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	return la + lb - lab
}
