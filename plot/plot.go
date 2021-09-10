package plot

import (
	"fmt"
	"math/rand"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func test01() {
	points := plotter.XYs{}
	points2 := plotter.XYs{}

	var a, b float64 = 0.7, 3
	for i := 0; i <= 20; i++ {
		points = append(points, plotter.XY{
			X: float64(i),
			Y: a*float64(i) + b,
		})

		points2 = append(points2, plotter.XY{
			X: float64(i),
			Y: a*float64(i) + b + (2*rand.Float64()-1),
		})
	}

	plt := plot.New()
	plt.Y.Min, plt.X.Min, plt.Y.Max, plt.X.Max = 0, 0, 22, 22 

	if err := plotutil.AddLinePoints(plt,
		"line1", points,
		"line2", points2,
	); err != nil {
		panic(err)
	}

	// if err := plotutil.AddScatters(plt,
	// 	"line1", points,
	// 	"line2", points2,
	// ); err != nil {
	// 	panic(err)
	// }

	if err := plt.Save(5*vg.Inch, 5*vg.Inch, "01-draw-line.png"); err != nil {
		panic(err)
	}
}

func test02(){
	fmt.Printf("git.....")
}

func Run(){
	test01()
}