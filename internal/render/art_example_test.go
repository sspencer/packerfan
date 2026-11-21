package render_test

import (
	"testing"
	"testing/fstest"

	"pacfan/internal/render"
)

func TestArtPrefersRealArtwork(t *testing.T) {
	base := fstest.MapFS{
		"assets/game-real.png": {Data: []byte("real game")},
		"assets/text-real.png": {Data: []byte("real text")},
		"assets/game.png":      {Data: []byte("example game")},
		"assets/text.png":      {Data: []byte("example text")},
	}
	if got := render.GameAssetPath(base); got != "assets/game-real.png" {
		t.Errorf("GameAssetPath = %q, want assets/game-real.png", got)
	}
	if got := render.FontAssetPath(base); got != "assets/text-real.png" {
		t.Errorf("FontAssetPath = %q, want assets/text-real.png", got)
	}
	if err := render.CheckAssets(base); err != nil {
		t.Errorf("CheckAssets: %v", err)
	}
}

func TestArtFallsBackToExampleArtwork(t *testing.T) {
	base := fstest.MapFS{
		"assets/game.png":        {Data: []byte("example game")},
		"assets/text.png":        {Data: []byte("example text")},
		"assets/text-real.png/x": {Data: []byte("x")}, // directory, not a file
	}
	if got := render.GameAssetPath(base); got != "assets/game.png" {
		t.Errorf("GameAssetPath = %q, want assets/game.png", got)
	}
	if got := render.FontAssetPath(base); got != "assets/text.png" {
		t.Errorf("FontAssetPath = %q, want assets/text.png", got)
	}
	if err := render.CheckAssets(base); err != nil {
		t.Errorf("CheckAssets: %v", err)
	}
}

func TestArtMixesRealAndExampleArtwork(t *testing.T) {
	base := fstest.MapFS{
		"assets/game-real.png": {Data: []byte("real game")},
		"assets/text.png":      {Data: []byte("example text")},
	}
	if got := render.GameAssetPath(base); got != "assets/game-real.png" {
		t.Errorf("GameAssetPath = %q, want assets/game-real.png", got)
	}
	if got := render.FontAssetPath(base); got != "assets/text.png" {
		t.Errorf("FontAssetPath = %q, want assets/text.png", got)
	}
	if err := render.CheckAssets(base); err != nil {
		t.Errorf("CheckAssets: %v", err)
	}
}
