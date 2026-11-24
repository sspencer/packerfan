package render

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"cheddarhead/internal/config"
)

// DrawText renders uppercase arcade bitmap font text at tile coordinates (x, y).
func DrawText(dst, font *ebiten.Image, text string, x, y int, clr color.Color) {
	drawTextWithCamera(dst, font, camera2D{zoom: 1}, text, x, y, clr)
}

func drawTextWithCamera(dst, font *ebiten.Image, cam camera2D, text string, x, y int, clr color.Color) {
	str := []byte(strings.ToUpper(text))
	var fx, fy int

	var op ebiten.DrawImageOptions
	op.ColorScale.ScaleWithColor(clr)

	for i, c := range str {
		ch := int(c)
		switch {
		case ch >= 48 && ch <= 57:
			// 0 - 9
			fx = ch - 48
			fy = 2
		case ch >= 65 && ch <= 79:
			// A - O
			fx = ch - 65
			fy = 0
		case ch >= 80 && ch <= 90:
			// P - Z
			fx = ch - 80
			fy = 1
		case ch == 33:
			// "!"
			fx = 11
			fy = 1
		case ch == 47:
			// "/"
			fx = 10
			fy = 2
		case ch == 45:
			// "-"
			fx = 11
			fy = 2
		case ch == 34:
			// '"'
			fx = 12
			fy = 2
		default:
			continue
		}

		sx := fx * config.Tile
		sy := fy * config.Tile
		sub := font.SubImage(image.Rect(sx, sy, sx+config.Tile, sy+config.Tile)).(*ebiten.Image)
		op.GeoM = cam.geoM(float64((x+i)*config.Tile), float64(y*config.Tile))
		dst.DrawImage(sub, &op)
	}
}
