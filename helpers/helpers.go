package helpers

import (
	"image"
	"image/draw"
)

func ClampUINT8(val int) int {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}

	return val
}

func MultiplyMatrices(m1, m2 [9]int) [9]int {

	var res [9]int
	for i := range 9 {
		curRow := i / 3
		curCol := i % 3

		var m1Vals [3]int
		var m2Vals [3]int

		for k := range 3 {
			m1Vals[k] = m1[curRow*3+k]
		}

		for k := range 3 {
			m2Vals[k] = m2[k*3+curCol]
		}

		sum := 0
		for j := range 3 {
			sum += m1Vals[j] * m2Vals[j]
		}

		res[i] = sum
	}
	return res
}

func DotProduct(m1, m2 [9]int) int {
	sum := 0
	for i := range 9 {
		sum += m1[i] * m2[i]
	}
	return sum
}

func CopyImage(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}
