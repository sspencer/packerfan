package system

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/event"
)

// GameStateSystem advances global round counters, ghost Scatter/Chase wave
// scheduling, frightened mode expiration, and level-completion checks.
type GameStateSystem struct{}

func NewGameStateSystem() *GameStateSystem {
	return &GameStateSystem{}
}

func (s *GameStateSystem) Update(w donburi.World) {
	gameState := component.MustFindGameState(w)
	gameState.FrameCount++

	if gameState.DotsRemaining == 0 {
		event.LevelCompleted.Publish(w, event.LevelCompletedEvent{})
	}

	nextGhostState := config.GetGhostWaveState(gameState.Level, gameState.FrameCount)
	if nextGhostState != gameState.GhostState {
		event.GhostStateChanged.Publish(w, event.GhostStateChangedEvent{
			State: nextGhostState,
		})
	}

	if gameState.IsFrightMode {
		gameState.FrightFrames--
		if gameState.FrightFrames <= 0 {
			event.FrightModeEnded.Publish(w, event.FrightModeEndedEvent{})
		}
	}
}
