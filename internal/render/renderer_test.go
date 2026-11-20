package render_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"pacfan/internal/config"
	"pacfan/internal/render"
)

func TestGhostScoreSpriteMapping(t *testing.T) {
	tests := []struct {
		score  int
		want   config.Vector
		wantOK bool
	}{
		{200, config.Vector{X: 456, Y: 128}, true},
		{400, config.Vector{X: 472, Y: 128}, true},
		{800, config.Vector{X: 488, Y: 128}, true},
		{1600, config.Vector{X: 504, Y: 128}, true},
		{0, config.Vector{}, false},
		{100, config.Vector{}, false},
		{3200, config.Vector{}, false},
	}

	for _, tt := range tests {
		got, ok := render.GhostScoreSprite(tt.score)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("GhostScoreSprite(%d) = (%v, %v), want (%v, %v)", tt.score, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestGhostScoreSpritesPresentInSpritesheetIfPresent(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "game.png")
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode spritesheet: %v", err)
	}

	for _, score := range []int{200, 400, 800, 1600} {
		origin, ok := render.GhostScoreSprite(score)
		if !ok {
			t.Fatalf("GhostScoreSprite(%d) not mapped", score)
		}

		litCount := 0
		for dy := range config.Sprite {
			for dx := range config.Sprite {
				r, g, b, _ := img.At(origin.X+dx, origin.Y+dy).RGBA()
				if r > 0 || g > 0 || b > 0 {
					litCount++
				}
			}
		}
		if litCount == 0 {
			t.Errorf("GhostScoreSprite(%d) at %v has no lit pixels in assets/game.png", score, origin)
		}
	}
}

func TestPlayerDeathSpritesPresentInSpritesheetIfPresent(t *testing.T) {
	if _, ok := render.PlayerDeathSprite(true, -1); ok {
		t.Fatalf("expected ok=false for negative death frame")
	}
	if _, ok := render.PlayerDeathSprite(true, config.PlayerDyingFrameCount); ok {
		t.Fatalf("expected ok=false for out-of-range death frame")
	}

	path := filepath.Join("..", "..", "assets", "game.png")
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode spritesheet: %v", err)
	}

	for _, pacmanMode := range []bool{true, false} {
		for frame := range config.PlayerDyingFrameCount {
			origin, ok := render.PlayerDeathSprite(pacmanMode, frame)
			if !ok {
				t.Fatalf("PlayerDeathSprite(pacman=%v, frame=%d) returned ok=false", pacmanMode, frame)
			}
			litCount := 0
			for dy := range config.Sprite {
				for dx := range config.Sprite {
					r, g, b, _ := img.At(origin.X+dx, origin.Y+dy).RGBA()
					if r > 0 || g > 0 || b > 0 {
						litCount++
					}
				}
			}
			if litCount == 0 {
				t.Errorf("PlayerDeathSprite(pacman=%v, frame=%d) at %v has no lit pixels", pacmanMode, frame, origin)
			}
		}
	}
}

func TestBonusFruitAndScoreSpritesInSpritesheetIfPresent(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "game.png")
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode spritesheet: %v", err)
	}

	symbols := []config.BonusSymbol{
		config.BonusCherries, config.BonusStrawberry, config.BonusPeach, config.BonusApple,
		config.BonusGrapes, config.BonusGalaxian, config.BonusBell, config.BonusKey,
	}
	for _, pacmanMode := range []bool{true, false} {
		for _, sym := range symbols {
			origin, ok := render.BonusFruitSprite(pacmanMode, sym)
			if !ok {
				t.Fatalf("BonusFruitSprite(pacman=%v, %v) returned ok=false", pacmanMode, sym)
			}
			lit := 0
			for dy := range config.Sprite {
				for dx := range config.Sprite {
					r, g, b, _ := img.At(origin.X+dx, origin.Y+dy).RGBA()
					if r > 0 || g > 0 || b > 0 {
						lit++
					}
				}
			}
			if lit == 0 {
				t.Errorf("BonusFruitSprite(pacman=%v, %v) at %v has no lit pixels", pacmanMode, sym, origin)
			}
		}
		for _, pts := range []int{100, 300, 500, 700, 1000, 2000, 3000, 5000} {
			origin, w, ok := render.BonusScoreSprite(pacmanMode, pts)
			if !ok || w <= 0 {
				t.Fatalf("BonusScoreSprite(pacman=%v, %d) returned ok=%v w=%d", pacmanMode, pts, ok, w)
			}
			lit := 0
			for dy := range config.Sprite {
				for dx := range w {
					r, g, b, _ := img.At(origin.X+dx, origin.Y+dy).RGBA()
					if r > 0 || g > 0 || b > 0 {
						lit++
					}
				}
			}
			if lit == 0 {
				t.Errorf("BonusScoreSprite(pacman=%v, %d) at %v (w=%d) has no lit pixels", pacmanMode, pts, origin, w)
			}
		}
	}
}
