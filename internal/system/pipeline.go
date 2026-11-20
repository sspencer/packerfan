package system

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/event"
)

// Pipeline orchestrates the ordered execution of ECS systems and event flushes
// for a single simulation frame.
type Pipeline struct {
	input     *InputSystem
	gameState *GameStateSystem
	collision *CollisionSystem
	ghosts    *GhostSystem
	fruit     *FruitSystem
	player    *PlayerSystem

	cache worldCache
}

func NewPipeline(inputProvider InputProvider) *Pipeline {
	return &Pipeline{
		input:     NewInputSystem(inputProvider),
		gameState: NewGameStateSystem(),
		collision: NewCollisionSystem(),
		ghosts:    NewGhostSystem(),
		fruit:     NewFruitSystem(),
		player:    NewPlayerSystem(),
	}
}

// Input returns the pipeline's InputSystem so scenes can share a single input
// sampler without double-polling edge-triggered keys.
func (p *Pipeline) Input() *InputSystem {
	return p.input
}

// UpdateEatenGhosts advances previously eaten ghost eyes during the score freeze.
func (p *Pipeline) UpdateEatenGhosts(w donburi.World) {
	p.ghosts.UpdateEatenGhosts(w)
}

func (p *Pipeline) flushEvents(w donburi.World) {
	if p.cache.bus != nil {
		if p.cache.bus.HasPending() {
			p.cache.bus.ProcessAll(w)
		}
		return
	}
	if w.Len() != p.cache.entityLen {
		p.cache.bus = event.FindBus(w)
		p.cache.entityLen = w.Len()
		if p.cache.bus != nil && p.cache.bus.HasPending() {
			p.cache.bus.ProcessAll(w)
		}
	}
}

// Update advances the ECS world by one fixed tick.
func (p *Pipeline) Update(w donburi.World) {
	p.cache.ensure(w)
	c := &p.cache

	p.input.Sample()
	if p.input.current.PausePressed && c.gs != nil {
		c.gs.Paused = !c.gs.Paused
	}

	gameState := c.gs
	if gameState.Paused || gameState.LevelComplete || gameState.PlayerDying || gameState.GameOver {
		return
	}

	if gameState.GhostEatenPause > 0 {
		p.input.applyPlayerInputCached(c)
		p.ghosts.updateEatenGhostsCached(c)
		gameState.GhostEatenPause--
		if gameState.GhostEatenPause > 0 {
			return
		}
		gameState.GhostEatenEntity = donburi.Null
		gameState.GhostEatenScore = 0
	}

	p.gameState.updateCached(w, c)
	p.collision.updateCached(w, c)
	p.flushEvents(w)
	// Flush follow-up events published by collision handlers (e.g., ExtraLifeAwarded on GhostEaten/FruitEaten).
	p.flushEvents(w)

	if gameState.GhostEatenPause > 0 || gameState.LevelComplete || gameState.PlayerDying {
		return
	}

	p.ghosts.updateCached(c)
	p.fruit.updateCached(w, c)
	p.flushEvents(w)

	p.input.applyPlayerInputCached(c)
	p.player.updateCached(w, c)
	p.flushEvents(w)
	// Flush follow-up events published by dot/powerup handlers (e.g., LevelCompleted).
	p.flushEvents(w)
}
