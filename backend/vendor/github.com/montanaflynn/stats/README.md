# Stats - Golang Statistics Package

[![][action-svg]][action-url] [![][codecov-svg]][codecov-url] [![][pkggodev-svg]][pkggodev-url] [![][license-svg]][license-url]

A well tested and comprehensive Golang statistics library / package / module with no dependencies.

If you have any suggestions, problems or bug reports please [create an issue](https://github.com/montanaflynn/stats/issues) and I'll do my best to accommodate you. In addition simply starring the repo would show your support for the project and be very much appreciated!

## Installation

```
go get github.com/montanaflynn/stats
```

## Example Usage

All the functions can be seen in [examples/functions/main.go](examples/functions/main.go) but here's a little taste:

```go
// start with some source data to use
data := []float64{1.0, 2.1, 3.2, 4.823, 4.1, 5.8}

// you could also use different types like this
// data := stats.LoadRawData([]int{1, 2, 3, 4, 5})
// data := stats.LoadRawData([]interface{}{1.1, "2", 3})
// etc...

median, _ := stats.Median(data)
fmt.Println(median) // 3.65

roundedMedian, _ := stats.Round(median, 0)
fmt.Println(roundedMedian) // 4
```

## Documentation

The entire API documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/montanaflynn/stats).

You can also view docs offline with the following commands:

```
# Command line
go doc -all .       # show all exported apis
go doc Median       # show a single function
go doc Float64Data  # show the type and methods

# Local website (go install golang.org/x/pkgsite/cmd/pkgsite@latest)
pkgsite -http=:4444 # start the pkgsite server on port 4444
open http://localhost:4444/github.com/montanaflynn/stats
```

The exported API is as follows:

```go
var (
    ErrEmptyInput = statsError{"Input must not be empty."}
    ErrNaN        = statsError{"Not a number."}
    ErrNegative   = statsError{"Must not contain negative values."}
    ErrZero       = statsError{"Must not contain zero values."}
    ErrBounds     = statsError{"Input is outside of range."}
    ErrSize       = statsError{"Must be the same length."}
    ErrInfValue   = statsError{"Value is infinite."}
    ErrYCoord     = statsError{"Y Value must be greater than zero."}
)

func Round(input float64, places int) (rounded float64, err error) {}

type Float64Data []float64

func LoadRawData(raw interface{}) (f Float64Data) {}

func ArgMax(input Float64Data) (int, error) {}
func ArgMin(input Float64Data) (int, error) {}
func AutoCorrelation(data Float64Data, lags int) (float64, error) {}
func ChebyshevDistance(dataPointX, dataPointY Float64Data) (distance float64, err error) {}
func Clip(input Float64Data, min, max float64) ([]float64, error) {}
func CoefficientOfVariation(input Float64Data) (float64, error) {}
func Correlation(data1, data2 Float64Data) (float64, error) {}
func Covariance(data1, data2 Float64Data) (float64, error) {}
func CovariancePopulation(data1, data2 Float64Data) (float64, error) {}
func CumulativeMax(input Float64Data) ([]float64, error) {}
func CumulativeMin(input Float64Data) ([]float64, error) {}
func CumulativeProduct(input Float64Data) ([]float64, error) {}
func CumulativeSum(input Float64Data) ([]float64, error) {}
func Describe(input Float64Data, allowNaN bool, percentiles *[]float64) (*Description, error) {}
func DescribePercentileFunc(input Float64Data, allowNaN bool, percentiles *[]float64, percentileFunc func(Float64Data, float64) (float64, error)) (*Description, error) {}
func Diff(input Float64Data) ([]float64, error) {}
func EWMA(input Float64Data, alpha float64) ([]float64, error) {}
func Entropy(input Float64Data) (float64, error) {}
func EuclideanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error) {}
func GeometricMean(input Float64Data) (float64, error) {}
func HarmonicMean(input Float64Data) (float64, error) {}
func Histogram(input Float64Data, bins int) ([]int, []float64, error) {}
func InterQuartileRange(input Float64Data) (float64, error) {}
func Interp(x, xp, fp Float64Data) ([]float64, error) {}
func KendallTau(data1, data2 Float64Data) (float64, error) {}
func Kurtosis(input Float64Data) (float64, error) {}
func ManhattanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error) {}
func Max(input Float64Data) (max float64, err error) {}
func Mean(input Float64Data) (float64, error) {}
func Median(input Float64Data) (median float64, err error) {}
func MedianAbsoluteDeviation(input Float64Data) (mad float64, err error) {}
func MedianAbsoluteDeviationPopulation(input Float64Data) (mad float64, err error) {}
func Midhinge(input Float64Data) (float64, error) {}
func Min(input Float64Data) (min float64, err error) {}
func MinkowskiDistance(dataPointX, dataPointY Float64Data, lambda float64) (distance float64, err error) {}
func Mode(input Float64Data) (mode []float64, err error) {}
func MovingAverage(input Float64Data, window int) ([]float64, error) {}
func MovingMax(input Float64Data, window int) ([]float64, error) {}
func MovingMedian(input Float64Data, window int) ([]float64, error) {}
func MovingMin(input Float64Data, window int) ([]float64, error) {}
func MovingStdDev(input Float64Data, window int) ([]float64, error) {}
func MovingSum(input Float64Data, window int) ([]float64, error) {}
func NormBoxMullerRvs(loc float64, scale float64, size int) []float64 {}
func NormCdf(x float64, loc float64, scale float64) float64 {}
func NormEntropy(loc float64, scale float64) float64 {}
func NormFit(data []float64) [2]float64{}
func NormInterval(alpha float64, loc float64,  scale float64 ) [2]float64 {}
func NormIsf(p float64, loc float64, scale float64) (x float64) {}
func NormLogCdf(x float64, loc float64, scale float64) float64 {}
func NormLogPdf(x float64, loc float64, scale float64) float64 {}
func NormLogSf(x float64, loc float64, scale float64) float64 {}
func NormMean(loc float64, scale float64) float64 {}
func NormMedian(loc float64, scale float64) float64 {}
func NormMoment(n int, loc float64, scale float64) float64 {}
func NormPdf(x float64, loc float64, scale float64) float64 {}
func NormPpf(p float64, loc float64, scale float64) (x float64) {}
func NormSample(loc float64, scale float64, size int) []float64 {}
func NormPpfRvs(loc float64, scale float64, size int) []float64 {}
func NormSf(x float64, loc float64, scale float64) float64 {}
func NormStats(loc float64, scale float64, moments string) []float64 {}
func NormStd(loc float64, scale float64) float64 {}
func NormVar(loc float64, scale float64) float64 {}
func Pearson(data1, data2 Float64Data) (float64, error) {}
func PercentChange(input Float64Data) ([]float64, error) {}
func Percentile(input Float64Data, percent float64) (percentile float64, err error) {}
func PercentileNearestRank(input Float64Data, percent float64) (percentile float64, err error) {}
func PercentileOfScore(input Float64Data, score float64) (float64, error) {}
func PercentileWeighted(data, weights Float64Data, percent float64) (percentile float64, err error) {}
func PopulationKurtosis(input Float64Data) (float64, error) {}
func PopulationSkewness(input Float64Data) (float64, error) {}
func PopulationVariance(input Float64Data) (pvar float64, err error) {}
func Product(input Float64Data) (float64, error) {}
func RMS(input Float64Data) (float64, error) {}
func Range(input Float64Data) (float64, error) {}
func Rank(input Float64Data) ([]float64, error) {}
func Rescale(input Float64Data) ([]float64, error) {}
func SEM(input Float64Data) (float64, error) {}
func Sample(input Float64Data, takenum int, replacement bool) ([]float64, error) {}
func SampleKurtosis(input Float64Data) (float64, error) {}
func SampleSkewness(input Float64Data) (float64, error) {}
func SampleVariance(input Float64Data) (svar float64, err error) {}
func Skewness(input Float64Data) (float64, error) {}
func Spearman(data1, data2 Float64Data) (float64, error) {}
func Sigmoid(input Float64Data) ([]float64, error) {}
func SoftMax(input Float64Data) ([]float64, error) {}
func StableSample(input Float64Data, takenum int) ([]float64, error) {}
func StandardDeviation(input Float64Data) (sdev float64, err error) {}
func StandardDeviationPopulation(input Float64Data) (sdev float64, err error) {}
func StandardDeviationSample(input Float64Data) (sdev float64, err error) {}
func StdDevP(input Float64Data) (sdev float64, err error) {}
func StdDevS(input Float64Data) (sdev float64, err error) {}
func Sum(input Float64Data) (sum float64, err error) {}
func TTest(data1, data2 Float64Data, populationMean float64) (t float64, pvalue float64, err error) {}
func Trimean(input Float64Data) (float64, error) {}
func TrimmedMean(input Float64Data, percent float64) (float64, error) {}
func VarP(input Float64Data) (sdev float64, err error) {}
func VarS(input Float64Data) (sdev float64, err error) {}
func Variance(input Float64Data) (sdev float64, err error) {}
func WeightedMean(data, weights Float64Data) (float64, error) {}
func Winsorize(input Float64Data, percent float64) ([]float64, error) {}
func ZScore(input Float64Data) ([]float64, error) {}
func ZTest(data1, data2 Float64Data, populationMean, populationStdDev float64) (z float64, pvalue float64, err error) {}
func ProbGeom(a int, b int, p float64) (prob float64, err error) {}
func ExpGeom(p float64) (exp float64, err error) {}
func VarGeom(p float64) (exp float64, err error) {}

type Coordinate struct {
    X, Y float64
}

type Series []Coordinate

func ExponentialRegression(s Series) (regressions Series, err error) {}
func LinearRegression(s Series) (regressions Series, err error) {}
func LogarithmicRegression(s Series) (regressions Series, err error) {}

type Outliers struct {
    Mild    Float64Data
    Extreme Float64Data
}

type Quartiles struct {
    Q1 float64
    Q2 float64
    Q3 float64
}

func Quartile(input Float64Data) (Quartiles, error) {}
func QuartileOutliers(input Float64Data) (Outliers, error) {}
```

## Contributing

Pull request are always welcome no matter how big or small. I've included a [Makefile](https://github.com/montanaflynn/stats/blob/master/Makefile) that has a lot of helper targets for common actions such as linting, testing, code coverage reporting and more.

1. Fork the repo and clone your fork
2. Create new branch (`git checkout -b some-thing`)
3. Make the desired changes
4. Ensure tests pass (`go test -cover` or `make test`)
5. Run lint and fix problems (`go vet .` or `make lint`)
6. Commit changes (`git commit -am 'Did something'`)
7. Push branch (`git push origin some-thing`)
8. Submit pull request

To make things as seamless as possible please also consider the following steps:

- Update `examples/functions/main.go` with a simple example of the new feature
- Update `README.md` documentation section with any new exported API
- Keep 100% code coverage (you can check with `make coverage`)
- Squash commits into single units of work with `git rebase -i new-feature`

## Releasing

Releases are automated with [GoReleaser](https://goreleaser.com/) via GitHub Actions. To create a new release, run the release target with the next version tag:

```
make release TAG=v0.x.x
```

This updates `CHANGELOG.md`, commits it, creates an annotated tag, and pushes the commit and tag to GitHub, where the release workflow publishes the release.

## MIT License

Copyright (c) 2014-2026 Montana Flynn (https://montanaflynn.com)

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

[action-url]: https://github.com/montanaflynn/stats/actions
[action-svg]: https://img.shields.io/github/actions/workflow/status/montanaflynn/stats/go.yml

[codecov-url]: https://app.codecov.io/gh/montanaflynn/stats
[codecov-svg]: https://img.shields.io/codecov/c/github/montanaflynn/stats?token=wnw8dActnH

[pkggodev-url]: https://pkg.go.dev/github.com/montanaflynn/stats
[pkggodev-svg]: https://gistcdn.githack.com/montanaflynn/b02f1d78d8c0de8435895d7e7cd0d473/raw/17f2a5a69f1323ecd42c00e0683655da96d9ecc8/badge.svg

[license-url]: https://github.com/montanaflynn/stats/blob/master/LICENSE
[license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
