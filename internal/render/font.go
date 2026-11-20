package render

import (
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"pacfan/internal/config"
)

// DrawText renders uppercase arcade bitmap font text at tile coordinates (x, y).
func DrawText(font rl.Texture2D, text string, x, y int, color rl.Color) {
	str := []byte(strings.ToUpper(text))
	var fx, fy int

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

		src := rl.NewRectangle(float32(fx*config.Tile), float32(fy*config.Tile), config.Tile, config.Tile)
		dst := rl.NewRectangle(float32((x+i)*config.Tile), float32(y*config.Tile), config.Tile, config.Tile)
		rl.DrawTexturePro(font, src, dst, rl.Vector2{}, 0, color)
	}
}
