package game

import (
	"fmt"
	"image/color"
	"io/fs"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"cheddarhead/internal/component"
	"cheddarhead/internal/render"
	"cheddarhead/internal/scene"
)

var (
	overlayBg        = color.RGBA{R: 0, G: 0, B: 0, A: 200}
	overlayLightGray = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	overlayRed       = color.RGBA{R: 230, G: 41, B: 55, A: 255}
	overlayOrange    = color.RGBA{R: 255, G: 161, B: 0, A: 255}
)

// ReplayViewer renders a ReplayPlayer in an Ebitengine window using the existing
// render.Renderer.DrawWorld without initializing audio or scenes.
type ReplayViewer struct {
	assetsFS     fs.FS
	player       *ReplayPlayer
	art          *render.Art
	artSHA       string
	renderer     *render.Renderer
	status       string
	rightRepeat  float64
	overlayText  *ebiten.Image
	overlayWidth int
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

const (
	stepRepeatDelay    = 0.4  // seconds before held right-arrow starts repeating
	stepRepeatInterval = 0.05 // seconds between repeated steps
	maxStepRepeatDT    = 0.25 // per-frame dt cap for repeat stepping (at most 5 steps)
)

// Update handles Ebitengine replay controls and advances the simulation clock.
func (v *ReplayViewer) Update(dt float64) {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		v.player.TogglePause()
	}
	if v.player.Paused() {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			v.player.StepTick()
			v.rightRepeat = stepRepeatDelay
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			// Cap catch-up steps so a long stall between frames can't
			// run hundreds of ticks synchronously.
			v.rightRepeat -= min(dt, maxStepRepeatDT)
			for v.rightRepeat <= 0 {
				v.player.StepTick()
				v.rightRepeat += stepRepeatInterval
			}
		} else {
			v.rightRepeat = 0
		}
	} else {
		v.rightRepeat = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		v.player.Faster()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		v.player.Slower()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && v.player.FileCount() > 1 {
		if err := v.player.NextFile(); err != nil {
			v.status = err.Error()
		} else {
			v.status = ""
			v.syncArt()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) && v.player.FileCount() > 1 {
		if err := v.player.PrevFile(); err != nil {
			v.status = err.Error()
		} else {
			v.status = ""
			v.syncArt()
		}
	}

	v.player.Advance(dt)
}

// DrawWorld renders the sim.Env world into the 224x288 native canvas dst.
func (v *ReplayViewer) DrawWorld(dst *ebiten.Image) {
	w := v.player.Env().World()
	gs := component.MustFindGameState(w)
	v.renderer.SetTarget(dst)
	v.renderer.DrawWorld(w, scene.DrawOptions{
		ShowGameOver: v.player.Finished() && gs.GameOver,
	})
}

// DrawOverlay renders the replay status line along the bottom of screen.
func (v *ReplayViewer) DrawOverlay(screen *ebiten.Image, vp render.Viewport) {
	bounds := screen.Bounds()
	winW := bounds.Dx()
	winH := bounds.Dy()

	scale := max(1.0, math.Round(float64(vp.Scale)/3.0))
	const glyphW = 6
	const glyphH = 16
	barH := int(float64(glyphH)*scale) + 4
	vector.FillRect(screen, 0, float32(winH-barH), float32(winW), float32(barH), overlayBg, false)

	line := v.player.HUDLine()
	clr := overlayLightGray
	if err := v.player.HashError(); err != nil {
		line = err.Error()
		clr = overlayRed
	} else if v.status != "" {
		line = v.status
		clr = overlayOrange
	}

	needW := max(1, len(line)*glyphW+2)
	if v.overlayText == nil || v.overlayWidth < needW {
		if v.overlayText != nil {
			v.overlayText.Deallocate()
		}
		v.overlayWidth = needW
		v.overlayText = ebiten.NewImage(needW, glyphH)
	}
	v.overlayText.Clear()
	ebitenutil.DebugPrintAt(v.overlayText, line, 0, 0)

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(6, float64(winH-barH+2))
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(v.overlayText, &op)
}

// Unload releases GPU resources owned by the viewer.
func (v *ReplayViewer) Unload() {
	if v.art != nil {
		v.art.Unload()
		v.art = nil
	}
	if v.overlayText != nil {
		v.overlayText.Deallocate()
		v.overlayText = nil
	}
}
