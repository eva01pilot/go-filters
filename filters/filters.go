package filters

import (
	_ "fmt"
	"go-filters/fonts"
	"go-filters/helpers"
	"image"
	"image/color"
	"math"
)

const RECT_SIZE = 8.0

type Filter interface {
	Filter(img image.Image)
}

type DynamicFilter interface {
	Filter(img image.Image, frame_index int)
}

type ChannelShiftFilter struct {
	dr uint8
	dg uint8
	db uint8
}

func (f *ChannelShiftFilter) Configure(dr, dg, db uint8) {
	f.dr = dr
	f.dg = dg
	f.db = db
}

func (f *ChannelShiftFilter) Filter(img image.Image, frame_count int) {
	rgba := img.(*image.RGBA)

	for y := rgba.Bounds().Min.Y; y < rgba.Bounds().Max.Y; y++ {
		rowStart := (y - rgba.Bounds().Min.Y) * rgba.Stride
		for x := rgba.Bounds().Min.X; x < rgba.Bounds().Max.X; x++ {
			idx := rowStart + (x-rgba.Bounds().Min.X)*4

			r := int(rgba.Pix[idx+0] + f.dr)
			g := int(rgba.Pix[idx+1] + f.dg)
			b := int(rgba.Pix[idx+2] + f.db)

			rgba.Pix[idx+0] = uint8(helpers.ClampUINT8(r))
			rgba.Pix[idx+1] = uint8(helpers.ClampUINT8(g))
			rgba.Pix[idx+2] = uint8(helpers.ClampUINT8(b))
		}
	}

}

type WaveFilter struct{}

func (f *WaveFilter) Filter(img image.Image, frame_index int) {
	const amplitude = 10.0
	spatialFreq := 0.05 // 2π every ~125 pixels
	temporalFreq := 0.2 // phase shift per frame

	rgba := img.(*image.RGBA)
	nrgba := image.NewRGBA(rgba.Rect)

	copy(nrgba.Pix, rgba.Pix)

	height := rgba.Bounds().Dy()

	offsetY := func(x, y int) {
		offset := int(math.Sin(float64(x)*spatialFreq+float64(frame_index)*temporalFreq) * amplitude)
		srcY := y - offset
		if srcY >= 0 && srcY < height {
			srcIdx := srcY*rgba.Stride + x*4
			dstIdx := y*nrgba.Stride + x*4
			copy(nrgba.Pix[dstIdx:dstIdx+4], rgba.Pix[srcIdx:srcIdx+4])
		}
	}

	for x := rgba.Bounds().Min.X; x < rgba.Bounds().Max.X; x++ {
		for y := rgba.Bounds().Min.Y; y < rgba.Bounds().Max.Y; y++ {
			offsetY(x, y)
		}

	}

	copy(rgba.Pix, nrgba.Pix)

}

type AsciiFilter struct {
}

func (f *AsciiFilter) Filter(img image.Image, current_frame int) {
	rgba := img.(*image.RGBA)
	bounds := rgba.Bounds()

	for x := bounds.Min.X; x < bounds.Max.X; x += RECT_SIZE {
		for y := bounds.Min.Y; y < bounds.Max.Y; y += RECT_SIZE {
			rect := image.Rect(x, y, x+RECT_SIZE, y+RECT_SIZE)
			luminance, clr := quantizeRect(rgba, &rect)

			char := fonts.PickCharOnLuminance(luminance)
			fonts.RenderChar(char, img, rect, clr)
		}
	}

}

type GrayscaleFilter struct{}

func (f *GrayscaleFilter) Filter(img image.Image, current_frame int) {
	bounds := img.Bounds()
	rgba := img.(*image.RGBA)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			a8 := uint8(a >> 8)

			var gray = uint8((0.2126*float32(r8) + 0.7152*float32(g8) + 0.0722*float32(b8)))
			grayColor := color.RGBA{R: gray, G: gray, B: gray, A: a8}
			rgba.Set(x, y, grayColor)
		}
	}

}

type Edge struct {
	Magnitude float64
	Direction float64
}

type SobelFilter struct{}

func (f *SobelFilter) Filter(img image.Image, current_frame int) {
	bounds := img.Bounds()

	copyImg := helpers.CopyImage(img)

	grayScaler := GrayscaleFilter{}
	grayScaler.Filter(copyImg, current_frame)

	rgba := img.(*image.RGBA)

	var edgesMap = make(map[image.Point]Edge)

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			patch, _ := getPatchForXY(copyImg, x, y)
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

	for x := bounds.Min.X; x < bounds.Max.X; x += RECT_SIZE {
		for y := bounds.Min.Y; y < bounds.Max.Y; y += RECT_SIZE {
			minPoint := image.Point{X: x, Y: y}
			maxPoint := image.Point{X: x + RECT_SIZE, Y: y + RECT_SIZE}

			rect := image.Rectangle{Min: minPoint, Max: maxPoint}

			avgMag, avgDir := quantizeBasedOnEdges(&rect, edgesMap)

			if avgMag < 80 {
				luminance, clr := quantizeRect(rgba, &rect)
				char := fonts.PickCharOnLuminance(luminance)
				fonts.RenderChar(char, rgba, rect, clr)
			} else {
				_, _ = quantizeRect(rgba, &rect)
				char := fonts.PickCharOnAngle(avgDir)
				fonts.RenderChar(char, rgba, rect, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}
	}

}

func getPatchForXY(img *image.RGBA, x, y int) ([9]int, map[int]image.Point) {
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

func quantizeBasedOnEdges(rect *image.Rectangle, edgesMap map[image.Point]Edge) (avgMag, avgDir float64) {
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

			if edge.Magnitude < 80 {
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

func quantizeRect(img *image.RGBA, rect *image.Rectangle) (float32, color.RGBA) {
	bounds := rect.Bounds()
	var sum_r, sum_g, sum_b int
	var pixCount int
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			sum_r += int(r8)
			sum_g += int(g8)
			sum_b += int(b8)
			pixCount++
		}
	}

	var avg_r = sum_r / pixCount
	var avg_g = sum_g / pixCount
	var avg_b = sum_b / pixCount

	var luminance = (0.2126*float32(avg_r) + 0.7152*float32(avg_g) + 0.0722*float32(avg_b)) / 255

	clr := color.RGBA{R: uint8(avg_r), G: uint8(avg_g), B: uint8(avg_b), A: uint8(255)}

	return luminance, clr
}
