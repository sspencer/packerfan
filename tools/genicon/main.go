// Command genicon renders the 1024x1024 application icon used for the macOS
// app bundle (see "make app").
//
//	go run ./tools/genicon packaging/macos/icon.png
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const (
	size    = 1024
	samples = 4 // supersampling per axis for anti-aliasing

	// macOS icon grid: an 824x824 rounded-square body centred on the canvas.
	bodyMin    = 100.0
	bodyMax    = 924.0
	bodyRadius = 185.0

	wallInset  = 56.0
	wallWidth  = 26.0
	wallRadius = bodyRadius - wallInset

	playerX, playerY = 430.0, 512.0
	playerR          = 215.0
	mouthHalfAngle   = 34 * math.Pi / 180

	dotR = 24.0
)

var (
	colBackground = color.NRGBA{0x0a, 0x0a, 0x28, 0xff}
	colWall       = color.NRGBA{0x21, 0x21, 0xde, 0xff}
	colPlayer     = color.NRGBA{0xff, 0xe0, 0x1a, 0xff}
	colDot        = color.NRGBA{0xff, 0xb8, 0xae, 0xff}
	dotXs         = []float64{700, 800}
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genicon OUTPUT.png")
		os.Exit(2)
	}
	if err := writePNG(os.Args[1], render()); err != nil {
		fmt.Fprintln(os.Stderr, "genicon:", err)
		os.Exit(1)
	}
}

func render() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const n = samples * samples
	for y := range size {
		for x := range size {
			var r, g, b, a float64
			for sy := range samples {
				for sx := range samples {
					px := float64(x) + (float64(sx)+0.5)/samples
					py := float64(y) + (float64(sy)+0.5)/samples
					c, ok := shade(px, py)
					if !ok {
						continue
					}
					r += float64(c.R)
					g += float64(c.G)
					b += float64(c.B)
					a++
				}
			}
			if a == 0 {
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / a),
				G: uint8(g / a),
				B: uint8(b / a),
				A: uint8(255 * a / n),
			})
		}
	}
	return img
}

// shade returns the colour at (x, y), or false outside the icon body.
func shade(x, y float64) (color.NRGBA, bool) {
	if roundedRectDist(x, y, bodyMin, bodyMax, bodyRadius) > 0 {
		return color.NRGBA{}, false
	}
	if inPlayer(x, y) {
		return colPlayer, true
	}
	for _, dx := range dotXs {
		if math.Hypot(x-dx, y-playerY) <= dotR {
			return colDot, true
		}
	}
	d := roundedRectDist(x, y, bodyMin+wallInset, bodyMax-wallInset, wallRadius)
	if math.Abs(d) <= wallWidth/2 {
		return colWall, true
	}
	return colBackground, true
}

func inPlayer(x, y float64) bool {
	dx, dy := x-playerX, y-playerY
	if math.Hypot(dx, dy) > playerR {
		return false
	}
	return math.Abs(math.Atan2(dy, dx)) > mouthHalfAngle
}

// roundedRectDist is the signed distance from (x, y) to a square rounded
// rectangle spanning [lo, hi] on both axes (negative inside).
func roundedRectDist(x, y, lo, hi, radius float64) float64 {
	c := (lo + hi) / 2
	half := (hi-lo)/2 - radius
	qx := math.Abs(x-c) - half
	qy := math.Abs(y-c) - half
	outside := math.Hypot(math.Max(qx, 0), math.Max(qy, 0))
	inside := math.Min(math.Max(qx, qy), 0)
	return outside + inside - radius
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
