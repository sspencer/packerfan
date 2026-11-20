package render

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestCheckAssets(t *testing.T) {
	all := fstest.MapFS{
		"assets/text.png":      {Data: []byte("x")},
		"assets/game.png":      {Data: []byte("x")},
		"assets/chroma_key.fs": {Data: []byte("x")},
	}
	if err := CheckAssets(all); err != nil {
		t.Fatalf("CheckAssets with all files = %v, want nil", err)
	}

	partial := fstest.MapFS{
		"assets/chroma_key.fs": {Data: []byte("x")},
		"assets/game.png/x":    {Data: []byte("x")}, // directory, not a file
	}
	err := CheckAssets(partial)
	if err == nil {
		t.Fatal("CheckAssets with missing files = nil, want error")
	}
	for _, want := range []string{"assets/text.png", "assets/game.png"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
	if strings.Contains(err.Error(), "chroma_key.fs") {
		t.Errorf("error %q mentions present file chroma_key.fs", err)
	}
}
