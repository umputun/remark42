package stats

import "math"

// Series is a container for a series of data
type Series []Coordinate

// Coordinate holds the data in a series
type Coordinate struct {
	X, Y float64
}

// LinearRegression finds the least squares linear regression on data series.
// A series without at least two distinct X values returns ErrBounds.
func LinearRegression(s Series) (regressions Series, err error) {

	if len(s) == 0 {
		return nil, EmptyInputErr
	}

	var sumX, sumY float64
	for _, coordinate := range s {
		sumX += coordinate.X
		sumY += coordinate.Y
	}
	meanX := sumX / float64(len(s))
	meanY := sumY / float64(len(s))

	var covariance, variance float64
	for _, coordinate := range s {
		dx := coordinate.X - meanX
		covariance += dx * (coordinate.Y - meanY)
		variance += dx * dx
	}
	if variance == 0 {
		return nil, ErrBounds
	}
	gradient := covariance / variance

	// Create the new regression series
	for j := 0; j < len(s); j++ {
		regressions = append(regressions, Coordinate{
			X: s[j].X,
			Y: meanY + gradient*(s[j].X-meanX),
		})
	}

	return regressions, nil
}

// ExponentialRegression returns an exponential regression on data series.
// A non-positive Y value returns ErrYCoord, and a series without at least two
// distinct X values returns ErrBounds.
func ExponentialRegression(s Series) (regressions Series, err error) {

	if len(s) == 0 {
		return nil, EmptyInputErr
	}

	var sumY, sumDeltaXY, sumYLogY float64
	referenceX := s[0].X

	for i := 0; i < len(s); i++ {
		if s[i].Y <= 0 {
			return nil, ErrYCoord
		}
		sumY += s[i].Y
		sumDeltaXY += (s[i].X - referenceX) * s[i].Y
		sumYLogY += s[i].Y * math.Log(s[i].Y)
	}
	meanDeltaX := sumDeltaXY / sumY
	meanLogY := sumYLogY / sumY

	var covariance, variance float64
	for _, coordinate := range s {
		dx := coordinate.X - referenceX - meanDeltaX
		covariance += coordinate.Y * dx * (math.Log(coordinate.Y) - meanLogY)
		variance += coordinate.Y * dx * dx
	}
	if variance == 0 {
		return nil, ErrBounds
	}
	b := covariance / variance

	for j := 0; j < len(s); j++ {
		regressions = append(regressions, Coordinate{
			X: s[j].X,
			Y: math.Exp(meanLogY + b*(s[j].X-referenceX-meanDeltaX)),
		})
	}

	return regressions, nil
}

// LogarithmicRegression returns a logarithmic regression on data series.
// A non-positive X value or a series without at least two distinct X values
// returns ErrBounds.
func LogarithmicRegression(s Series) (regressions Series, err error) {

	if len(s) == 0 {
		return nil, EmptyInputErr
	}
	if s[0].X <= 0 {
		return nil, ErrBounds
	}

	logX := make([]float64, len(s))
	referenceLogX := math.Log(s[0].X)
	var sumDeltaLogX, sumY float64

	for i := 0; i < len(s); i++ {
		if s[i].X <= 0 {
			return nil, ErrBounds
		}
		logX[i] = math.Log(s[i].X)
		sumDeltaLogX += logX[i] - referenceLogX
		sumY += s[i].Y
	}
	meanDeltaLogX := sumDeltaLogX / float64(len(s))
	meanY := sumY / float64(len(s))

	var covariance, variance float64
	for i, coordinate := range s {
		dx := logX[i] - referenceLogX - meanDeltaLogX
		covariance += dx * (coordinate.Y - meanY)
		variance += dx * dx
	}
	if variance == 0 {
		return nil, ErrBounds
	}
	a := covariance / variance

	for j := 0; j < len(s); j++ {
		regressions = append(regressions, Coordinate{
			X: s[j].X,
			Y: meanY + a*(logX[j]-referenceLogX-meanDeltaLogX),
		})
	}

	return regressions, nil
}
