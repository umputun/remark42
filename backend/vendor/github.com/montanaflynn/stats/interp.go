package stats

import (
	"math"
	"sort"
)

// Interp calculates the one-dimensional piecewise-linear interpolant to a
// function with given discrete data points (xp, fp), evaluated at each x.
// Values of x below xp[0] return fp[0] and values above xp[len(xp)-1] return
// fp[len(xp)-1], so no extrapolation is performed. Unlike numpy's interp,
// which silently returns nonsense for unsorted coordinates, xp must be
// strictly increasing or ErrBounds is returned. An empty x or xp returns
// ErrEmptyInput and xp and fp of different lengths return ErrSize.
// A NaN in xp returns ErrBounds and a NaN in x gives a NaN in the output.
func Interp(x, xp, fp Float64Data) ([]float64, error) {

	if x.Len() == 0 || xp.Len() == 0 {
		return nil, ErrEmptyInput
	}

	if xp.Len() != fp.Len() {
		return nil, ErrSize
	}

	// NaN loses every comparison, so the ordering check can't catch it
	for i := 0; i < xp.Len(); i++ {
		if math.IsNaN(xp[i]) || (i > 0 && xp[i] <= xp[i-1]) {
			return nil, ErrBounds
		}
	}

	output := make([]float64, x.Len())

	for i, xv := range x {
		switch {
		case math.IsNaN(xv):
			output[i] = math.NaN()
		case xv <= xp[0]:
			output[i] = fp[0]
		case xv >= xp[xp.Len()-1]:
			output[i] = fp[fp.Len()-1]
		default:
			// The first index with xp[j] >= xv, which the clamping
			// above guarantees is within [1, len(xp)-1]
			j := sort.SearchFloat64s(xp, xv)
			if xv == xp[j] {
				// An exact knot hit returns fp[j] exactly, with no
				// interpolation arithmetic that could lose precision
				output[i] = fp[j]
				continue
			}
			t := (xv - xp[j-1]) / (xp[j] - xp[j-1])
			if math.IsInf(xp[j]-xp[j-1], 1) {
				// The knot spacing overflows float64, so halve each
				// term before dividing; halving is exact for the huge
				// values that make an overflowing difference possible
				t = (xv/2 - xp[j-1]/2) / (xp[j]/2 - xp[j-1]/2)
			}
			// The symmetric form stays finite for any finite fp where
			// fp[j]-fp[j-1] would overflow, since 0 < t < 1 here
			output[i] = fp[j-1]*(1-t) + fp[j]*t
		}
	}

	return output, nil
}
