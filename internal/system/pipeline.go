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

// Input returns the pipeline's InputSystem so scenes can share a single input
// sampler without double-polling edge-triggered keys.
func (p *Pipeline) Input() *InputSystem {
	return p.input
}

// Update advances the ECS world by one fixed tick.
func (p *Pipeline) Update(w donburi.World) {
	p.input.Sample()
	p.input.ApplyPause(w)

	gameState := component.MustFindGameState(w)
	if gameState.Paused || gameState.LevelComplete || gameState.PlayerDying || gameState.GameOver {
		return
	}

	if gameState.GhostEatenPause > 0 {
		p.input.ApplyPlayerInput(w)
		gameState.GhostEatenPause--
		if gameState.GhostEatenPause > 0 {
			return
		}
		gameState.GhostEatenEntity = donburi.Null
		gameState.GhostEatenScore = 0
	}

	p.gameState.Update(w)
	p.collision.Update(w)
	events.ProcessAllEvents(w)

	if gameState.GhostEatenPause > 0 || gameState.LevelComplete || gameState.PlayerDying {
		return
	}

	p.ghosts.Update(w)
	events.ProcessAllEvents(w)

	p.input.ApplyPlayerInput(w)
	p.player.Update(w)
	events.ProcessAllEvents(w)
	// Flush follow-up events published by dot/powerup handlers (e.g., LevelCompleted).
	events.ProcessAllEvents(w)
}
