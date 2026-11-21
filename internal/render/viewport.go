package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"pacfan/internal/config"
)

const (
	// NativeWidth and NativeHeight are the logical pixel dimensions of the
	// unscaled arcade display (28x36 8px tiles = 224x288).
	NativeWidth  = config.ScreenWidth * config.Tile
	NativeHeight = config.ScreenHeight * config.Tile
)

// Viewport describes the destination rectangle on the host window/screen where
// the native 224x288 arcade canvas is drawn, preserving aspect ratio and
// centered with black margins.
type Viewport struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
	Scale  float32
}

// FitViewport computes the largest aspect-ratio-preserving centered rectangle
// for the native 224x288 arcade canvas inside a screenW x screenH display.
// If maxZoom > 0, the scale factor is capped at maxZoom while still shrinking
// to fit if the display is smaller.
func FitViewport(screenW, screenH int, maxZoom float32) Viewport {
	if screenW <= 0 || screenH <= 0 {
		return Viewport{
			Width:  NativeWidth,
			Height: NativeHeight,
			Scale:  1,
		}
	}

	scaleX := float32(screenW) / float32(NativeWidth)
	scaleY := float32(screenH) / float32(NativeHeight)
	scale := min(scaleX, scaleY)
	if maxZoom > 0 && scale > maxZoom {
		scale = maxZoom
	}

	width := float32(NativeWidth) * scale
	height := float32(NativeHeight) * scale
	x := (float32(screenW) - width) / 2
	y := (float32(screenH) - height) / 2

	return Viewport{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Scale:  scale,
	}
}

// Canvas renders the native 224x288 arcade frame into an offscreen image with
// nearest-neighbor filtering and presents it scaled and centered on screen.
type Canvas struct {
	target  *ebiten.Image
	maxZoom float32
}

// NewCanvas allocates a 224x288 offscreen image. When maxZoom <= 0, Present
// scales the canvas to the maximum size that fits the display; when maxZoom > 0,
// Present caps the scale at maxZoom.
func NewCanvas(maxZoom float32) *Canvas {
	return &Canvas{
		target:  ebiten.NewImage(NativeWidth, NativeHeight),
		maxZoom: maxZoom,
	}
}

// Image returns the underlying 224x288 offscreen render target.
func (c *Canvas) Image() *ebiten.Image {
	return c.target
}

// Clear fills the native 224x288 canvas with solid black.
func (c *Canvas) Clear() {
	c.target.Fill(color.Black)
}

// Present draws the native canvas onto screen, zoomed to fit and centered with
// nearest-neighbor scaling, returning the screen-space Viewport used.
func (c *Canvas) Present(screen *ebiten.Image) Viewport {
	bounds := screen.Bounds()
	vp := FitViewport(bounds.Dx(), bounds.Dy(), c.maxZoom)

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(vp.Scale), float64(vp.Scale))
	op.GeoM.Translate(float64(vp.X), float64(vp.Y))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(c.target, &op)
	return vp
}

// Unload releases the offscreen GPU image.
func (c *Canvas) Unload() {
	if c.target != nil {
		c.target.Deallocate()
	}
}
