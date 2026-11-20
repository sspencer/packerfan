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
	pausePressed := rl.IsKeyPressed(rl.KeySpace)
	anyKeyPressed := pausePressed
	for rl.GetKeyPressed() != 0 {
		anyKeyPressed = true
	}

	state := system.InputState{
		PausePressed:  pausePressed,
		AnyKeyPressed: anyKeyPressed,
	}

	switch {
	case rl.IsKeyDown(rl.KeyUp):
		state.HasDirection = true
		state.Direction = config.Up
	case rl.IsKeyDown(rl.KeyDown):
		state.HasDirection = true
		state.Direction = config.Down
	case rl.IsKeyDown(rl.KeyLeft):
		state.HasDirection = true
		state.Direction = config.Left
	case rl.IsKeyDown(rl.KeyRight):
		state.HasDirection = true
		state.Direction = config.Right
	}

	return state
}
