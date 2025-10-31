package filters

import (
	_ "fmt"
	"go-filters/filters/edge"
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

type AsciiFilter struct {
}

func (f *AsciiFilter) Filter(img image.Image, current_frame int) {
	bounds := img.Bounds()

	edgeDetector := edge.SobelEdgeDetector{}

	copyImg := helpers.CopyImage(img)

	grayScaler := GrayscaleFilter{}
	grayScaler.Filter(copyImg, current_frame)

	rgba := img.(*image.RGBA)

	edgesMap := edgeDetector.FindEdges(img)

	for x := bounds.Min.X; x < bounds.Max.X; x += RECT_SIZE {
		for y := bounds.Min.Y; y < bounds.Max.Y; y += RECT_SIZE {
			minPoint := image.Point{X: x, Y: y}
			maxPoint := image.Point{X: x + RECT_SIZE, Y: y + RECT_SIZE}

			rect := image.Rectangle{Min: minPoint, Max: maxPoint}

			avgMag, avgDir := edge.QuantizeBasedOnEdges(&rect, edgesMap)

			if avgMag < 70 {
				luminance, clr := quantizeRect(rgba, &rect)
				char := fonts.PickCharOnLuminance(luminance)
				fonts.RenderChar(char, rgba, rect, clr)
			} else {
				_, clr := quantizeRect(rgba, &rect)
				char := fonts.PickCharOnAngle(avgDir)
				fonts.RenderChar(char, rgba, rect, clr)
			}
		}
	}

}

type GaussianBlur struct{}

func (f *GaussianBlur) Filter(img image.Image, current_frame int) {
	bounds := img.Bounds()
	rgba := img.(*image.RGBA)

	kernel := GenerateGaussianKernel9(2.0)

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			intensities, _ := edge.GetPatchForXY(rgba, x, y)

			var sum float64
			for i := 0; i < 9; i++ {
				sum += float64(intensities[i]) * kernel[i]
			}

			if sum > 255 {
				sum = 255
			}

			conv := uint8(sum)
			newColor := color.RGBA{R: conv, G: conv, B: conv, A: 255}
			rgba.Set(x, y, newColor)
		}
	}
}

// GenerateGaussianKernel1D creates a 1D Gaussian kernel of given size and sigma.
// The kernel is centered, normalized (sums to 1), and suitable for separable convolution.
func GenerateGaussianKernel9(sigma float64) [9]float64 {
	const size = 9
	const radius = size / 2
	var kernel [size]float64
	var sum float64

	for i := -radius; i <= radius; i++ {
		exponent := -(float64(i * i)) / (2 * sigma * sigma)
		value := math.Exp(exponent) / (math.Sqrt(2*math.Pi) * sigma)
		kernel[i+radius] = value
		sum += value
	}

	// Normalize so that total sum = 1
	for i := 0; i < size; i++ {
		kernel[i] /= sum
	}

	return kernel
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
