package system

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
	"cheddarhead/internal/event"
)

// GameStateSystem advances global round counters, ghost Scatter/Chase wave
// scheduling, frightened mode expiration, and level-completion checks.
type GameStateSystem struct{}

func NewGameStateSystem() *GameStateSystem {
	return &GameStateSystem{}
}

func (s *GameStateSystem) Update(w donburi.World) {
	s.updateGameState(w, component.MustFindGameState(w), nil)
}

func (s *GameStateSystem) updateCached(w donburi.World, c *worldCache) {
	s.updateGameState(w, c.gs, c.bus)
}

func (s *GameStateSystem) updateGameState(w donburi.World, gameState *component.GameStateData, bus *event.Bus) {
	gameState.FrameCount++

	if gameState.DotsRemaining == 0 && !gameState.LevelComplete {
		if bus != nil {
			event.LevelCompleted.PublishTo(bus, event.LevelCompletedEvent{})
		} else {
			event.LevelCompleted.Publish(w, event.LevelCompletedEvent{})
		}
	}

	if gameState.FruitActive {
		gameState.FruitTimer--
		if gameState.FruitTimer <= 0 {
			gameState.FruitActive = false
			gameState.FruitMoving = false
			gameState.FruitExiting = false
			gameState.FruitTimer = 0
		}
	}
	if gameState.FruitScoreTimer > 0 {
		gameState.FruitScoreTimer--
		if gameState.FruitScoreTimer <= 0 {
			gameState.FruitScoreTimer = 0
			gameState.FruitScoreValue = 0
		}
	}

	if gameState.IsFrightMode {
		gameState.FrightFrames--
		if gameState.FrightFrames <= 0 {
			if bus != nil {
				event.FrightModeEnded.PublishTo(bus, event.FrightModeEndedEvent{})
			} else {
				event.FrightModeEnded.Publish(w, event.FrightModeEndedEvent{})
			}
		}
		return
	}

	gameState.WaveFrames++
	nextGhostState := config.GetGhostWaveState(gameState.Level, gameState.WaveFrames)
	if nextGhostState != gameState.GhostState {
		ev := event.GhostStateChangedEvent{State: nextGhostState}
		if bus != nil {
			event.GhostStateChanged.PublishTo(bus, ev)
		} else {
			event.GhostStateChanged.Publish(w, ev)
		}
	}
}
