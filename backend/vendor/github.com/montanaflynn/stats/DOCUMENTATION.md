

# stats
`import "github.com/montanaflynn/stats"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>
Package stats is a well tested and comprehensive
statistics library package with no dependencies.

Example Usage:


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

MIT License Copyright (c) 2014-2026 Montana Flynn (<a href="https://montanaflynn.com">https://montanaflynn.com</a>)




## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [func ArgMax(input Float64Data) (int, error)](#ArgMax)
* [func ArgMin(input Float64Data) (int, error)](#ArgMin)
* [func AutoCorrelation(data Float64Data, lags int) (float64, error)](#AutoCorrelation)
* [func ChebyshevDistance(dataPointX, dataPointY Float64Data) (distance float64, err error)](#ChebyshevDistance)
* [func Clip(input Float64Data, min, max float64) ([]float64, error)](#Clip)
* [func CoefficientOfVariation(input Float64Data) (float64, error)](#CoefficientOfVariation)
* [func Correlation(data1, data2 Float64Data) (float64, error)](#Correlation)
* [func Covariance(data1, data2 Float64Data) (float64, error)](#Covariance)
* [func CovariancePopulation(data1, data2 Float64Data) (float64, error)](#CovariancePopulation)
* [func CumulativeMax(input Float64Data) ([]float64, error)](#CumulativeMax)
* [func CumulativeMin(input Float64Data) ([]float64, error)](#CumulativeMin)
* [func CumulativeProduct(input Float64Data) ([]float64, error)](#CumulativeProduct)
* [func CumulativeSum(input Float64Data) ([]float64, error)](#CumulativeSum)
* [func Diff(input Float64Data) ([]float64, error)](#Diff)
* [func EWMA(input Float64Data, alpha float64) ([]float64, error)](#EWMA)
* [func Entropy(input Float64Data) (float64, error)](#Entropy)
* [func EuclideanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error)](#EuclideanDistance)
* [func ExpGeom(p float64) (exp float64, err error)](#ExpGeom)
* [func GeometricMean(input Float64Data) (float64, error)](#GeometricMean)
* [func HarmonicMean(input Float64Data) (float64, error)](#HarmonicMean)
* [func Histogram(input Float64Data, bins int) ([]int, []float64, error)](#Histogram)
* [func InterQuartileRange(input Float64Data) (float64, error)](#InterQuartileRange)
* [func Interp(x, xp, fp Float64Data) ([]float64, error)](#Interp)
* [func KendallTau(data1, data2 Float64Data) (float64, error)](#KendallTau)
* [func Kurtosis(input Float64Data) (float64, error)](#Kurtosis)
* [func ManhattanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error)](#ManhattanDistance)
* [func Max(input Float64Data) (max float64, err error)](#Max)
* [func Mean(input Float64Data) (float64, error)](#Mean)
* [func Median(input Float64Data) (median float64, err error)](#Median)
* [func MedianAbsoluteDeviation(input Float64Data) (mad float64, err error)](#MedianAbsoluteDeviation)
* [func MedianAbsoluteDeviationPopulation(input Float64Data) (mad float64, err error)](#MedianAbsoluteDeviationPopulation)
* [func Midhinge(input Float64Data) (float64, error)](#Midhinge)
* [func Min(input Float64Data) (min float64, err error)](#Min)
* [func MinkowskiDistance(dataPointX, dataPointY Float64Data, lambda float64) (distance float64, err error)](#MinkowskiDistance)
* [func Mode(input Float64Data) (mode []float64, err error)](#Mode)
* [func MovingAverage(input Float64Data, window int) ([]float64, error)](#MovingAverage)
* [func MovingMax(input Float64Data, window int) ([]float64, error)](#MovingMax)
* [func MovingMedian(input Float64Data, window int) ([]float64, error)](#MovingMedian)
* [func MovingMin(input Float64Data, window int) ([]float64, error)](#MovingMin)
* [func MovingStdDev(input Float64Data, window int) ([]float64, error)](#MovingStdDev)
* [func MovingSum(input Float64Data, window int) ([]float64, error)](#MovingSum)
* [func Ncr(n, r int) int](#Ncr)
* [func NormBoxMullerRvs(loc float64, scale float64, size int) []float64](#NormBoxMullerRvs)
* [func NormCdf(x float64, loc float64, scale float64) float64](#NormCdf)
* [func NormEntropy(loc float64, scale float64) float64](#NormEntropy)
* [func NormFit(data []float64) [2]float64](#NormFit)
* [func NormInterval(alpha float64, loc float64, scale float64) [2]float64](#NormInterval)
* [func NormIsf(p float64, loc float64, scale float64) float64](#NormIsf)
* [func NormLogCdf(x float64, loc float64, scale float64) float64](#NormLogCdf)
* [func NormLogPdf(x float64, loc float64, scale float64) float64](#NormLogPdf)
* [func NormLogSf(x float64, loc float64, scale float64) float64](#NormLogSf)
* [func NormMean(loc float64, scale float64) float64](#NormMean)
* [func NormMedian(loc float64, scale float64) float64](#NormMedian)
* [func NormMoment(n int, loc float64, scale float64) float64](#NormMoment)
* [func NormPdf(x float64, loc float64, scale float64) float64](#NormPdf)
* [func NormPpf(p float64, loc float64, scale float64) (x float64)](#NormPpf)
* [func NormPpfRvs(loc float64, scale float64, size int) []float64](#NormPpfRvs)
* [func NormSample(loc float64, scale float64, size int) []float64](#NormSample)
* [func NormSf(x float64, loc float64, scale float64) float64](#NormSf)
* [func NormStats(loc float64, scale float64, moments string) []float64](#NormStats)
* [func NormStd(loc float64, scale float64) float64](#NormStd)
* [func NormVar(loc float64, scale float64) float64](#NormVar)
* [func Pearson(data1, data2 Float64Data) (float64, error)](#Pearson)
* [func PercentChange(input Float64Data) ([]float64, error)](#PercentChange)
* [func Percentile(input Float64Data, percent float64) (percentile float64, err error)](#Percentile)
* [func PercentileNearestRank(input Float64Data, percent float64) (percentile float64, err error)](#PercentileNearestRank)
* [func PercentileOfScore(input Float64Data, score float64) (float64, error)](#PercentileOfScore)
* [func PercentileWeighted(data, weights Float64Data, percent float64) (percentile float64, err error)](#PercentileWeighted)
* [func PopulationKurtosis(input Float64Data) (float64, error)](#PopulationKurtosis)
* [func PopulationSkewness(input Float64Data) (float64, error)](#PopulationSkewness)
* [func PopulationVariance(input Float64Data) (pvar float64, err error)](#PopulationVariance)
* [func ProbGeom(a int, b int, p float64) (prob float64, err error)](#ProbGeom)
* [func Product(input Float64Data) (float64, error)](#Product)
* [func RMS(input Float64Data) (float64, error)](#RMS)
* [func Range(input Float64Data) (float64, error)](#Range)
* [func Rank(input Float64Data) ([]float64, error)](#Rank)
* [func Rescale(input Float64Data) ([]float64, error)](#Rescale)
* [func Round(input float64, places int) (rounded float64, err error)](#Round)
* [func SEM(input Float64Data) (float64, error)](#SEM)
* [func Sample(input Float64Data, takenum int, replacement bool) ([]float64, error)](#Sample)
* [func SampleKurtosis(input Float64Data) (float64, error)](#SampleKurtosis)
* [func SampleSkewness(input Float64Data) (float64, error)](#SampleSkewness)
* [func SampleVariance(input Float64Data) (svar float64, err error)](#SampleVariance)
* [func Sigmoid(input Float64Data) ([]float64, error)](#Sigmoid)
* [func Skewness(input Float64Data) (float64, error)](#Skewness)
* [func SoftMax(input Float64Data) ([]float64, error)](#SoftMax)
* [func Spearman(data1, data2 Float64Data) (float64, error)](#Spearman)
* [func StableSample(input Float64Data, takenum int) ([]float64, error)](#StableSample)
* [func StandardDeviation(input Float64Data) (sdev float64, err error)](#StandardDeviation)
* [func StandardDeviationPopulation(input Float64Data) (sdev float64, err error)](#StandardDeviationPopulation)
* [func StandardDeviationSample(input Float64Data) (sdev float64, err error)](#StandardDeviationSample)
* [func StdDevP(input Float64Data) (sdev float64, err error)](#StdDevP)
* [func StdDevS(input Float64Data) (sdev float64, err error)](#StdDevS)
* [func Sum(input Float64Data) (sum float64, err error)](#Sum)
* [func TTest(data1, data2 Float64Data, populationMean float64) (t float64, pvalue float64, err error)](#TTest)
* [func Trimean(input Float64Data) (float64, error)](#Trimean)
* [func TrimmedMean(input Float64Data, percent float64) (float64, error)](#TrimmedMean)
* [func VarGeom(p float64) (exp float64, err error)](#VarGeom)
* [func VarP(input Float64Data) (sdev float64, err error)](#VarP)
* [func VarS(input Float64Data) (sdev float64, err error)](#VarS)
* [func Variance(input Float64Data) (sdev float64, err error)](#Variance)
* [func WeightedMean(data, weights Float64Data) (float64, error)](#WeightedMean)
* [func Winsorize(input Float64Data, percent float64) ([]float64, error)](#Winsorize)
* [func ZScore(input Float64Data) ([]float64, error)](#ZScore)
* [func ZTest(data1, data2 Float64Data, populationMean, populationStdDev float64) (z float64, pvalue float64, err error)](#ZTest)
* [type Coordinate](#Coordinate)
  * [func ExpReg(s []Coordinate) (regressions []Coordinate, err error)](#ExpReg)
  * [func LinReg(s []Coordinate) (regressions []Coordinate, err error)](#LinReg)
  * [func LogReg(s []Coordinate) (regressions []Coordinate, err error)](#LogReg)
* [type Description](#Description)
  * [func Describe(input Float64Data, allowNaN bool, percentiles *[]float64) (*Description, error)](#Describe)
  * [func DescribePercentileFunc(input Float64Data, allowNaN bool, percentiles *[]float64, percentileFunc func(Float64Data, float64) (float64, error)) (*Description, error)](#DescribePercentileFunc)
  * [func (d *Description) String(decimals int) string](#Description.String)
* [type Float64Data](#Float64Data)
  * [func LoadRawData(raw interface{}) (f Float64Data)](#LoadRawData)
  * [func (f Float64Data) ArgMax() (int, error)](#Float64Data.ArgMax)
  * [func (f Float64Data) ArgMin() (int, error)](#Float64Data.ArgMin)
  * [func (f Float64Data) AutoCorrelation(lags int) (float64, error)](#Float64Data.AutoCorrelation)
  * [func (f Float64Data) Clip(min, max float64) ([]float64, error)](#Float64Data.Clip)
  * [func (f Float64Data) CoefficientOfVariation() (float64, error)](#Float64Data.CoefficientOfVariation)
  * [func (f Float64Data) Correlation(d Float64Data) (float64, error)](#Float64Data.Correlation)
  * [func (f Float64Data) Covariance(d Float64Data) (float64, error)](#Float64Data.Covariance)
  * [func (f Float64Data) CovariancePopulation(d Float64Data) (float64, error)](#Float64Data.CovariancePopulation)
  * [func (f Float64Data) CumulativeMax() ([]float64, error)](#Float64Data.CumulativeMax)
  * [func (f Float64Data) CumulativeMin() ([]float64, error)](#Float64Data.CumulativeMin)
  * [func (f Float64Data) CumulativeProduct() ([]float64, error)](#Float64Data.CumulativeProduct)
  * [func (f Float64Data) CumulativeSum() ([]float64, error)](#Float64Data.CumulativeSum)
  * [func (f Float64Data) Diff() ([]float64, error)](#Float64Data.Diff)
  * [func (f Float64Data) EWMA(alpha float64) ([]float64, error)](#Float64Data.EWMA)
  * [func (f Float64Data) Entropy() (float64, error)](#Float64Data.Entropy)
  * [func (f Float64Data) GeometricMean() (float64, error)](#Float64Data.GeometricMean)
  * [func (f Float64Data) Get(i int) float64](#Float64Data.Get)
  * [func (f Float64Data) HarmonicMean() (float64, error)](#Float64Data.HarmonicMean)
  * [func (f Float64Data) Histogram(bins int) ([]int, []float64, error)](#Float64Data.Histogram)
  * [func (f Float64Data) InterQuartileRange() (float64, error)](#Float64Data.InterQuartileRange)
  * [func (f Float64Data) KendallTau(d Float64Data) (float64, error)](#Float64Data.KendallTau)
  * [func (f Float64Data) Kurtosis() (float64, error)](#Float64Data.Kurtosis)
  * [func (f Float64Data) Len() int](#Float64Data.Len)
  * [func (f Float64Data) Less(i, j int) bool](#Float64Data.Less)
  * [func (f Float64Data) Max() (float64, error)](#Float64Data.Max)
  * [func (f Float64Data) Mean() (float64, error)](#Float64Data.Mean)
  * [func (f Float64Data) Median() (float64, error)](#Float64Data.Median)
  * [func (f Float64Data) MedianAbsoluteDeviation() (float64, error)](#Float64Data.MedianAbsoluteDeviation)
  * [func (f Float64Data) MedianAbsoluteDeviationPopulation() (float64, error)](#Float64Data.MedianAbsoluteDeviationPopulation)
  * [func (f Float64Data) Midhinge(d Float64Data) (float64, error)](#Float64Data.Midhinge)
  * [func (f Float64Data) Min() (float64, error)](#Float64Data.Min)
  * [func (f Float64Data) Mode() ([]float64, error)](#Float64Data.Mode)
  * [func (f Float64Data) MovingAverage(window int) ([]float64, error)](#Float64Data.MovingAverage)
  * [func (f Float64Data) MovingMax(window int) ([]float64, error)](#Float64Data.MovingMax)
  * [func (f Float64Data) MovingMedian(window int) ([]float64, error)](#Float64Data.MovingMedian)
  * [func (f Float64Data) MovingMin(window int) ([]float64, error)](#Float64Data.MovingMin)
  * [func (f Float64Data) MovingStdDev(window int) ([]float64, error)](#Float64Data.MovingStdDev)
  * [func (f Float64Data) MovingSum(window int) ([]float64, error)](#Float64Data.MovingSum)
  * [func (f Float64Data) Pearson(d Float64Data) (float64, error)](#Float64Data.Pearson)
  * [func (f Float64Data) PercentChange() ([]float64, error)](#Float64Data.PercentChange)
  * [func (f Float64Data) Percentile(p float64) (float64, error)](#Float64Data.Percentile)
  * [func (f Float64Data) PercentileNearestRank(p float64) (float64, error)](#Float64Data.PercentileNearestRank)
  * [func (f Float64Data) PercentileOfScore(score float64) (float64, error)](#Float64Data.PercentileOfScore)
  * [func (f Float64Data) PopulationKurtosis() (float64, error)](#Float64Data.PopulationKurtosis)
  * [func (f Float64Data) PopulationVariance() (float64, error)](#Float64Data.PopulationVariance)
  * [func (f Float64Data) Product() (float64, error)](#Float64Data.Product)
  * [func (f Float64Data) Quartile(d Float64Data) (Quartiles, error)](#Float64Data.Quartile)
  * [func (f Float64Data) QuartileOutliers() (Outliers, error)](#Float64Data.QuartileOutliers)
  * [func (f Float64Data) Quartiles() (Quartiles, error)](#Float64Data.Quartiles)
  * [func (f Float64Data) RMS() (float64, error)](#Float64Data.RMS)
  * [func (f Float64Data) Range() (float64, error)](#Float64Data.Range)
  * [func (f Float64Data) Rank() ([]float64, error)](#Float64Data.Rank)
  * [func (f Float64Data) Rescale() ([]float64, error)](#Float64Data.Rescale)
  * [func (f Float64Data) SEM() (float64, error)](#Float64Data.SEM)
  * [func (f Float64Data) Sample(n int, r bool) ([]float64, error)](#Float64Data.Sample)
  * [func (f Float64Data) SampleKurtosis() (float64, error)](#Float64Data.SampleKurtosis)
  * [func (f Float64Data) SampleVariance() (float64, error)](#Float64Data.SampleVariance)
  * [func (f Float64Data) Sigmoid() ([]float64, error)](#Float64Data.Sigmoid)
  * [func (f Float64Data) SoftMax() ([]float64, error)](#Float64Data.SoftMax)
  * [func (f Float64Data) Spearman(d Float64Data) (float64, error)](#Float64Data.Spearman)
  * [func (f Float64Data) StandardDeviation() (float64, error)](#Float64Data.StandardDeviation)
  * [func (f Float64Data) StandardDeviationPopulation() (float64, error)](#Float64Data.StandardDeviationPopulation)
  * [func (f Float64Data) StandardDeviationSample() (float64, error)](#Float64Data.StandardDeviationSample)
  * [func (f Float64Data) Sum() (float64, error)](#Float64Data.Sum)
  * [func (f Float64Data) Swap(i, j int)](#Float64Data.Swap)
  * [func (f Float64Data) Trimean(d Float64Data) (float64, error)](#Float64Data.Trimean)
  * [func (f Float64Data) TrimmedMean(percent float64) (float64, error)](#Float64Data.TrimmedMean)
  * [func (f Float64Data) Variance() (float64, error)](#Float64Data.Variance)
  * [func (f Float64Data) WeightedMean(weights Float64Data) (float64, error)](#Float64Data.WeightedMean)
  * [func (f Float64Data) Winsorize(percent float64) ([]float64, error)](#Float64Data.Winsorize)
  * [func (f Float64Data) ZScore() ([]float64, error)](#Float64Data.ZScore)
* [type Outliers](#Outliers)
  * [func QuartileOutliers(input Float64Data) (Outliers, error)](#QuartileOutliers)
* [type Quartiles](#Quartiles)
  * [func Quartile(input Float64Data) (Quartiles, error)](#Quartile)
* [type Series](#Series)
  * [func ExponentialRegression(s Series) (regressions Series, err error)](#ExponentialRegression)
  * [func LinearRegression(s Series) (regressions Series, err error)](#LinearRegression)
  * [func LogarithmicRegression(s Series) (regressions Series, err error)](#LogarithmicRegression)

#### <a name="pkg-examples">Examples</a>
* [ArgMax](#example_ArgMax)
* [ArgMin](#example_ArgMin)
* [AutoCorrelation](#example_AutoCorrelation)
* [ChebyshevDistance](#example_ChebyshevDistance)
* [Clip](#example_Clip)
* [Correlation](#example_Correlation)
* [CumulativeMax](#example_CumulativeMax)
* [CumulativeMin](#example_CumulativeMin)
* [CumulativeProduct](#example_CumulativeProduct)
* [CumulativeSum](#example_CumulativeSum)
* [Diff](#example_Diff)
* [EWMA](#example_EWMA)
* [Entropy](#example_Entropy)
* [ExpGeom](#example_ExpGeom)
* [Histogram](#example_Histogram)
* [Interp](#example_Interp)
* [KendallTau](#example_KendallTau)
* [Kurtosis](#example_Kurtosis)
* [LinearRegression](#example_LinearRegression)
* [LoadRawData](#example_LoadRawData)
* [Max](#example_Max)
* [Median](#example_Median)
* [Min](#example_Min)
* [MovingAverage](#example_MovingAverage)
* [MovingMax](#example_MovingMax)
* [MovingMedian](#example_MovingMedian)
* [MovingMin](#example_MovingMin)
* [MovingStdDev](#example_MovingStdDev)
* [MovingSum](#example_MovingSum)
* [PercentChange](#example_PercentChange)
* [PercentileOfScore](#example_PercentileOfScore)
* [ProbGeom](#example_ProbGeom)
* [Product](#example_Product)
* [RMS](#example_RMS)
* [Range](#example_Range)
* [Rank](#example_Rank)
* [Rescale](#example_Rescale)
* [Round](#example_Round)
* [SEM](#example_SEM)
* [SampleKurtosis](#example_SampleKurtosis)
* [Sigmoid](#example_Sigmoid)
* [SoftMax](#example_SoftMax)
* [Spearman](#example_Spearman)
* [Sum](#example_Sum)
* [TrimmedMean](#example_TrimmedMean)
* [VarGeom](#example_VarGeom)
* [Winsorize](#example_Winsorize)
* [ZScore](#example_ZScore)

#### <a name="pkg-files">Package files</a>
[clip.go](/src/github.com/montanaflynn/stats/clip.go) [coefficient_of_variation.go](/src/github.com/montanaflynn/stats/coefficient_of_variation.go) [correlation.go](/src/github.com/montanaflynn/stats/correlation.go) [cumulative.go](/src/github.com/montanaflynn/stats/cumulative.go) [cumulative_sum.go](/src/github.com/montanaflynn/stats/cumulative_sum.go) [data.go](/src/github.com/montanaflynn/stats/data.go) [describe.go](/src/github.com/montanaflynn/stats/describe.go) [deviation.go](/src/github.com/montanaflynn/stats/deviation.go) [diff.go](/src/github.com/montanaflynn/stats/diff.go) [distances.go](/src/github.com/montanaflynn/stats/distances.go) [doc.go](/src/github.com/montanaflynn/stats/doc.go) [entropy.go](/src/github.com/montanaflynn/stats/entropy.go) [errors.go](/src/github.com/montanaflynn/stats/errors.go) [ewma.go](/src/github.com/montanaflynn/stats/ewma.go) [extremes.go](/src/github.com/montanaflynn/stats/extremes.go) [geometric_distribution.go](/src/github.com/montanaflynn/stats/geometric_distribution.go) [histogram.go](/src/github.com/montanaflynn/stats/histogram.go) [interp.go](/src/github.com/montanaflynn/stats/interp.go) [kendall.go](/src/github.com/montanaflynn/stats/kendall.go) [kurtosis.go](/src/github.com/montanaflynn/stats/kurtosis.go) [legacy.go](/src/github.com/montanaflynn/stats/legacy.go) [load.go](/src/github.com/montanaflynn/stats/load.go) [max.go](/src/github.com/montanaflynn/stats/max.go) [mean.go](/src/github.com/montanaflynn/stats/mean.go) [median.go](/src/github.com/montanaflynn/stats/median.go) [min.go](/src/github.com/montanaflynn/stats/min.go) [mode.go](/src/github.com/montanaflynn/stats/mode.go) [moving.go](/src/github.com/montanaflynn/stats/moving.go) [norm.go](/src/github.com/montanaflynn/stats/norm.go) [outlier.go](/src/github.com/montanaflynn/stats/outlier.go) [percentile.go](/src/github.com/montanaflynn/stats/percentile.go) [percentile_of_score.go](/src/github.com/montanaflynn/stats/percentile_of_score.go) [percentile_weighted.go](/src/github.com/montanaflynn/stats/percentile_weighted.go) [product.go](/src/github.com/montanaflynn/stats/product.go) [quartile.go](/src/github.com/montanaflynn/stats/quartile.go) [rank.go](/src/github.com/montanaflynn/stats/rank.go) [ranksum.go](/src/github.com/montanaflynn/stats/ranksum.go) [regression.go](/src/github.com/montanaflynn/stats/regression.go) [rescale.go](/src/github.com/montanaflynn/stats/rescale.go) [rms.go](/src/github.com/montanaflynn/stats/rms.go) [rolling.go](/src/github.com/montanaflynn/stats/rolling.go) [round.go](/src/github.com/montanaflynn/stats/round.go) [sample.go](/src/github.com/montanaflynn/stats/sample.go) [sem.go](/src/github.com/montanaflynn/stats/sem.go) [sigmoid.go](/src/github.com/montanaflynn/stats/sigmoid.go) [skewness.go](/src/github.com/montanaflynn/stats/skewness.go) [softmax.go](/src/github.com/montanaflynn/stats/softmax.go) [sum.go](/src/github.com/montanaflynn/stats/sum.go) [trimmed_mean.go](/src/github.com/montanaflynn/stats/trimmed_mean.go) [ttest.go](/src/github.com/montanaflynn/stats/ttest.go) [util.go](/src/github.com/montanaflynn/stats/util.go) [variance.go](/src/github.com/montanaflynn/stats/variance.go) [weighted_mean.go](/src/github.com/montanaflynn/stats/weighted_mean.go) [winsorize.go](/src/github.com/montanaflynn/stats/winsorize.go) [zscore.go](/src/github.com/montanaflynn/stats/zscore.go) [ztest.go](/src/github.com/montanaflynn/stats/ztest.go) 



## <a name="pkg-variables">Variables</a>
``` go
var (
    // ErrEmptyInput Input must not be empty
    ErrEmptyInput = statsError{"Input must not be empty."}
    // ErrNaN Not a number
    ErrNaN = statsError{"Not a number."}
    // ErrNegative Must not contain negative values
    ErrNegative = statsError{"Must not contain negative values."}
    // ErrZero Must not contain zero values
    ErrZero = statsError{"Must not contain zero values."}
    // ErrBounds Input is outside of range
    ErrBounds = statsError{"Input is outside of range."}
    // ErrSize Must be the same length
    ErrSize = statsError{"Must be the same length."}
    // ErrInfValue Value is infinite
    ErrInfValue = statsError{"Value is infinite."}
    // ErrYCoord Y Value must be greater than zero
    ErrYCoord = statsError{"Y Value must be greater than zero."}
)
```
These are the package-wide error values.
All error identification should use these values.
<a href="https://github.com/golang/go/wiki/Errors#naming">https://github.com/golang/go/wiki/Errors#naming</a>

``` go
var (
    EmptyInputErr = ErrEmptyInput
    NaNErr        = ErrNaN
    NegativeErr   = ErrNegative
    ZeroErr       = ErrZero
    BoundsErr     = ErrBounds
    SizeErr       = ErrSize
    InfValue      = ErrInfValue
    YCoordErr     = ErrYCoord
    EmptyInput    = ErrEmptyInput
)
```
Legacy error names that didn't start with Err



## <a name="ArgMax">func</a> [ArgMax](/extremes.go?s=129:172#L5)
``` go
func ArgMax(input Float64Data) (int, error)
```
ArgMax finds the index of the highest number in a slice,
returning the first occurrence in the case of ties



## <a name="ArgMin">func</a> [ArgMin](/extremes.go?s=671:714#L28)
``` go
func ArgMin(input Float64Data) (int, error)
```
ArgMin finds the index of the lowest number in a slice,
returning the first occurrence in the case of ties



## <a name="AutoCorrelation">func</a> [AutoCorrelation](/correlation.go?s=2282:2347#L102)
``` go
func AutoCorrelation(data Float64Data, lags int) (float64, error)
```
AutoCorrelation is the correlation of a signal with a delayed copy of itself as a function of delay



## <a name="ChebyshevDistance">func</a> [ChebyshevDistance](/distances.go?s=368:456#L20)
``` go
func ChebyshevDistance(dataPointX, dataPointY Float64Data) (distance float64, err error)
```
ChebyshevDistance computes the Chebyshev distance between two data sets



## <a name="Clip">func</a> [Clip](/clip.go?s=109:174#L5)
``` go
func Clip(input Float64Data, min, max float64) ([]float64, error)
```
Clip clamps each value in the input slice into the
inclusive range between min and max.



## <a name="CoefficientOfVariation">func</a> [CoefficientOfVariation](/coefficient_of_variation.go?s=322:385#L11)
``` go
func CoefficientOfVariation(input Float64Data) (float64, error)
```
CoefficientOfVariation finds the coefficient of variation of a slice
of floats, defined as the sample standard deviation divided by the
mean. This matches the behavior of Python's scipy.stats.variation
with ddof=1.

The input must not be empty and its mean must not be zero.



## <a name="Correlation">func</a> [Correlation](/correlation.go?s=120:179#L9)
``` go
func Correlation(data1, data2 Float64Data) (float64, error)
```
Correlation describes the degree of relationship between two sets of data



## <a name="Covariance">func</a> [Covariance](/variance.go?s=1284:1342#L53)
``` go
func Covariance(data1, data2 Float64Data) (float64, error)
```
Covariance is a measure of how much two sets of data change



## <a name="CovariancePopulation">func</a> [CovariancePopulation](/variance.go?s=1864:1932#L81)
``` go
func CovariancePopulation(data1, data2 Float64Data) (float64, error)
```
CovariancePopulation computes covariance for entire population between two variables.



## <a name="CumulativeMax">func</a> [CumulativeMax](/cumulative.go?s=486:542#L24)
``` go
func CumulativeMax(input Float64Data) ([]float64, error)
```
CumulativeMax calculates the cumulative maximum of the input slice



## <a name="CumulativeMin">func</a> [CumulativeMin](/cumulative.go?s=874:930#L44)
``` go
func CumulativeMin(input Float64Data) ([]float64, error)
```
CumulativeMin calculates the cumulative minimum of the input slice



## <a name="CumulativeProduct">func</a> [CumulativeProduct](/cumulative.go?s=89:149#L4)
``` go
func CumulativeProduct(input Float64Data) ([]float64, error)
```
CumulativeProduct calculates the cumulative product of the input slice



## <a name="CumulativeSum">func</a> [CumulativeSum](/cumulative_sum.go?s=81:137#L4)
``` go
func CumulativeSum(input Float64Data) ([]float64, error)
```
CumulativeSum calculates the cumulative sum of the input slice



## <a name="Diff">func</a> [Diff](/diff.go?s=238:285#L7)
``` go
func Diff(input Float64Data) ([]float64, error)
```
Diff calculates the successive differences of the input slice,
returning input[i] - input[i-1] for each i in 1..len(input)-1.
The output has length len(input) - 1; a single-element input
returns an empty slice.



## <a name="EWMA">func</a> [EWMA](/ewma.go?s=392:454#L9)
``` go
func EWMA(input Float64Data, alpha float64) ([]float64, error)
```
EWMA calculates the exponentially weighted moving average of the input
with smoothing factor alpha. The first output equals the first input and
each subsequent entry is alpha*input[i] + (1-alpha)*output[i-1], so the
result has the same length as the input. The alpha must satisfy
0 < alpha <= 1 or ErrBounds is returned. An empty input returns
ErrEmptyInput.



## <a name="Entropy">func</a> [Entropy](/entropy.go?s=77:125#L6)
``` go
func Entropy(input Float64Data) (float64, error)
```
Entropy provides calculation of the entropy



## <a name="EuclideanDistance">func</a> [EuclideanDistance](/distances.go?s=836:924#L36)
``` go
func EuclideanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error)
```
EuclideanDistance computes the Euclidean distance between two data sets



## <a name="ExpGeom">func</a> [ExpGeom](/geometric_distribution.go?s=816:864#L28)
``` go
func ExpGeom(p float64) (exp float64, err error)
```
ProbGeom generates the expectation or average number of trials
for a geometric random variable with parameter p



## <a name="GeometricMean">func</a> [GeometricMean](/mean.go?s=319:373#L18)
``` go
func GeometricMean(input Float64Data) (float64, error)
```
GeometricMean gets the geometric mean for a slice of numbers



## <a name="HarmonicMean">func</a> [HarmonicMean](/mean.go?s=842:895#L41)
``` go
func HarmonicMean(input Float64Data) (float64, error)
```
HarmonicMean gets the harmonic mean for a slice of numbers



## <a name="Histogram">func</a> [Histogram](/histogram.go?s=327:396#L10)
``` go
func Histogram(input Float64Data, bins int) ([]int, []float64, error)
```
Histogram calculates the histogram of a slice using the given
number of equal-width bins over [min, max], returning the count
of values in each bin along with the bins+1 bin edges. Each bin
is half-open [edges[i], edges[i+1]) except the last, which also
includes the maximum value.



## <a name="InterQuartileRange">func</a> [InterQuartileRange](/quartile.go?s=821:880#L45)
``` go
func InterQuartileRange(input Float64Data) (float64, error)
```
InterQuartileRange finds the range between Q1 and Q3



## <a name="Interp">func</a> [Interp](/interp.go?s=635:688#L16)
``` go
func Interp(x, xp, fp Float64Data) ([]float64, error)
```
Interp calculates the one-dimensional piecewise-linear interpolant to a
function with given discrete data points (xp, fp), evaluated at each x.
Values of x below xp[0] return fp[0] and values above xp[len(xp)-1] return
fp[len(xp)-1], so no extrapolation is performed. Unlike numpy's interp,
which silently returns nonsense for unsorted coordinates, xp must be
strictly increasing or ErrBounds is returned. An empty x or xp returns
ErrEmptyInput and xp and fp of different lengths return ErrSize.
A NaN in xp returns ErrBounds and a NaN in x gives a NaN in the output.



## <a name="KendallTau">func</a> [KendallTau](/kendall.go?s=302:360#L9)
``` go
func KendallTau(data1, data2 Float64Data) (float64, error)
```
KendallTau calculates Kendall's tau-b rank correlation coefficient
between two variables. Tau-b corrects for ties, matching the values
produced by SciPy's kendalltau and pandas' corr(method="kendall").
Pairs are compared with a simple O(n^2) loop for clarity.



## <a name="Kurtosis">func</a> [Kurtosis](/kurtosis.go?s=97:146#L6)
``` go
func Kurtosis(input Float64Data) (float64, error)
```
Kurtosis computes the population excess kurtosis of the dataset



## <a name="ManhattanDistance">func</a> [ManhattanDistance](/distances.go?s=1277:1365#L50)
``` go
func ManhattanDistance(dataPointX, dataPointY Float64Data) (distance float64, err error)
```
ManhattanDistance computes the Manhattan distance between two data sets



## <a name="Max">func</a> [Max](/max.go?s=78:130#L8)
``` go
func Max(input Float64Data) (max float64, err error)
```
Max finds the highest number in a slice



## <a name="Mean">func</a> [Mean](/mean.go?s=77:122#L6)
``` go
func Mean(input Float64Data) (float64, error)
```
Mean gets the average of a slice of numbers



## <a name="Median">func</a> [Median](/median.go?s=85:143#L6)
``` go
func Median(input Float64Data) (median float64, err error)
```
Median gets the median number in a slice of numbers



## <a name="MedianAbsoluteDeviation">func</a> [MedianAbsoluteDeviation](/deviation.go?s=125:197#L6)
``` go
func MedianAbsoluteDeviation(input Float64Data) (mad float64, err error)
```
MedianAbsoluteDeviation finds the median of the absolute deviations from the dataset median



## <a name="MedianAbsoluteDeviationPopulation">func</a> [MedianAbsoluteDeviationPopulation](/deviation.go?s=360:442#L11)
``` go
func MedianAbsoluteDeviationPopulation(input Float64Data) (mad float64, err error)
```
MedianAbsoluteDeviationPopulation finds the median of the absolute deviations from the population median



## <a name="Midhinge">func</a> [Midhinge](/quartile.go?s=1075:1124#L55)
``` go
func Midhinge(input Float64Data) (float64, error)
```
Midhinge finds the average of the first and third quartiles



## <a name="Min">func</a> [Min](/min.go?s=78:130#L6)
``` go
func Min(input Float64Data) (min float64, err error)
```
Min finds the lowest number in a set of data



## <a name="MinkowskiDistance">func</a> [MinkowskiDistance](/distances.go?s=2133:2237#L78)
``` go
func MinkowskiDistance(dataPointX, dataPointY Float64Data, lambda float64) (distance float64, err error)
```
MinkowskiDistance computes the Minkowski distance between two data sets

Arguments:


	dataPointX: First set of data points
	dataPointY: Second set of data points. Length of both data
	            sets must be equal.
	lambda:     aka p or city blocks; With lambda = 1
	            returned distance is manhattan distance and
	            lambda = 2; it is euclidean distance. Lambda
	            reaching to infinite - distance would be chebysev
	            distance.

Return:


	Distance or error



## <a name="Mode">func</a> [Mode](/mode.go?s=85:141#L4)
``` go
func Mode(input Float64Data) (mode []float64, err error)
```
Mode gets the mode [most frequent value(s)] of a slice of float64s



## <a name="MovingAverage">func</a> [MovingAverage](/rolling.go?s=362:430#L8)
``` go
func MovingAverage(input Float64Data, window int) ([]float64, error)
```
MovingAverage calculates the rolling mean of the input over a trailing
window. Only fully-populated windows produce output, so the result has
len(input)-window+1 entries and entry i is the mean of input[i : i+window].
The window must satisfy 1 <= window <= len(input) or ErrBounds is
returned. An empty input returns ErrEmptyInput.



## <a name="MovingMax">func</a> [MovingMax](/moving.go?s=1892:1956#L60)
``` go
func MovingMax(input Float64Data, window int) ([]float64, error)
```
MovingMax calculates the rolling maximum of the input over a trailing
window. Only fully-populated windows produce output, so the result has
len(input)-window+1 entries and entry i is the maximum of
input[i : i+window]. The window must satisfy 1 <= window <= len(input) or
ErrBounds is returned. An empty input returns ErrEmptyInput.



## <a name="MovingMedian">func</a> [MovingMedian](/moving.go?s=365:432#L8)
``` go
func MovingMedian(input Float64Data, window int) ([]float64, error)
```
MovingMedian calculates the rolling median of the input over a trailing
window. Only fully-populated windows produce output, so the result has
len(input)-window+1 entries and entry i is the median of input[i : i+window].
The window must satisfy 1 <= window <= len(input) or ErrBounds is
returned. An empty input returns ErrEmptyInput.



## <a name="MovingMin">func</a> [MovingMin](/moving.go?s=1136:1200#L34)
``` go
func MovingMin(input Float64Data, window int) ([]float64, error)
```
MovingMin calculates the rolling minimum of the input over a trailing
window. Only fully-populated windows produce output, so the result has
len(input)-window+1 entries and entry i is the minimum of
input[i : i+window]. The window must satisfy 1 <= window <= len(input) or
ErrBounds is returned. An empty input returns ErrEmptyInput.



## <a name="MovingStdDev">func</a> [MovingStdDev](/rolling.go?s=1239:1306#L36)
``` go
func MovingStdDev(input Float64Data, window int) ([]float64, error)
```
MovingStdDev calculates the rolling sample standard deviation of the input
over a trailing window. Only fully-populated windows produce output, so the
result has len(input)-window+1 entries and entry i is the sample standard
deviation of input[i : i+window]. The window must satisfy
2 <= window <= len(input) or ErrBounds is returned, since the sample
standard deviation of a single value is undefined. An empty input returns
ErrEmptyInput.



## <a name="MovingSum">func</a> [MovingSum](/moving.go?s=2640:2704#L86)
``` go
func MovingSum(input Float64Data, window int) ([]float64, error)
```
MovingSum calculates the rolling sum of the input over a trailing
window. Only fully-populated windows produce output, so the result has
len(input)-window+1 entries and entry i is the sum of input[i : i+window].
The window must satisfy 1 <= window <= len(input) or ErrBounds is
returned. An empty input returns ErrEmptyInput.



## <a name="Ncr">func</a> [Ncr](/norm.go?s=8827:8849#L277)
``` go
func Ncr(n, r int) int
```
Ncr is an N choose R algorithm.
Aaron Cannon's algorithm.



## <a name="NormBoxMullerRvs">func</a> [NormBoxMullerRvs](/norm.go?s=906:975#L29)
``` go
func NormBoxMullerRvs(loc float64, scale float64, size int) []float64
```
NormBoxMullerRvs generates random variates using the Box–Muller transform.
For more information please visit: <a href="http://mathworld.wolfram.com/Box-MullerTransformation.html">http://mathworld.wolfram.com/Box-MullerTransformation.html</a>



## <a name="NormCdf">func</a> [NormCdf](/norm.go?s=2034:2093#L59)
``` go
func NormCdf(x float64, loc float64, scale float64) float64
```
NormCdf is the cumulative distribution function.



## <a name="NormEntropy">func</a> [NormEntropy](/norm.go?s=7117:7169#L219)
``` go
func NormEntropy(loc float64, scale float64) float64
```
NormEntropy is the differential entropy of the RV.



## <a name="NormFit">func</a> [NormFit](/norm.go?s=7402:7441#L226)
``` go
func NormFit(data []float64) [2]float64
```
NormFit returns the maximum likelihood estimators for the Normal Distribution.
Takes array of float64 values.
Returns array of Mean followed by Standard Deviation.



## <a name="NormInterval">func</a> [NormInterval](/norm.go?s=8320:8391#L260)
``` go
func NormInterval(alpha float64, loc float64, scale float64) [2]float64
```
NormInterval finds endpoints of the range that contains alpha percent of the distribution.



## <a name="NormIsf">func</a> [NormIsf](/norm.go?s=5589:5648#L177)
``` go
func NormIsf(p float64, loc float64, scale float64) float64
```
NormIsf is the inverse survival function (inverse of sf).



## <a name="NormLogCdf">func</a> [NormLogCdf](/norm.go?s=2218:2280#L64)
``` go
func NormLogCdf(x float64, loc float64, scale float64) float64
```
NormLogCdf is the log of the cumulative distribution function.



## <a name="NormLogPdf">func</a> [NormLogPdf](/norm.go?s=1829:1891#L53)
``` go
func NormLogPdf(x float64, loc float64, scale float64) float64
```
NormLogPdf is the log of the probability density function.



## <a name="NormLogSf">func</a> [NormLogSf](/norm.go?s=2664:2725#L78)
``` go
func NormLogSf(x float64, loc float64, scale float64) float64
```
NormLogSf is the log of the survival function.



## <a name="NormMean">func</a> [NormMean](/norm.go?s=7904:7953#L245)
``` go
func NormMean(loc float64, scale float64) float64
```
NormMean is the mean/expected value of the distribution.



## <a name="NormMedian">func</a> [NormMedian](/norm.go?s=7775:7826#L240)
``` go
func NormMedian(loc float64, scale float64) float64
```
NormMedian is the median of the distribution.



## <a name="NormMoment">func</a> [NormMoment](/norm.go?s=6038:6096#L185)
``` go
func NormMoment(n int, loc float64, scale float64) float64
```
NormMoment approximates the non-central (raw) moment of order n.
For more information please visit: <a href="https://math.stackexchange.com/questions/1945448/methods-for-finding-raw-moments-of-the-normal-distribution">https://math.stackexchange.com/questions/1945448/methods-for-finding-raw-moments-of-the-normal-distribution</a>



## <a name="NormPdf">func</a> [NormPdf](/norm.go?s=1596:1655#L48)
``` go
func NormPdf(x float64, loc float64, scale float64) float64
```
NormPdf is the probability density function.



## <a name="NormPpf">func</a> [NormPpf](/norm.go?s=3828:3891#L107)
``` go
func NormPpf(p float64, loc float64, scale float64) (x float64)
```
NormPpf is the point percentile function.
This is based on Peter John Acklam's inverse normal CDF.
algorithm: <a href="http://home.online.no/~pjacklam/notes/invnorm/">http://home.online.no/~pjacklam/notes/invnorm/</a> (no longer visible).
For more information please visit: <a href="https://stackedboxes.org/2017/05/01/acklams-normal-quantile-function/">https://stackedboxes.org/2017/05/01/acklams-normal-quantile-function/</a>



## <a name="NormPpfRvs">func</a> [NormPpfRvs](/norm.go?s=486:549#L18)
``` go
func NormPpfRvs(loc float64, scale float64, size int) []float64
```
NormPpfRvs generates random variates using the Point Percentile Function.
For more information please visit: <a href="https://demonstrations.wolfram.com/TheMethodOfInverseTransforms/">https://demonstrations.wolfram.com/TheMethodOfInverseTransforms/</a>



## <a name="NormSample">func</a> [NormSample](/norm.go?s=194:257#L12)
``` go
func NormSample(loc float64, scale float64, size int) []float64
```
NormSample generates random samples from a normal distribution
with the given mean (loc) and standard deviation (scale).



## <a name="NormSf">func</a> [NormSf](/norm.go?s=2498:2556#L73)
``` go
func NormSf(x float64, loc float64, scale float64) float64
```
NormSf is the survival function (also defined as 1 - cdf, but sf is sometimes more accurate).



## <a name="NormStats">func</a> [NormStats](/norm.go?s=6621:6689#L201)
``` go
func NormStats(loc float64, scale float64, moments string) []float64
```
NormStats returns the mean, variance, skew, and/or kurtosis.
Mean(‘m’), variance(‘v’), skew(‘s’), and/or kurtosis(‘k’).
Takes string containing any of 'mvsk'.
Returns array of m v s k in that order.



## <a name="NormStd">func</a> [NormStd](/norm.go?s=8158:8206#L255)
``` go
func NormStd(loc float64, scale float64) float64
```
NormStd is the standard deviation of the distribution.



## <a name="NormVar">func</a> [NormVar](/norm.go?s=8019:8067#L250)
``` go
func NormVar(loc float64, scale float64) float64
```
NormVar is the variance of the distribution.



## <a name="Pearson">func</a> [Pearson](/correlation.go?s=663:718#L34)
``` go
func Pearson(data1, data2 Float64Data) (float64, error)
```
Pearson calculates the Pearson product-moment correlation coefficient between two variables



## <a name="PercentChange">func</a> [PercentChange](/diff.go?s=891:947#L29)
``` go
func PercentChange(input Float64Data) ([]float64, error)
```
PercentChange calculates the fractional change between successive
elements of the input slice, returning
(input[i] - input[i-1]) / input[i-1] for each i in 1..len(input)-1.
The output has length len(input) - 1; a single-element input
returns an empty slice. A zero denominator follows IEEE 754
semantics, yielding +Inf, -Inf, or NaN (for 0/0), matching the
behavior of pandas pct_change.



## <a name="Percentile">func</a> [Percentile](/percentile.go?s=598:681#L20)
``` go
func Percentile(input Float64Data, percent float64) (percentile float64, err error)
```
Percentile finds the relative standing in a slice of floats.

The function uses the Linear Interpolation Between Closest Ranks method
as recommended by NIST [1] and used by Excel (PERCENTILE), Google Sheets,
NumPy (default), and other standard tools.

Algorithm (for percent p and sorted data of length n):


	1. Compute the rank: rank = (p / 100) * (n - 1)
	2. Split into integer part k and fractional part f
	3. Result = data[k] + f * (data[k+1] - data[k])

[1] <a href="https://www.itl.nist.gov/div898/handbook/prc/section2/prc262.htm">https://www.itl.nist.gov/div898/handbook/prc/section2/prc262.htm</a>



## <a name="PercentileNearestRank">func</a> [PercentileNearestRank](/percentile.go?s=1405:1499#L55)
``` go
func PercentileNearestRank(input Float64Data, percent float64) (percentile float64, err error)
```
PercentileNearestRank finds the relative standing in a slice of floats using the Nearest Rank method



## <a name="PercentileOfScore">func</a> [PercentileOfScore](/percentile_of_score.go?s=374:447#L11)
``` go
func PercentileOfScore(input Float64Data, score float64) (float64, error)
```
PercentileOfScore calculates the percentile rank of a score
relative to a slice of floats, defined as the percentage of
values strictly below the score plus half the percentage of
values equal to the score. The result is between 0 and 100.
This matches the behavior of Python's
scipy.stats.percentileofscore with kind="mean".



## <a name="PercentileWeighted">func</a> [PercentileWeighted](/percentile_weighted.go?s=620:719#L19)
``` go
func PercentileWeighted(data, weights Float64Data, percent float64) (percentile float64, err error)
```
PercentileWeighted finds the weighted percentile of a slice of floats
using the weighted empirical CDF (inverse CDF / nearest-rank method).

For a given percent p, it returns the smallest data value x such that
the cumulative weight of all values <= x is at least p% of the total
weight. This matches the behavior of Python's statsmodels
DescrStatsW.quantile.

The data and weights slices must be the same length. Weights must be
non-negative and at least one weight must be positive. The percent
parameter must be between 0 and 100 (exclusive).



## <a name="PopulationKurtosis">func</a> [PopulationKurtosis](/kurtosis.go?s=391:450#L13)
``` go
func PopulationKurtosis(input Float64Data) (float64, error)
```
PopulationKurtosis computes the population excess kurtosis (Fisher
definition) using the fourth central moment normalized by the squared
variance, so a normal distribution has a kurtosis of zero.



## <a name="PopulationSkewness">func</a> [PopulationSkewness](/skewness.go?s=318:377#L12)
``` go
func PopulationSkewness(input Float64Data) (float64, error)
```
PopulationSkewness computes the population skewness using the third
central moment normalized by the cube of the standard deviation.



## <a name="PopulationVariance">func</a> [PopulationVariance](/variance.go?s=828:896#L31)
``` go
func PopulationVariance(input Float64Data) (pvar float64, err error)
```
PopulationVariance finds the amount of variance within a population



## <a name="ProbGeom">func</a> [ProbGeom](/geometric_distribution.go?s=258:322#L10)
``` go
func ProbGeom(a int, b int, p float64) (prob float64, err error)
```
ProbGeom generates the probability for a geometric random variable
with parameter p to achieve success in the interval of [a, b] trials
See <a href="https://en.wikipedia.org/wiki/Geometric_distribution">https://en.wikipedia.org/wiki/Geometric_distribution</a> for more information



## <a name="Product">func</a> [Product](/product.go?s=299:347#L10)
``` go
func Product(input Float64Data) (float64, error)
```
Product calculates the product of a slice of floats by
multiplying the values from left to right. It is the scalar
counterpart of CumulativeProduct. Large inputs can overflow
to Inf; use GeometricMean for an overflow-safe summary of
multiplicative data.



## <a name="RMS">func</a> [RMS](/rms.go?s=156:200#L7)
``` go
func RMS(input Float64Data) (float64, error)
```
RMS calculates the root mean square of a slice of floats,
defined as the square root of the mean of the squared values.



## <a name="Range">func</a> [Range](/extremes.go?s=1181:1227#L51)
``` go
func Range(input Float64Data) (float64, error)
```
Range finds the difference between the highest and
lowest numbers in a slice



## <a name="Rank">func</a> [Rank](/rank.go?s=183:230#L6)
``` go
func Rank(input Float64Data) ([]float64, error)
```
Rank assigns fractional (average) ranks to the input values.
Ranks are 1-based and tied values receive the average of the
ranks they would have been assigned.



## <a name="Rescale">func</a> [Rescale](/rescale.go?s=174:224#L6)
``` go
func Rescale(input Float64Data) ([]float64, error)
```
Rescale normalizes the input values to the range of 0 to 1
by subtracting the minimum and dividing by the range,
also known as min-max normalization.



## <a name="Round">func</a> [Round](/round.go?s=88:154#L6)
``` go
func Round(input float64, places int) (rounded float64, err error)
```
Round a float to a specific decimal place or precision



## <a name="SEM">func</a> [SEM](/sem.go?s=265:309#L9)
``` go
func SEM(input Float64Data) (float64, error)
```
SEM calculates the standard error of the mean of a slice
of floats, defined as the sample standard deviation divided
by the square root of the sample size. This matches the
behavior of Python's scipy.stats.sem with ddof=1.



## <a name="Sample">func</a> [Sample](/sample.go?s=112:192#L9)
``` go
func Sample(input Float64Data, takenum int, replacement bool) ([]float64, error)
```
Sample returns sample from input with replacement or without



## <a name="SampleKurtosis">func</a> [SampleKurtosis](/kurtosis.go?s=1071:1126#L41)
``` go
func SampleKurtosis(input Float64Data) (float64, error)
```
SampleKurtosis computes the bias-corrected sample excess kurtosis,
matching pandas .kurt() and scipy.stats.kurtosis with bias=False.



## <a name="SampleSkewness">func</a> [SampleSkewness](/skewness.go?s=1049:1104#L44)
``` go
func SampleSkewness(input Float64Data) (float64, error)
```
SampleSkewness computes the adjusted Fisher-Pearson standardized moment
coefficient, correcting for bias in small samples.



## <a name="SampleVariance">func</a> [SampleVariance](/variance.go?s=1058:1122#L42)
``` go
func SampleVariance(input Float64Data) (svar float64, err error)
```
SampleVariance finds the amount of variance within a sample



## <a name="Sigmoid">func</a> [Sigmoid](/sigmoid.go?s=228:278#L9)
``` go
func Sigmoid(input Float64Data) ([]float64, error)
```
Sigmoid returns the input values in the range of -1 to 1
along the sigmoid or s-shaped curve, commonly used in
machine learning while training neural networks as an
activation function.



## <a name="Skewness">func</a> [Skewness](/skewness.go?s=90:139#L6)
``` go
func Skewness(input Float64Data) (float64, error)
```
Skewness computes the population skewness of the dataset



## <a name="SoftMax">func</a> [SoftMax](/softmax.go?s=206:256#L8)
``` go
func SoftMax(input Float64Data) ([]float64, error)
```
SoftMax returns the input values in the range of 0 to 1
with sum of all the probabilities being equal to one. It
is commonly used in machine learning neural networks.



## <a name="Spearman">func</a> [Spearman](/correlation.go?s=1006:1062#L41)
``` go
func Spearman(data1, data2 Float64Data) (float64, error)
```
Spearman calculates the Spearman rank correlation coefficient between two variables.
It works by ranking the data and then computing the Pearson correlation of the ranks.
This method handles tied values using fractional (average) ranking.



## <a name="StableSample">func</a> [StableSample](/sample.go?s=974:1042#L50)
``` go
func StableSample(input Float64Data, takenum int) ([]float64, error)
```
StableSample like stable sort, it returns samples from input while keeps the order of original data.



## <a name="StandardDeviation">func</a> [StandardDeviation](/deviation.go?s=695:762#L27)
``` go
func StandardDeviation(input Float64Data) (sdev float64, err error)
```
StandardDeviation the amount of variation in the dataset



## <a name="StandardDeviationPopulation">func</a> [StandardDeviationPopulation](/deviation.go?s=892:969#L32)
``` go
func StandardDeviationPopulation(input Float64Data) (sdev float64, err error)
```
StandardDeviationPopulation finds the amount of variation from the population



## <a name="StandardDeviationSample">func</a> [StandardDeviationSample](/deviation.go?s=1250:1323#L46)
``` go
func StandardDeviationSample(input Float64Data) (sdev float64, err error)
```
StandardDeviationSample finds the amount of variation from a sample



## <a name="StdDevP">func</a> [StdDevP](/legacy.go?s=339:396#L14)
``` go
func StdDevP(input Float64Data) (sdev float64, err error)
```
StdDevP is a shortcut to StandardDeviationPopulation



## <a name="StdDevS">func</a> [StdDevS](/legacy.go?s=497:554#L19)
``` go
func StdDevS(input Float64Data) (sdev float64, err error)
```
StdDevS is a shortcut to StandardDeviationSample



## <a name="Sum">func</a> [Sum](/sum.go?s=78:130#L6)
``` go
func Sum(input Float64Data) (sum float64, err error)
```
Sum adds all the numbers of a slice together



## <a name="TTest">func</a> [TTest](/ttest.go?s=505:604#L16)
``` go
func TTest(data1, data2 Float64Data, populationMean float64) (t float64, pvalue float64, err error)
```
TTest performs a one-sample or two-sample (independent) Student's t-test.

For a one-sample t-test, pass the sample data as data1, nil for data2,
and the expected population mean as populationMean.

For a two-sample independent t-test (assuming equal variance), pass both
sample datasets. The populationMean parameter is ignored in this case.

Returns the t statistic and the two-tailed p-value.

<a href="https://en.wikipedia.org/wiki/Student%27s_t-test">https://en.wikipedia.org/wiki/Student%27s_t-test</a>



## <a name="Trimean">func</a> [Trimean](/quartile.go?s=1320:1368#L65)
``` go
func Trimean(input Float64Data) (float64, error)
```
Trimean finds the average of the median and the midhinge



## <a name="TrimmedMean">func</a> [TrimmedMean](/trimmed_mean.go?s=450:519#L13)
``` go
func TrimmedMean(input Float64Data, percent float64) (float64, error)
```
TrimmedMean finds the mean of a slice of floats after removing a
fraction of the smallest and largest values. This matches the
behavior of Python's scipy.stats.trim_mean.

The percent parameter is the fraction removed from each tail and
must be in the range [0, 0.5). The number of elements trimmed from
each tail is floor(len(input) * percent). A percent of zero returns
the same result as Mean.



## <a name="VarGeom">func</a> [VarGeom](/geometric_distribution.go?s=1049:1097#L38)
``` go
func VarGeom(p float64) (exp float64, err error)
```
ProbGeom generates the variance for number for a
geometric random variable with parameter p



## <a name="VarP">func</a> [VarP](/legacy.go?s=59:113#L4)
``` go
func VarP(input Float64Data) (sdev float64, err error)
```
VarP is a shortcut to PopulationVariance



## <a name="VarS">func</a> [VarS](/legacy.go?s=193:247#L9)
``` go
func VarS(input Float64Data) (sdev float64, err error)
```
VarS is a shortcut to SampleVariance



## <a name="Variance">func</a> [Variance](/variance.go?s=659:717#L26)
``` go
func Variance(input Float64Data) (sdev float64, err error)
```
Variance the amount of variation in the dataset



## <a name="WeightedMean">func</a> [WeightedMean](/weighted_mean.go?s=415:476#L12)
``` go
func WeightedMean(data, weights Float64Data) (float64, error)
```
WeightedMean finds the weighted mean of a slice of floats, defined as
the sum of each data value multiplied by its weight divided by the sum
of all the weights. This matches the behavior of Python's
numpy.average with the weights argument.

The data and weights slices must be the same length. Weights must be
non-negative and at least one weight must be positive.



## <a name="Winsorize">func</a> [Winsorize](/winsorize.go?s=618:687#L16)
``` go
func Winsorize(input Float64Data, percent float64) ([]float64, error)
```
Winsorize limits the effect of outliers in a slice of floats by
clamping a fraction of the smallest and largest values. This matches
the behavior of Python's scipy.stats.mstats.winsorize with symmetric
limits.

The percent parameter is the fraction clamped in each tail and must
be in the range [0, 0.5). With k = floor(len(input) * percent),
values below the k-th smallest value are set to it and values above
the k-th largest value are set to it. The returned slice preserves
the original element order and a percent of zero returns a copy of
the input.



## <a name="ZScore">func</a> [ZScore](/zscore.go?s=205:254#L6)
``` go
func ZScore(input Float64Data) ([]float64, error)
```
ZScore standardizes the input values by subtracting the mean
and dividing by the sample standard deviation, returning the
number of standard deviations each value is from the mean.



## <a name="ZTest">func</a> [ZTest](/ztest.go?s=537:654#L17)
``` go
func ZTest(data1, data2 Float64Data, populationMean, populationStdDev float64) (z float64, pvalue float64, err error)
```
ZTest performs a one-sample or two-sample Z-test.

For a one-sample Z-test, pass the sample data as data1, nil for data2,
the known population mean as populationMean, and the known population
standard deviation as populationStdDev.

For a two-sample Z-test, pass both sample datasets and the known population
standard deviations. The populationMean parameter is ignored in this case.

Returns the Z statistic and the two-tailed p-value.

<a href="https://en.wikipedia.org/wiki/Z-test">https://en.wikipedia.org/wiki/Z-test</a>




## <a name="Coordinate">type</a> [Coordinate](/regression.go?s=143:183#L9)
``` go
type Coordinate struct {
    X, Y float64
}

```
Coordinate holds the data in a series







### <a name="ExpReg">func</a> [ExpReg](/legacy.go?s=791:856#L29)
``` go
func ExpReg(s []Coordinate) (regressions []Coordinate, err error)
```
ExpReg is a shortcut to ExponentialRegression


### <a name="LinReg">func</a> [LinReg](/legacy.go?s=643:708#L24)
``` go
func LinReg(s []Coordinate) (regressions []Coordinate, err error)
```
LinReg is a shortcut to LinearRegression


### <a name="LogReg">func</a> [LogReg](/legacy.go?s=944:1009#L34)
``` go
func LogReg(s []Coordinate) (regressions []Coordinate, err error)
```
LogReg is a shortcut to LogarithmicRegression





## <a name="Description">type</a> [Description](/describe.go?s=89:381#L6)
``` go
type Description struct {
    Count                  int
    Mean                   float64
    Std                    float64
    Max                    float64
    Min                    float64
    Range                  float64
    DescriptionPercentiles []descriptionPercentile
    AllowedNaN             bool
}

```
Holds information about the dataset provided to Describe







### <a name="Describe">func</a> [Describe](/describe.go?s=611:704#L24)
``` go
func Describe(input Float64Data, allowNaN bool, percentiles *[]float64) (*Description, error)
```
Describe generates descriptive statistics about a provided dataset, similar to python's pandas.describe()


### <a name="DescribePercentileFunc">func</a> [DescribePercentileFunc](/describe.go?s=949:1116#L30)
``` go
func DescribePercentileFunc(input Float64Data, allowNaN bool, percentiles *[]float64, percentileFunc func(Float64Data, float64) (float64, error)) (*Description, error)
```
Describe generates descriptive statistics about a provided dataset, similar to python's pandas.describe()
Takes in a function to use for percentile calculation





### <a name="Description.String">func</a> (\*Description) [String](/describe.go?s=2161:2210#L71)
``` go
func (d *Description) String(decimals int) string
```
Represents the Description instance in a string format with specified number of decimals


	count   3
	mean    2.00
	std     0.82
	max     3.00
	min     1.00
	range   2.00
	25.00%  NaN
	50.00%  1.50
	75.00%  2.50
	NaN OK  true




## <a name="Float64Data">type</a> [Float64Data](/data.go?s=80:106#L4)
``` go
type Float64Data []float64
```
Float64Data is a named type for []float64 with helper methods







### <a name="LoadRawData">func</a> [LoadRawData](/load.go?s=145:194#L12)
``` go
func LoadRawData(raw interface{}) (f Float64Data)
```
LoadRawData parses and converts a slice of mixed data types to floats





### <a name="Float64Data.ArgMax">func</a> (Float64Data) [ArgMax](/extremes.go?s=1521:1563#L66)
``` go
func (f Float64Data) ArgMax() (int, error)
```
ArgMax returns the index of the highest number in the data




### <a name="Float64Data.ArgMin">func</a> (Float64Data) [ArgMin](/extremes.go?s=1647:1689#L69)
``` go
func (f Float64Data) ArgMin() (int, error)
```
ArgMin returns the index of the lowest number in the data




### <a name="Float64Data.AutoCorrelation">func</a> (Float64Data) [AutoCorrelation](/data.go?s=3274:3337#L91)
``` go
func (f Float64Data) AutoCorrelation(lags int) (float64, error)
```
AutoCorrelation is the correlation of a signal with a delayed copy of itself as a function of delay




### <a name="Float64Data.Clip">func</a> (Float64Data) [Clip](/clip.go?s=550:612#L30)
``` go
func (f Float64Data) Clip(min, max float64) ([]float64, error)
```
Clip clamps each value in the input slice into the
inclusive range between min and max.




### <a name="Float64Data.CoefficientOfVariation">func</a> (Float64Data) [CoefficientOfVariation](/coefficient_of_variation.go?s=768:830#L29)
``` go
func (f Float64Data) CoefficientOfVariation() (float64, error)
```
CoefficientOfVariation finds the sample standard deviation divided by the mean




### <a name="Float64Data.Correlation">func</a> (Float64Data) [Correlation](/data.go?s=3075:3139#L86)
``` go
func (f Float64Data) Correlation(d Float64Data) (float64, error)
```
Correlation describes the degree of relationship between two sets of data




### <a name="Float64Data.Covariance">func</a> (Float64Data) [Covariance](/data.go?s=4996:5059#L146)
``` go
func (f Float64Data) Covariance(d Float64Data) (float64, error)
```
Covariance is a measure of how much two sets of data change




### <a name="Float64Data.CovariancePopulation">func</a> (Float64Data) [CovariancePopulation](/data.go?s=5178:5251#L151)
``` go
func (f Float64Data) CovariancePopulation(d Float64Data) (float64, error)
```
CovariancePopulation computes covariance for entire population between two variables




### <a name="Float64Data.CumulativeMax">func</a> (Float64Data) [CumulativeMax](/cumulative.go?s=1416:1471#L69)
``` go
func (f Float64Data) CumulativeMax() ([]float64, error)
```
CumulativeMax calculates the cumulative maximum of the data




### <a name="Float64Data.CumulativeMin">func</a> (Float64Data) [CumulativeMin](/cumulative.go?s=1565:1620#L74)
``` go
func (f Float64Data) CumulativeMin() ([]float64, error)
```
CumulativeMin calculates the cumulative minimum of the data




### <a name="Float64Data.CumulativeProduct">func</a> (Float64Data) [CumulativeProduct](/cumulative.go?s=1259:1318#L64)
``` go
func (f Float64Data) CumulativeProduct() ([]float64, error)
```
CumulativeProduct calculates the cumulative product of the data




### <a name="Float64Data.CumulativeSum">func</a> (Float64Data) [CumulativeSum](/data.go?s=883:938#L28)
``` go
func (f Float64Data) CumulativeSum() ([]float64, error)
```
CumulativeSum returns the cumulative sum of the data




### <a name="Float64Data.Diff">func</a> (Float64Data) [Diff](/diff.go?s=1220:1266#L45)
``` go
func (f Float64Data) Diff() ([]float64, error)
```
Diff returns the successive differences of the data




### <a name="Float64Data.EWMA">func</a> (Float64Data) [EWMA](/ewma.go?s=848:907#L30)
``` go
func (f Float64Data) EWMA(alpha float64) ([]float64, error)
```
EWMA returns the exponentially weighted moving average of the data with smoothing factor alpha




### <a name="Float64Data.Entropy">func</a> (Float64Data) [Entropy](/data.go?s=5675:5722#L167)
``` go
func (f Float64Data) Entropy() (float64, error)
```
Entropy provides calculation of the entropy




### <a name="Float64Data.GeometricMean">func</a> (Float64Data) [GeometricMean](/data.go?s=1340:1393#L40)
``` go
func (f Float64Data) GeometricMean() (float64, error)
```
GeometricMean returns the geometric mean of the data




### <a name="Float64Data.Get">func</a> (Float64Data) [Get](/data.go?s=129:168#L7)
``` go
func (f Float64Data) Get(i int) float64
```
Get item in slice




### <a name="Float64Data.HarmonicMean">func</a> (Float64Data) [HarmonicMean](/data.go?s=1477:1529#L43)
``` go
func (f Float64Data) HarmonicMean() (float64, error)
```
HarmonicMean returns the harmonic mean of the data




### <a name="Float64Data.Histogram">func</a> (Float64Data) [Histogram](/histogram.go?s=1359:1425#L55)
``` go
func (f Float64Data) Histogram(bins int) ([]int, []float64, error)
```
Histogram returns the counts and equal-width bin edges of the data




### <a name="Float64Data.InterQuartileRange">func</a> (Float64Data) [InterQuartileRange](/data.go?s=3950:4008#L111)
``` go
func (f Float64Data) InterQuartileRange() (float64, error)
```
InterQuartileRange finds the range between Q1 and Q3




### <a name="Float64Data.KendallTau">func</a> (Float64Data) [KendallTau](/kendall.go?s=1384:1447#L55)
``` go
func (f Float64Data) KendallTau(d Float64Data) (float64, error)
```
KendallTau calculates Kendall's tau-b rank correlation coefficient
between two variables.




### <a name="Float64Data.Kurtosis">func</a> (Float64Data) [Kurtosis](/kurtosis.go?s=1501:1549#L58)
``` go
func (f Float64Data) Kurtosis() (float64, error)
```
Kurtosis finds the population excess kurtosis of a slice of floats




### <a name="Float64Data.Len">func</a> (Float64Data) [Len](/data.go?s=217:247#L10)
``` go
func (f Float64Data) Len() int
```
Len returns length of slice




### <a name="Float64Data.Less">func</a> (Float64Data) [Less](/data.go?s=318:358#L13)
``` go
func (f Float64Data) Less(i, j int) bool
```
Less returns if one number is less than another




### <a name="Float64Data.Max">func</a> (Float64Data) [Max](/data.go?s=645:688#L22)
``` go
func (f Float64Data) Max() (float64, error)
```
Max returns the maximum number in the data




### <a name="Float64Data.Mean">func</a> (Float64Data) [Mean](/data.go?s=1005:1049#L31)
``` go
func (f Float64Data) Mean() (float64, error)
```
Mean returns the mean of the data




### <a name="Float64Data.Median">func</a> (Float64Data) [Median](/data.go?s=1111:1157#L34)
``` go
func (f Float64Data) Median() (float64, error)
```
Median returns the median of the data




### <a name="Float64Data.MedianAbsoluteDeviation">func</a> (Float64Data) [MedianAbsoluteDeviation](/data.go?s=1647:1710#L46)
``` go
func (f Float64Data) MedianAbsoluteDeviation() (float64, error)
```
MedianAbsoluteDeviation the median of the absolute deviations from the dataset median




### <a name="Float64Data.MedianAbsoluteDeviationPopulation">func</a> (Float64Data) [MedianAbsoluteDeviationPopulation](/data.go?s=1859:1932#L51)
``` go
func (f Float64Data) MedianAbsoluteDeviationPopulation() (float64, error)
```
MedianAbsoluteDeviationPopulation finds the median of the absolute deviations from the population median




### <a name="Float64Data.Midhinge">func</a> (Float64Data) [Midhinge](/data.go?s=4107:4168#L116)
``` go
func (f Float64Data) Midhinge(d Float64Data) (float64, error)
```
Midhinge finds the average of the first and third quartiles




### <a name="Float64Data.Min">func</a> (Float64Data) [Min](/data.go?s=536:579#L19)
``` go
func (f Float64Data) Min() (float64, error)
```
Min returns the minimum number in the data




### <a name="Float64Data.Mode">func</a> (Float64Data) [Mode](/data.go?s=1217:1263#L37)
``` go
func (f Float64Data) Mode() ([]float64, error)
```
Mode returns the mode of the data




### <a name="Float64Data.MovingAverage">func</a> (Float64Data) [MovingAverage](/rolling.go?s=1768:1833#L58)
``` go
func (f Float64Data) MovingAverage(window int) ([]float64, error)
```
MovingAverage returns the rolling mean of the data over a trailing window




### <a name="Float64Data.MovingMax">func</a> (Float64Data) [MovingMax](/moving.go?s=3475:3536#L118)
``` go
func (f Float64Data) MovingMax(window int) ([]float64, error)
```
MovingMax returns the rolling maximum of the data over a trailing window




### <a name="Float64Data.MovingMedian">func</a> (Float64Data) [MovingMedian](/moving.go?s=3125:3189#L108)
``` go
func (f Float64Data) MovingMedian(window int) ([]float64, error)
```
MovingMedian returns the rolling median of the data over a trailing window




### <a name="Float64Data.MovingMin">func</a> (Float64Data) [MovingMin](/moving.go?s=3303:3364#L113)
``` go
func (f Float64Data) MovingMin(window int) ([]float64, error)
```
MovingMin returns the rolling minimum of the data over a trailing window




### <a name="Float64Data.MovingStdDev">func</a> (Float64Data) [MovingStdDev](/rolling.go?s=1969:2033#L63)
``` go
func (f Float64Data) MovingStdDev(window int) ([]float64, error)
```
MovingStdDev returns the rolling sample standard deviation of the data over a trailing window




### <a name="Float64Data.MovingSum">func</a> (Float64Data) [MovingSum](/moving.go?s=3643:3704#L123)
``` go
func (f Float64Data) MovingSum(window int) ([]float64, error)
```
MovingSum returns the rolling sum of the data over a trailing window




### <a name="Float64Data.Pearson">func</a> (Float64Data) [Pearson](/data.go?s=3472:3532#L96)
``` go
func (f Float64Data) Pearson(d Float64Data) (float64, error)
```
Pearson calculates the Pearson product-moment correlation coefficient between two variables.




### <a name="Float64Data.PercentChange">func</a> (Float64Data) [PercentChange](/diff.go?s=1374:1429#L48)
``` go
func (f Float64Data) PercentChange() ([]float64, error)
```
PercentChange returns the fractional change between successive elements of the data




### <a name="Float64Data.Percentile">func</a> (Float64Data) [Percentile](/data.go?s=2713:2772#L76)
``` go
func (f Float64Data) Percentile(p float64) (float64, error)
```
Percentile finds the relative standing in a slice of floats




### <a name="Float64Data.PercentileNearestRank">func</a> (Float64Data) [PercentileNearestRank](/data.go?s=2886:2956#L81)
``` go
func (f Float64Data) PercentileNearestRank(p float64) (float64, error)
```
PercentileNearestRank finds the relative standing using the Nearest Rank method




### <a name="Float64Data.PercentileOfScore">func</a> (Float64Data) [PercentileOfScore](/percentile_of_score.go?s=786:856#L29)
``` go
func (f Float64Data) PercentileOfScore(score float64) (float64, error)
```
PercentileOfScore calculates the percentile rank of a score relative to the data




### <a name="Float64Data.PopulationKurtosis">func</a> (Float64Data) [PopulationKurtosis](/kurtosis.go?s=1655:1713#L63)
``` go
func (f Float64Data) PopulationKurtosis() (float64, error)
```
PopulationKurtosis finds the population excess kurtosis of a slice of floats




### <a name="Float64Data.PopulationVariance">func</a> (Float64Data) [PopulationVariance](/data.go?s=4690:4748#L136)
``` go
func (f Float64Data) PopulationVariance() (float64, error)
```
PopulationVariance finds the amount of variance within a population




### <a name="Float64Data.Product">func</a> (Float64Data) [Product](/product.go?s=544:591#L24)
``` go
func (f Float64Data) Product() (float64, error)
```
Product calculates the product of the data




### <a name="Float64Data.Quartile">func</a> (Float64Data) [Quartile](/data.go?s=3805:3868#L106)
``` go
func (f Float64Data) Quartile(d Float64Data) (Quartiles, error)
```
Quartile returns the three quartile points from a slice of data




### <a name="Float64Data.QuartileOutliers">func</a> (Float64Data) [QuartileOutliers](/data.go?s=2559:2616#L71)
``` go
func (f Float64Data) QuartileOutliers() (Outliers, error)
```
QuartileOutliers finds the mild and extreme outliers




### <a name="Float64Data.Quartiles">func</a> (Float64Data) [Quartiles](/data.go?s=5823:5874#L172)
``` go
func (f Float64Data) Quartiles() (Quartiles, error)
```
Quartiles returns the three quartile points from instance of Float64Data




### <a name="Float64Data.RMS">func</a> (Float64Data) [RMS](/rms.go?s=454:497#L21)
``` go
func (f Float64Data) RMS() (float64, error)
```
RMS calculates the root mean square of the data




### <a name="Float64Data.Range">func</a> (Float64Data) [Range](/extremes.go?s=1795:1840#L72)
``` go
func (f Float64Data) Range() (float64, error)
```
Range returns the difference between the highest and lowest numbers in the data




### <a name="Float64Data.Rank">func</a> (Float64Data) [Rank](/rank.go?s=382:428#L14)
``` go
func (f Float64Data) Rank() ([]float64, error)
```
Rank assigns fractional (average) ranks to the input values




### <a name="Float64Data.Rescale">func</a> (Float64Data) [Rescale](/rescale.go?s=603:652#L27)
``` go
func (f Float64Data) Rescale() ([]float64, error)
```
Rescale normalizes the input values to the range of 0 to 1
by subtracting the minimum and dividing by the range




### <a name="Float64Data.SEM">func</a> (Float64Data) [SEM](/sem.go?s=625:668#L22)
``` go
func (f Float64Data) SEM() (float64, error)
```
SEM calculates the standard error of the mean of the data




### <a name="Float64Data.Sample">func</a> (Float64Data) [Sample](/data.go?s=4403:4464#L126)
``` go
func (f Float64Data) Sample(n int, r bool) ([]float64, error)
```
Sample returns sample from input with replacement or without




### <a name="Float64Data.SampleKurtosis">func</a> (Float64Data) [SampleKurtosis](/kurtosis.go?s=1836:1890#L68)
``` go
func (f Float64Data) SampleKurtosis() (float64, error)
```
SampleKurtosis finds the bias-corrected sample excess kurtosis of a slice of floats




### <a name="Float64Data.SampleVariance">func</a> (Float64Data) [SampleVariance](/data.go?s=4847:4901#L141)
``` go
func (f Float64Data) SampleVariance() (float64, error)
```
SampleVariance finds the amount of variance within a sample




### <a name="Float64Data.Sigmoid">func</a> (Float64Data) [Sigmoid](/data.go?s=5364:5413#L156)
``` go
func (f Float64Data) Sigmoid() ([]float64, error)
```
Sigmoid returns the input values along the sigmoid or s-shaped curve




### <a name="Float64Data.SoftMax">func</a> (Float64Data) [SoftMax](/data.go?s=5554:5603#L162)
``` go
func (f Float64Data) SoftMax() ([]float64, error)
```
SoftMax returns the input values in the range of 0 to 1
with sum of all the probabilities being equal to one.




### <a name="Float64Data.Spearman">func</a> (Float64Data) [Spearman](/data.go?s=3648:3709#L101)
``` go
func (f Float64Data) Spearman(d Float64Data) (float64, error)
```
Spearman calculates the Spearman rank correlation coefficient between two variables.




### <a name="Float64Data.StandardDeviation">func</a> (Float64Data) [StandardDeviation](/data.go?s=2043:2100#L56)
``` go
func (f Float64Data) StandardDeviation() (float64, error)
```
StandardDeviation the amount of variation in the dataset




### <a name="Float64Data.StandardDeviationPopulation">func</a> (Float64Data) [StandardDeviationPopulation](/data.go?s=2216:2283#L61)
``` go
func (f Float64Data) StandardDeviationPopulation() (float64, error)
```
StandardDeviationPopulation finds the amount of variation from the population




### <a name="Float64Data.StandardDeviationSample">func</a> (Float64Data) [StandardDeviationSample](/data.go?s=2399:2462#L66)
``` go
func (f Float64Data) StandardDeviationSample() (float64, error)
```
StandardDeviationSample finds the amount of variation from a sample




### <a name="Float64Data.Sum">func</a> (Float64Data) [Sum](/data.go?s=764:807#L25)
``` go
func (f Float64Data) Sum() (float64, error)
```
Sum returns the total of all the numbers in the data




### <a name="Float64Data.Swap">func</a> (Float64Data) [Swap](/data.go?s=425:460#L16)
``` go
func (f Float64Data) Swap(i, j int)
```
Swap switches out two numbers in slice




### <a name="Float64Data.Trimean">func</a> (Float64Data) [Trimean](/data.go?s=4254:4314#L121)
``` go
func (f Float64Data) Trimean(d Float64Data) (float64, error)
```
Trimean finds the average of the median and the midhinge




### <a name="Float64Data.TrimmedMean">func</a> (Float64Data) [TrimmedMean](/trimmed_mean.go?s=1132:1198#L36)
``` go
func (f Float64Data) TrimmedMean(percent float64) (float64, error)
```
TrimmedMean finds the mean of the data after removing a fraction of
the smallest and largest values from each tail




### <a name="Float64Data.Variance">func</a> (Float64Data) [Variance](/data.go?s=4545:4593#L131)
``` go
func (f Float64Data) Variance() (float64, error)
```
Variance the amount of variation in the dataset




### <a name="Float64Data.WeightedMean">func</a> (Float64Data) [WeightedMean](/weighted_mean.go?s=976:1047#L40)
``` go
func (f Float64Data) WeightedMean(weights Float64Data) (float64, error)
```
WeightedMean finds the weighted mean of the data using the given weights




### <a name="Float64Data.Winsorize">func</a> (Float64Data) [Winsorize](/winsorize.go?s=1319:1385#L48)
``` go
func (f Float64Data) Winsorize(percent float64) ([]float64, error)
```
Winsorize returns a copy of the data with a fraction of the smallest
and largest values in each tail clamped




### <a name="Float64Data.ZScore">func</a> (Float64Data) [ZScore](/zscore.go?s=632:680#L27)
``` go
func (f Float64Data) ZScore() ([]float64, error)
```
ZScore standardizes the input values by subtracting the mean
and dividing by the sample standard deviation




## <a name="Outliers">type</a> [Outliers](/outlier.go?s=73:139#L4)
``` go
type Outliers struct {
    Mild    Float64Data
    Extreme Float64Data
}

```
Outliers holds mild and extreme outliers found in data







### <a name="QuartileOutliers">func</a> [QuartileOutliers](/outlier.go?s=197:255#L10)
``` go
func QuartileOutliers(input Float64Data) (Outliers, error)
```
QuartileOutliers finds the mild and extreme outliers





## <a name="Quartiles">type</a> [Quartiles](/quartile.go?s=75:136#L6)
``` go
type Quartiles struct {
    Q1 float64
    Q2 float64
    Q3 float64
}

```
Quartiles holds the three quartile points







### <a name="Quartile">func</a> [Quartile](/quartile.go?s=205:256#L13)
``` go
func Quartile(input Float64Data) (Quartiles, error)
```
Quartile returns the three quartile points from a slice of data





## <a name="Series">type</a> [Series](/regression.go?s=76:100#L6)
``` go
type Series []Coordinate
```
Series is a container for a series of data







### <a name="ExponentialRegression">func</a> [ExponentialRegression](/regression.go?s=1269:1337#L54)
``` go
func ExponentialRegression(s Series) (regressions Series, err error)
```
ExponentialRegression returns an exponential regression on data series.
A non-positive Y value returns ErrYCoord, and a series without at least two
distinct X values returns ErrBounds.


### <a name="LinearRegression">func</a> [LinearRegression](/regression.go?s=333:396#L15)
``` go
func LinearRegression(s Series) (regressions Series, err error)
```
LinearRegression finds the least squares linear regression on data series.
A series without at least two distinct X values returns ErrBounds.


### <a name="LogarithmicRegression">func</a> [LogarithmicRegression](/regression.go?s=2368:2436#L98)
``` go
func LogarithmicRegression(s Series) (regressions Series, err error)
```
LogarithmicRegression returns a logarithmic regression on data series.
A non-positive X value or a series without at least two distinct X values
returns ErrBounds.









- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
