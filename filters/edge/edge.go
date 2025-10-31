package edge

import (
	"image"
	"math"
)

type Edge struct {
	Magnitude float64
	Direction float64
}

type EdgeMap map[image.Point]Edge

type EdgeSolver interface {
	findEdges(img image.Image) EdgeMap
}

// Finds average magnitude and angle of edge in rect
func QuantizeBasedOnEdges(rect *image.Rectangle, edgesMap map[image.Point]Edge) (avgMag, avgDir float64) {
	bounds := rect.Bounds()

	sumSin, sumCos := 0.0, 0.0
	sumMag := 0.0
	var count int

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			point := image.Point{X: x, Y: y}
			edge, ok := edgesMap[point]
			if !ok {
				continue
			}

			sumMag += edge.Magnitude
			sumSin += math.Sin(edge.Direction)
			sumCos += math.Cos(edge.Direction)
			count++
		}
	}

	if count == 0 {
		return 0, 0 // no valid edges in this block
	}

	avgMag = sumMag / float64(count)
	avgDir = math.Atan2(sumSin, sumCos)

	return avgMag, avgDir
}

// Constructs 3x3 patch with (x,y) at center
func GetPatchForXY(img *image.RGBA, x, y int) ([9]int, map[int]image.Point) {
	var patch [9]int
	var indexToPointMap = make(map[int]image.Point)
	currIndex := 0
	for x1 := x - 1; x1 < x+2; x1++ {
		for y1 := y - 1; y1 < y+2; y1++ {
			point := image.Point{X: x1, Y: y1}
			if point.In(img.Bounds()) {
				r, _, _, _ := img.At(x1, y1).RGBA()

				intensity := float32(r >> 8)

				patch[currIndex] = int(intensity)

				indexToPointMap[currIndex] = point

			} else {
				patch[currIndex] = 0
			}

			currIndex++
		}
	}

	return patch, indexToPointMap

}
