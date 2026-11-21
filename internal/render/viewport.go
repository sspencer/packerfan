package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

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

// Canvas renders the native 224x288 arcade frame into an offscreen texture with
// nearest-neighbor filtering and presents it scaled and centered on screen.
type Canvas struct {
	target  rl.RenderTexture2D
	maxZoom float32
}

// NewCanvas allocates a 224x288 render target. When maxZoom <= 0, Present
// scales the canvas to the maximum size that fits the display; when maxZoom > 0,
// Present caps the scale at maxZoom.
func NewCanvas(maxZoom float32) *Canvas {
	target := rl.LoadRenderTexture(NativeWidth, NativeHeight)
	rl.SetTextureFilter(target.Texture, rl.FilterPoint)
	return &Canvas{
		target:  target,
		maxZoom: maxZoom,
	}
}

// Begin redirects subsequent 2D drawing into the 224x288 native canvas cleared to black.
func (c *Canvas) Begin() {
	rl.BeginTextureMode(c.target)
	rl.ClearBackground(rl.Black)
}

// End finishes drawing into the native canvas.
func (c *Canvas) End() {
	rl.EndTextureMode()
}

// Present draws the native canvas onto the current screen, zoomed to fit and
// centered, returning the screen-space Viewport used.
func (c *Canvas) Present() Viewport {
	vp := FitViewport(rl.GetScreenWidth(), rl.GetScreenHeight(), c.maxZoom)
	// RenderTexture2D is stored upside down in OpenGL, so flip source height.
	src := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(c.target.Texture.Width),
		Height: -float32(c.target.Texture.Height),
	}
	dst := rl.Rectangle{
		X:      vp.X,
		Y:      vp.Y,
		Width:  vp.Width,
		Height: vp.Height,
	}
	rl.DrawTexturePro(c.target.Texture, src, dst, rl.Vector2{}, 0, rl.White)
	return vp
}

// Unload releases the offscreen GPU render texture.
func (c *Canvas) Unload() {
	rl.UnloadRenderTexture(c.target)
}
