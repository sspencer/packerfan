package render_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"pacfan/internal/render"
)

func TestExampleArtFSRedirectsArtwork(t *testing.T) {
	base := fstest.MapFS{
		"assets/game.png":         {Data: []byte("real game")},
		"assets/text.png":         {Data: []byte("real text")},
		"assets/game-example.png": {Data: []byte("example game")},
		"assets/text-example.png": {Data: []byte("example text")},
		"assets/chroma_key.fs":    {Data: []byte("shader")},
	}
	ex := render.ExampleArtFS(base)

	for path, want := range map[string]string{
		"assets/game.png":      "example game",
		"assets/text.png":      "example text",
		"assets/chroma_key.fs": "shader",
	} {
		got, err := fs.ReadFile(ex, path)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", path, err)
		}
		if string(got) != want {
			t.Errorf("ReadFile(%q) = %q, want %q", path, got, want)
		}
	}
	if err := render.CheckAssets(ex); err != nil {
		t.Errorf("CheckAssets: %v", err)
	}
}

func TestExampleArtFSReportsMissingExamples(t *testing.T) {
	base := fstest.MapFS{
		"assets/game.png":      {Data: []byte("real game")},
		"assets/text.png":      {Data: []byte("real text")},
		"assets/chroma_key.fs": {Data: []byte("shader")},
	}
	if err := render.CheckAssets(render.ExampleArtFS(base)); err == nil {
		t.Error("CheckAssets succeeded without example files, want error")
	}
}
