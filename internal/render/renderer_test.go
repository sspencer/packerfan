package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"cheddarhead/internal/config"
	"cheddarhead/internal/render"
	"cheddarhead/internal/sheet"
)

// spritesheetPath returns assets/game-real.png if it exists, otherwise the
// tracked placeholder assets/game.png, matching the game's own load order.
func spritesheetPath() string {
	real := filepath.Join("..", "..", "assets", "game-real.png")
	if _, err := os.Stat(real); err == nil {
		return real
	}
	return filepath.Join("..", "..", "assets", "game.png")
}

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
	path := spritesheetPath()
	s, err := sheet.LoadFile(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
	}

	for _, score := range []int{200, 400, 800, 1600} {
		origin, ok := render.GhostScoreSprite(score)
		if !ok {
			t.Fatalf("GhostScoreSprite(%d) not mapped", score)
		}

		litCount := 0
		for dy := range config.Sprite {
			for dx := range config.Sprite {
				if s.IsPixelLit(origin.X+dx, origin.Y+dy) {
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

	path := spritesheetPath()
	s, err := sheet.LoadFile(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
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
					if s.IsPixelLit(origin.X+dx, origin.Y+dy) {
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
	path := spritesheetPath()
	s, err := sheet.LoadFile(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
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
					if s.IsPixelLit(origin.X+dx, origin.Y+dy) {
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
					if s.IsPixelLit(origin.X+dx, origin.Y+dy) {
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

func TestFrightenedGhostFlashCount(t *testing.T) {
	tests := []struct {
		name        string
		totalFrames int
		flashes     int
	}{
		{"level 1 (6s, 5 flashes)", config.SecondsToFrames(6), 5},
		{"level 9 (1s, 3 flashes)", config.SecondsToFrames(1), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pulses := 0
			prevWhite := false
			for f := tt.totalFrames; f >= 1; f-- {
				white := render.FrightenedGhostFlashWhite(f, tt.flashes)
				if white && !prevWhite {
					pulses++
				}
				prevWhite = white
			}
			if pulses != tt.flashes {
				t.Errorf("white flash pulses = %d, want %d", pulses, tt.flashes)
			}
		})
	}
}

func TestEnergizerAndOneUpBlinking(t *testing.T) {
	// Energizers stay solid when solid=true, and alternate on/off every 10 frames otherwise.
	for f := range 30 {
		if !render.EnergizerVisible(f, true) {
			t.Fatalf("EnergizerVisible(%d, solid=true) = false, want true", f)
		}
	}
	if !render.EnergizerVisible(0, false) || render.EnergizerVisible(10, false) || !render.EnergizerVisible(20, false) {
		t.Fatalf("EnergizerVisible did not toggle every 10 frames")
	}

	// 1UP stays visible on game over, and toggles every 16 frames during play.
	if !render.OneUpVisible(16, true) {
		t.Fatalf("OneUpVisible(16, gameOver=true) = false, want true")
	}
	if !render.OneUpVisible(0, false) || render.OneUpVisible(16, false) || !render.OneUpVisible(32, false) {
		t.Fatalf("OneUpVisible did not toggle every 16 frames")
	}
}

func TestLevelBonusSymbolsWindow(t *testing.T) {
	l1 := render.LevelBonusSymbols(1)
	if len(l1) != 1 || l1[0] != config.BonusCherries {
		t.Fatalf("Level 1 symbols = %v, want [Cherries]", l1)
	}

	l2 := render.LevelBonusSymbols(2)
	if len(l2) != 2 || l2[0] != config.BonusCherries || l2[1] != config.BonusStrawberry {
		t.Fatalf("Level 2 symbols = %v, want [Cherries Strawberry]", l2)
	}

	// Level 8 slides the 7-symbol window to levels 2..8.
	l8 := render.LevelBonusSymbols(8)
	if len(l8) != 7 || l8[0] != config.BonusStrawberry || l8[6] != config.BonusGrapes {
		t.Fatalf("Level 8 symbols = %v, want len 7 starting Strawberry ending Grapes", l8)
	}

	// Level 19+ displays 7 Keys.
	l19 := render.LevelBonusSymbols(19)
	if len(l19) != 7 {
		t.Fatalf("Level 19 symbols len = %d, want 7", len(l19))
	}
	for i, s := range l19 {
		if s != config.BonusKey {
			t.Fatalf("Level 19 symbol[%d] = %v, want Key", i, s)
		}
	}
}
