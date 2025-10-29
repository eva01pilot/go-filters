package fonts

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var asciiSpriteCache = make(map[int]map[rune]image.Image)

var USED_CHARACTERS = []rune("@&%$#WM8B0QOZodc+;:,. ")
var EDGE_CHARACTERS = [5]rune{'/', '-', '\\', '|', '_'}

func RenderChar(char rune, dstImg image.Image, dstRect image.Rectangle, clr color.Color) {
	bounds := dstRect.Bounds()
	dstRGBA := dstImg.(*image.RGBA)

	charSprite := asciiSpriteCache[bounds.Dx()][char]
	ColorSprite(clr, charSprite)
	srcPoint := image.Point{0, 0}
	draw.Draw(dstRGBA, dstRect, charSprite, srcPoint, draw.Src)
}

func ColorSprite(clr color.Color, sprite image.Image) {
	bounds := sprite.Bounds()
	dst := sprite.(*image.RGBA)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := clr.RGBA()
			_, _, _, a := sprite.At(x, y).RGBA()
			newColor := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
			dst.Set(x, y, newColor)
		}
	}

}

func PickCharOnLuminance(luminance float32) rune {
	index := int((1 - luminance) * float32(len(USED_CHARACTERS)-1))
	return USED_CHARACTERS[index]
}

func PickCharOnAngle(angle float64) rune {
	angleDeg := angle * 180 / math.Pi

	switch {
	case angleDeg >= -22.5 && angleDeg < 22.5:
		return '|'
	case angleDeg >= 22.5 && angleDeg < 67.5:
		return '/'
	case angleDeg >= 67.5 && angleDeg < 112.5:
		return '-'
	case angleDeg >= 112.5 && angleDeg < 157.5:
		return '\\'
	default:
		return '|'
	}
}

func CreateASCIISprites(rect_size int) {
	min := image.Point{X: 0, Y: 0}
	max := image.Point{X: rect_size, Y: rect_size}
	rect := image.Rectangle{Min: min, Max: max}
	for _, char := range USED_CHARACTERS {
		img := image.NewRGBA(rect)
		d := font.Drawer{Dst: img, Src: image.White, Face: basicfont.Face7x13, Dot: fixed.P(0, rect.Dx()-2)}

		d.DrawString(string(char))

		if asciiSpriteCache[rect.Dx()] == nil {
			asciiSpriteCache[rect.Dx()] = make(map[rune]image.Image)
		}

		asciiSpriteCache[rect.Dx()][char] = img
	}
	for _, char := range EDGE_CHARACTERS {
		img := image.NewRGBA(rect)
		d := font.Drawer{Dst: img, Src: image.White, Face: basicfont.Face7x13, Dot: fixed.P(0, rect.Dx()-2)}

		d.DrawString(string(char))

		if asciiSpriteCache[rect.Dx()] == nil {
			asciiSpriteCache[rect.Dx()] = make(map[rune]image.Image)
		}

		asciiSpriteCache[rect.Dx()][char] = img
	}
}
