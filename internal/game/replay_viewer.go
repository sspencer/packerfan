package game

import (
	"fmt"
	"io/fs"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"

	"pacfan/internal/component"
	"pacfan/internal/render"
	"pacfan/internal/scene"
)

// ReplayViewer renders a ReplayPlayer in a Raylib window using the existing
// render.Renderer.DrawWorld without initializing audio or scenes.
type ReplayViewer struct {
	assetsFS fs.FS
	player   *ReplayPlayer
	art      *render.Art
	artSHA   string
	renderer *render.Renderer
	status   string
}

// NewReplayViewer constructs a GPU-backed viewer around an already-validated ReplayPlayer.
func NewReplayViewer(assetsFS fs.FS, player *ReplayPlayer) *ReplayViewer {
	v := &ReplayViewer{
		assetsFS: assetsFS,
		player:   player,
	}
	v.syncArt()
	return v
}

func (v *ReplayViewer) syncArt() {
	artInfo := v.player.Art()
	if artInfo == nil {
		return
	}
	if v.art != nil && v.artSHA == artInfo.Sheet.SHA256() {
		return
	}
	if v.art != nil {
		v.art.Unload()
	}
	v.art = render.NewArtWithGamePNG(v.assetsFS, artInfo.Bytes)
	v.artSHA = artInfo.Sheet.SHA256()
	v.renderer = render.NewRenderer(v.art)
	if artInfo.Warning != "" {
		fmt.Fprintf(os.Stderr, "warning: %s\n", artInfo.Warning)
	}
}

// Update handles Raylib replay controls and advances the simulation clock.
func (v *ReplayViewer) Update(dt float64) {
	if rl.IsKeyPressed(rl.KeySpace) {
		v.player.TogglePause()
	}
	if v.player.Paused() && (rl.IsKeyPressed(rl.KeyRight) || rl.IsKeyPressedRepeat(rl.KeyRight)) {
		v.player.StepTick()
	}
	if rl.IsKeyPressed(rl.KeyEqual) || rl.IsKeyPressed(rl.KeyKpAdd) {
		v.player.Faster()
	}
	if rl.IsKeyPressed(rl.KeyMinus) || rl.IsKeyPressed(rl.KeyKpSubtract) {
		v.player.Slower()
	}
	if rl.IsKeyPressed(rl.KeyN) && v.player.FileCount() > 1 {
		if err := v.player.NextFile(); err != nil {
			v.status = err.Error()
		} else {
			v.status = ""
			v.syncArt()
		}
	}
	if rl.IsKeyPressed(rl.KeyP) && v.player.FileCount() > 1 {
		if err := v.player.PrevFile(); err != nil {
			v.status = err.Error()
		} else {
			v.status = ""
			v.syncArt()
		}
	}

	v.player.Advance(dt)
}

// DrawWorld renders the sim.Env world into the active 224x288 native canvas.
func (v *ReplayViewer) DrawWorld() {
	w := v.player.Env().World()
	gs := component.MustFindGameState(w)
	v.renderer.DrawWorld(w, scene.DrawOptions{
		ShowGameOver: v.player.Finished() && gs.GameOver,
	})
}

// DrawOverlay renders the replay status line along the bottom of the screen.
func (v *ReplayViewer) DrawOverlay(vp render.Viewport) {
	winW := int32(rl.GetScreenWidth())
	winH := int32(rl.GetScreenHeight())
	fontSize := max(int32(10), int32(4*vp.Scale))
	barH := fontSize + 6
	rl.DrawRectangle(0, winH-barH, winW, barH, rl.Color{R: 0, G: 0, B: 0, A: 200})

	line := v.player.HUDLine()
	color := rl.LightGray
	if err := v.player.HashError(); err != nil {
		line = err.Error()
		color = rl.Red
	} else if v.status != "" {
		line = v.status
		color = rl.Orange
	}
	rl.DrawText(line, 6, winH-barH+3, fontSize, color)
}

// Unload releases GPU resources owned by the viewer.
func (v *ReplayViewer) Unload() {
	if v.art != nil {
		v.art.Unload()
		v.art = nil
	}
}
