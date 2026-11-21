package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCheckAssets(t *testing.T) {
	all := fstest.MapFS{
		"assets/text.png": {Data: []byte("x")},
		"assets/game.png": {Data: []byte("x")},
	}
	if err := CheckAssets(all); err != nil {
		t.Fatalf("CheckAssets with all files = %v, want nil", err)
	}

	partial := fstest.MapFS{
		"assets/text.png":   {Data: []byte("x")},
		"assets/game.png/x": {Data: []byte("x")}, // directory, not a file
	}
	err := CheckAssets(partial)
	if err == nil {
		t.Fatal("CheckAssets with missing files = nil, want error")
	}
	if !strings.Contains(err.Error(), "assets/game.png") {
		t.Errorf("error %q does not mention assets/game.png", err)
	}
	if strings.Contains(err.Error(), "assets/text.png") {
		t.Errorf("error %q mentions present file assets/text.png", err)
	}
}

func TestFlashShaderCompiles(t *testing.T) {
	shader, err := ebiten.NewShader([]byte(flashFragmentShader))
	if err != nil {
		t.Fatalf("flashFragmentShader failed to compile: %v", err)
	}
	shader.Deallocate()
}

func TestChromaKeyNRGBA(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{R: 12, G: 10, B: 5, A: 255})
	src.SetNRGBA(2, 0, color.NRGBA{R: 255, G: 255, B: 0, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}

	keyed := chromaKeyPNG(buf.Bytes())
	if got := keyed.NRGBAAt(0, 0); got.A != 0 {
		t.Errorf("pure black pixel alpha = %d, want 0", got.A)
	}
	if got := keyed.NRGBAAt(1, 0); got.A != 0 {
		t.Errorf("near-black pixel alpha = %d, want 0", got.A)
	}
	if got := keyed.NRGBAAt(2, 0); got != (color.NRGBA{R: 255, G: 255, B: 0, A: 255}) {
		t.Errorf("bright yellow pixel = %+v, want opaque yellow", got)
	}
}
