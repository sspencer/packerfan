package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

// InputState captures one frame of user input decoupled from any specific backend.
type InputState struct {
	PausePressed   bool
	ConfirmPressed bool
	Start1PPressed bool
	Start2PPressed bool
	AnyKeyPressed  bool
	Direction      config.Direction
	HasDirection   bool
}

// InputProvider samples hardware or simulated controls for the current frame.
type InputProvider interface {
	Sample() InputState
}

// InputSystem translates sampled input into pause toggles and PlayerInput components.
type InputSystem struct {
	provider    InputProvider
	current     InputState
	playerQuery *donburi.Query
}

func NewInputSystem(provider InputProvider) *InputSystem {
	return &InputSystem{
		provider: provider,
		playerQuery: donburi.NewQuery(
			filter.Contains(component.PlayerTag, component.PlayerInput),
		),
	}
}

// Sample reads the current frame's input from the configured provider.
func (s *InputSystem) Sample() {
	if s.provider == nil {
		s.current = InputState{}
		return
	}
	s.current = s.provider.Sample()
}

// AnyKeyPressed reports whether any key was pressed on the most recently sampled frame.
func (s *InputSystem) AnyKeyPressed() bool {
	return s.current.AnyKeyPressed
}

// PausePressed reports whether Space (pause/start) was pressed on the most recently sampled frame.
func (s *InputSystem) PausePressed() bool {
	return s.current.PausePressed
}

// ConfirmPressed reports whether Enter was pressed on the most recently sampled frame.
func (s *InputSystem) ConfirmPressed() bool {
	return s.current.ConfirmPressed
}

// Start1PPressed reports whether the "1" key was pressed on the most recently sampled frame.
func (s *InputSystem) Start1PPressed() bool {
	return s.current.Start1PPressed
}

// Start2PPressed reports whether the "2" key was pressed on the most recently sampled frame.
func (s *InputSystem) Start2PPressed() bool {
	return s.current.Start2PPressed
}

// ApplyPause toggles the world's pause flag when pause was pressed.
func (s *InputSystem) ApplyPause(w donburi.World) {
	if !s.current.PausePressed {
		return
	}
	gameState := component.MustFindGameState(w)
	gameState.Paused = !gameState.Paused
}

// ApplyPlayerInput writes directional input onto every player entity.
func (s *InputSystem) ApplyPlayerInput(w donburi.World) {
	for entry := range s.playerQuery.Iter(w) {
		input := component.PlayerInput.Get(entry)
		input.Active = s.current.HasDirection
		if s.current.HasDirection {
			input.Requested = s.current.Direction
		}
	}
}
