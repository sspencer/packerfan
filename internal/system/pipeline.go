package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/features/events"

	"pacfan/internal/component"
)

// Pipeline orchestrates the ordered execution of ECS systems and event flushes
// for a single simulation frame.
type Pipeline struct {
	input     *InputSystem
	gameState *GameStateSystem
	collision *CollisionSystem
	ghosts    *GhostSystem
	player    *PlayerSystem
}

func NewPipeline(inputProvider InputProvider) *Pipeline {
	return &Pipeline{
		input:     NewInputSystem(inputProvider),
		gameState: NewGameStateSystem(),
		collision: NewCollisionSystem(),
		ghosts:    NewGhostSystem(),
		player:    NewPlayerSystem(),
	}
}

// Update advances the ECS world by one fixed tick.
func (p *Pipeline) Update(w donburi.World) {
	p.input.Sample()
	p.input.ApplyPause(w)

	if component.MustFindGameState(w).Paused {
		return
	}

	p.gameState.Update(w)
	p.collision.Update(w)
	events.ProcessAllEvents(w)

	p.ghosts.Update(w)
	events.ProcessAllEvents(w)

	p.input.ApplyPlayerInput(w)
	p.player.Update(w)
	events.ProcessAllEvents(w)
}
