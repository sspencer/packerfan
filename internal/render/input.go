package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"cheddarhead/internal/config"
	"cheddarhead/internal/system"
)

// EbitenInput implements system.InputProvider using Ebitengine keyboard polling.
// Call Poll once per Ebitengine Update tick so edge-triggered presses are
// latched across variable fixed-step accumulator ticks without being dropped
// or double-fired.
type EbitenInput struct {
	pending system.InputState
	keyBuf  []ebiten.Key
}

func NewEbitenInput() *EbitenInput {
	return &EbitenInput{}
}

// Poll captures edge-triggered key presses for the current Ebitengine tick.
func (e *EbitenInput) Poll() {
	altDown := ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight)
	pausePressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)
	mutePressed := inpututil.IsKeyJustPressed(ebiten.KeyM)
	confirmPressed := (!altDown && inpututil.IsKeyJustPressed(ebiten.KeyEnter)) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
	start1P := inpututil.IsKeyJustPressed(ebiten.KeyDigit1) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad1)
	start2P := inpututil.IsKeyJustPressed(ebiten.KeyDigit2) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad2)
	switchMode := inpututil.IsKeyJustPressed(ebiten.KeyDigit3) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad3)
	anyKeyPressed := pausePressed || confirmPressed || start1P || start2P || switchMode

	e.keyBuf = inpututil.AppendJustPressedKeys(e.keyBuf[:0])
	for _, key := range e.keyBuf {
		if key != ebiten.KeyM &&
			key != ebiten.KeyF11 &&
			key != ebiten.KeyEscape &&
			key != ebiten.KeyAltLeft &&
			key != ebiten.KeyAltRight &&
			key != ebiten.KeyAlt &&
			!(altDown && key == ebiten.KeyEnter) {
			anyKeyPressed = true
		}
	}

	e.pending.PausePressed = e.pending.PausePressed || pausePressed
	e.pending.MutePressed = e.pending.MutePressed || mutePressed
	e.pending.ConfirmPressed = e.pending.ConfirmPressed || confirmPressed
	e.pending.Start1PPressed = e.pending.Start1PPressed || start1P
	e.pending.Start2PPressed = e.pending.Start2PPressed || start2P
	e.pending.SwitchModePressed = e.pending.SwitchModePressed || switchMode
	e.pending.AnyKeyPressed = e.pending.AnyKeyPressed || anyKeyPressed
}

// Sample consumes latched edge-triggered presses and reads current held
// directional inputs for one simulation step.
func (e *EbitenInput) Sample() system.InputState {
	state := e.pending
	e.pending = system.InputState{}

	switch {
	case ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW):
		state.HasDirection = true
		state.Direction = config.Up
	case ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS):
		state.HasDirection = true
		state.Direction = config.Down
	case ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA):
		state.HasDirection = true
		state.Direction = config.Left
	case ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD):
		state.HasDirection = true
		state.Direction = config.Right
	}

	return state
}
