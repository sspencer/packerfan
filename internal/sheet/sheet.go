package sheet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// Sheet holds a pure-Go lit-pixel bitmap decoded once from a game spritesheet PNG,
// together with its base filename and content SHA-256 digest.
type Sheet struct {
	name      string
	sha256Hex string
	minX      int
	minY      int
	width     int
	height    int
	lit       []byte // bit-packed row-major bitmap: 1 if R>0 || G>0 || B>0
}

// Load reads and decodes the PNG at assetPath from fsys.
func Load(fsys fs.FS, assetPath string) (*Sheet, error) {
	if fsys == nil {
		return nil, fmt.Errorf("sheet: nil filesystem for %q", assetPath)
	}
	data, err := fs.ReadFile(fsys, assetPath)
	if err != nil {
		return nil, fmt.Errorf("sheet: read %q: %w", assetPath, err)
	}
	return Decode(path.Base(assetPath), data)
}

// LoadFile reads and decodes a PNG file from disk.
func LoadFile(filePath string) (*Sheet, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("sheet: read %q: %w", filePath, err)
	}
	return Decode(filepath.Base(filePath), data)
}

// Decode decodes PNG bytes into a Sheet with the given base filename.
func Decode(name string, data []byte) (*Sheet, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("sheet: decode %q: %w", name, err)
	}
	return FromImage(name, data, img), nil
}

// FromImage builds a Sheet from an already-decoded image. data must be the
// PNG bytes img was decoded from; they are hashed for the sheet's SHA-256.
func FromImage(name string, data []byte, img image.Image) *Sheet {
	sum := sha256.Sum256(data)
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	lit := make([]byte, (w*h+7)/8)

	for y := range h {
		rowOff := y * w
		py := bounds.Min.Y + y
		for x := range w {
			r, g, b, _ := img.At(bounds.Min.X+x, py).RGBA()
			if r > 0 || g > 0 || b > 0 {
				idx := rowOff + x
				lit[idx>>3] |= 1 << (idx & 7)
			}
		}
	}

	return &Sheet{
		name:      name,
		sha256Hex: hex.EncodeToString(sum[:]),
		minX:      bounds.Min.X,
		minY:      bounds.Min.Y,
		width:     w,
		height:    h,
		lit:       lit,
	}
}

// IsPixelLit reports whether pixel (x, y) has any non-zero R, G, or B channel.
// Out-of-bounds coordinates return false.
func (s *Sheet) IsPixelLit(x, y int) bool {
	if s == nil {
		return false
	}
	rx := x - s.minX
	ry := y - s.minY
	if rx < 0 || rx >= s.width || ry < 0 || ry >= s.height {
		return false
	}
	idx := ry*s.width + rx
	return (s.lit[idx>>3] & (1 << (idx & 7))) != 0
}

// Name returns the base filename of the PNG (e.g. "game.png").
func (s *Sheet) Name() string {
	return s.name
}

// SHA256 returns the lowercase hex-encoded SHA-256 digest of the PNG bytes.
func (s *Sheet) SHA256() string {
	return s.sha256Hex
}

// Width returns the image width in pixels.
func (s *Sheet) Width() int {
	return s.width
}

// Height returns the image height in pixels.
func (s *Sheet) Height() int {
	return s.height
}
