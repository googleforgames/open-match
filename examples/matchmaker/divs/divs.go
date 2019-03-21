package divs

import (
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

var (
	cdf       float64
	divisions float64
	scores    []int
)

//func main() {
//	flag.Float64Var(&divisions, "divs", 100, "number of times to divide the population")
//	flag.Parse()
//}

func GenerateBuckets(divs int64) []int {

	const mu float64 = 2266.0
	const sigma float64 = 610.0
	const min int = 0
	const max int = 4350
	var divisions float64 = float64(divs)
	var x float64 = (1.0 / divisions)

	scores = append([]int{min}, scores...)

	for i := 0; i < 4350; i++ {

		fi := float64(i)
		zscore := stat.StdScore(fi, mu, sigma)
		norm := distuv.Normal{Mu: mu, Sigma: sigma}
		prevcdf := cdf
		cdf = norm.CDF(fi)

		y := int(cdf * divisions)
		z := int(prevcdf * divisions)
		this := ""
		j := (y - z)
		if float64(j)/divisions >= x {
			scores = append(scores, i)
			this = "======="
		}
		//fmt.Printf("%v: %0.4v %0.4v %v %v %v %v %v\n", i, zscore, cdf, y, z, j, x, this)
		_, _, _, _, _, _, _, _ = i, zscore, cdf, y, z, j, x, this
	}
	scores = append(scores, max)
	//fmt.Println(scores, len(scores))

	return scores
}
