package sheet_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"cheddarhead/assets"
	"cheddarhead/internal/sheet"
)

func TestDecodeLitPixelsAndMetadata(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	img.Set(0, 0, color.NRGBA{0, 0, 0, 255})   // black -> unlit
	img.Set(1, 0, color.NRGBA{1, 0, 0, 255})   // red -> lit
	img.Set(2, 1, color.NRGBA{0, 128, 0, 255}) // green -> lit
	img.Set(3, 2, color.NRGBA{0, 0, 255, 255}) // blue -> lit

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	raw := buf.Bytes()
	wantSum := sha256.Sum256(raw)
	wantHex := hex.EncodeToString(wantSum[:])

	fsys := fstest.MapFS{
		"assets/custom.png": {Data: raw},
	}
	s, err := sheet.Load(fsys, "assets/custom.png")
	if err != nil {
		t.Fatalf("sheet.Load: %v", err)
	}
	if s.Name() != "custom.png" {
		t.Errorf("Name() = %q, want custom.png", s.Name())
	}
	if s.SHA256() != wantHex {
		t.Errorf("SHA256() = %q, want %q", s.SHA256(), wantHex)
	}
	if s.Width() != 4 || s.Height() != 3 {
		t.Errorf("dimensions = %dx%d, want 4x3", s.Width(), s.Height())
	}

	litCoords := map[[2]int]bool{
		{1, 0}: true,
		{2, 1}: true,
		{3, 2}: true,
	}
	for y := -1; y <= 3; y++ {
		for x := -1; x <= 4; x++ {
			want := litCoords[[2]int{x, y}]
			if got := s.IsPixelLit(x, y); got != want {
				t.Errorf("IsPixelLit(%d, %d) = %v, want %v", x, y, got, want)
			}
		}
	}

	// Verify LoadFile produces identical sheet.
	tmp := filepath.Join(t.TempDir(), "custom.png")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	sf, err := sheet.LoadFile(tmp)
	if err != nil {
		t.Fatalf("sheet.LoadFile: %v", err)
	}
	if sf.Name() != "custom.png" || sf.SHA256() != wantHex {
		t.Errorf("LoadFile metadata mismatch: name=%q sha=%q", sf.Name(), sf.SHA256())
	}
	if !sf.IsPixelLit(3, 2) || sf.IsPixelLit(0, 0) {
		t.Errorf("LoadFile pixel bitmap mismatch")
	}
}

func TestLoadEmbeddedGameSheet(t *testing.T) {
	assetPath := assets.GameAssetPath(assets.FS)
	s, err := sheet.Load(assets.FS, assetPath)
	if err != nil {
		t.Fatalf("sheet.Load(%q): %v", assetPath, err)
	}
	if s.Width() <= 0 || s.Height() <= 0 {
		t.Fatalf("invalid sheet dimensions: %dx%d", s.Width(), s.Height())
	}
	if len(s.SHA256()) != 64 {
		t.Fatalf("unexpected SHA256 length %d (%q)", len(s.SHA256()), s.SHA256())
	}
}
