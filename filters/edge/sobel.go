package edge

import (
	"go-filters/helpers"
	"image"
	"math"
)

type SobelEdgeDetector struct{}

func (d *SobelEdgeDetector) FindEdges(img image.Image) EdgeMap {
	var edgesMap = make(EdgeMap)
	var rgba = img.(*image.RGBA)
	bounds := img.Bounds()

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			patch, _ := GetPatchForXY(rgba, x, y)
			gx := [9]int{-1, 0, 1, -2, 0, 2, -1, 0, 1}
			gy := [9]int{-1, -2, -1, 0, 0, 0, 1, 2, 1}

			point := image.Point{X: x, Y: y}

			Gx := float64(helpers.DotProduct(patch, gx))
			Gy := float64(helpers.DotProduct(patch, gy))

			direction := math.Atan2(float64(Gy), float64(Gx))

			magnitude := math.Sqrt(Gx*Gx + Gy*Gy)

			edge := Edge{Magnitude: magnitude, Direction: direction}
			edgesMap[point] = edge

		}

	}

	return edgesMap
}
