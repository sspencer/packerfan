package sim

import (
	"fmt"

	"cheddarhead/internal/config"
	"cheddarhead/internal/system"
)

// Action represents a discrete directional control matching config.Direction
// (0=right, 1=left, 2=up, 3=down) plus 4=noop.
type Action uint8

const (
	ActionRight Action = Action(config.Right)
	ActionLeft  Action = Action(config.Left)
	ActionUp    Action = Action(config.Up)
	ActionDown  Action = Action(config.Down)
	ActionNoop  Action = 4
)

// ActionNames lists the canonical wire names indexed by Action value.
var ActionNames = [5]string{"right", "left", "up", "down", "noop"}

// Valid reports whether a is one of the 5 defined actions.
func (a Action) Valid() bool {
	return a <= ActionNoop
}

// String returns the canonical lowercase wire name for the action.
func (a Action) String() string {
	if int(a) < len(ActionNames) {
		return ActionNames[a]
	}
	return fmt.Sprintf("action(%d)", a)
}

// ParseAction parses a wire name ("right", "left", "up", "down", "noop").
func ParseAction(s string) (Action, bool) {
	switch s {
	case "right":
		return ActionRight, true
	case "left":
		return ActionLeft, true
	case "up":
		return ActionUp, true
	case "down":
		return ActionDown, true
	case "noop":
		return ActionNoop, true
	default:
		return ActionNoop, false
	}
}

// ScriptedInput implements system.InputProvider for headless simulation.
// It only ever sets directional input and never triggers pause, mute, or start keys.
type ScriptedInput struct {
	action Action
}

// NewScriptedInput creates a ScriptedInput initialized to ActionNoop.
func NewScriptedInput() *ScriptedInput {
	return &ScriptedInput{action: ActionNoop}
}

// Set updates the action that will be returned on subsequent Sample calls.
func (s *ScriptedInput) Set(a Action) {
	s.action = a
}

// Action returns the currently latched action.
func (s *ScriptedInput) Action() Action {
	return s.action
}

// Sample returns the InputState corresponding to the currently latched action.
func (s *ScriptedInput) Sample() system.InputState {
	if s.action <= ActionDown {
		return system.InputState{
			Direction:    config.Direction(s.action),
			HasDirection: true,
		}
	}
	return system.InputState{}
}
