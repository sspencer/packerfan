package assets_test

import (
	"testing"
	"testing/fstest"

	"pacfan/assets"
)

func TestEmbeddedFS(t *testing.T) {
	if err := fstest.TestFS(assets.FS, "assets/game.png", "assets/text.png", "assets/chroma_key.fs"); err != nil {
		t.Fatalf("fstest.TestFS failed: %v", err)
	}
}

func TestGameAssetPath(t *testing.T) {
	withReal := fstest.MapFS{
		"assets/game-real.png": {Data: []byte("real")},
		"assets/game.png":      {Data: []byte("example")},
	}
	if got := assets.GameAssetPath(withReal); got != "assets/game-real.png" {
		t.Errorf("GameAssetPath(withReal) = %q, want assets/game-real.png", got)
	}

	withoutReal := fstest.MapFS{
		"assets/game.png": {Data: []byte("example")},
	}
	if got := assets.GameAssetPath(withoutReal); got != "assets/game.png" {
		t.Errorf("GameAssetPath(withoutReal) = %q, want assets/game.png", got)
	}

	dirAsReal := fstest.MapFS{
		"assets/game-real.png/x": {Data: []byte("dir")},
		"assets/game.png":        {Data: []byte("example")},
	}
	if got := assets.GameAssetPath(dirAsReal); got != "assets/game.png" {
		t.Errorf("GameAssetPath(dirAsReal) = %q, want assets/game.png", got)
	}
}
