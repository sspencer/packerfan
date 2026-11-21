package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"pacfan/internal/config"
	"pacfan/internal/system"
)

// RaylibInput implements system.InputProvider using Raylib keyboard polling.
type RaylibInput struct{}

func NewRaylibInput() *RaylibInput {
	return &RaylibInput{}
}

func (r *RaylibInput) Sample() system.InputState {
	altDown := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)
	pausePressed := rl.IsKeyPressed(rl.KeySpace)
	mutePressed := rl.IsKeyPressed(rl.KeyM)
	confirmPressed := (!altDown && rl.IsKeyPressed(rl.KeyEnter)) || rl.IsKeyPressed(rl.KeyKpEnter)
	start1P := rl.IsKeyPressed(rl.KeyOne) || rl.IsKeyPressed(rl.KeyKp1)
	start2P := rl.IsKeyPressed(rl.KeyTwo) || rl.IsKeyPressed(rl.KeyKp2)
	anyKeyPressed := pausePressed || confirmPressed || start1P || start2P
	for key := rl.GetKeyPressed(); key != 0; key = rl.GetKeyPressed() {
		if key != rl.KeyM && key != rl.KeyF11 && !(altDown && key == rl.KeyEnter) {
			anyKeyPressed = true
		}
	}

	state := system.InputState{
		PausePressed:   pausePressed,
		MutePressed:    mutePressed,
		ConfirmPressed: confirmPressed,
		Start1PPressed: start1P,
		Start2PPressed: start2P,
		AnyKeyPressed:  anyKeyPressed,
	}

	switch {
	case rl.IsKeyDown(rl.KeyUp) || rl.IsKeyDown(rl.KeyW):
		state.HasDirection = true
		state.Direction = config.Up
	case rl.IsKeyDown(rl.KeyDown) || rl.IsKeyDown(rl.KeyS):
		state.HasDirection = true
		state.Direction = config.Down
	case rl.IsKeyDown(rl.KeyLeft) || rl.IsKeyDown(rl.KeyA):
		state.HasDirection = true
		state.Direction = config.Left
	case rl.IsKeyDown(rl.KeyRight) || rl.IsKeyDown(rl.KeyD):
		state.HasDirection = true
		state.Direction = config.Right
	}

	return state
}
