package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
)

// ResetActors restores the player and all ghosts to their initial spawn state.
func ResetActors(w donburi.World) {
	playerQuery := donburi.NewQuery(
		filter.Contains(component.PlayerTag, component.Spawn, component.Transform),
	)
	for entry := range playerQuery.Iter(w) {
		component.ResetPlayer(entry)
	}

	ghostQuery := donburi.NewQuery(
		filter.Contains(component.GhostTag, component.Ghost, component.Spawn, component.Transform),
	)
	for entry := range ghostQuery.Iter(w) {
		component.ResetGhost(entry)
	}
}

// ResetGhostDotCounters zeroes every ghost's personal house dot counter (used on new level/game).
func ResetGhostDotCounters(w donburi.World) {
	ghostQuery := donburi.NewQuery(
		filter.Contains(component.GhostTag, component.Ghost),
	)
	for entry := range ghostQuery.Iter(w) {
		component.Ghost.Get(entry).DotCounter = 0
	}
}

// AdvanceToNextLevel increments the level (clamped to 255), loads the next
// level's maze when isPixelLit is provided, rewinds the PRNG, resets round
// state, and restores all actors to their spawn defaults.
func AdvanceToNextLevel(w donburi.World, isPixelLit func(x, y int) bool) int {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	nextLevel := min(gameState.Level+1, 255)
	dotsRemaining := 0
	if isPixelLit != nil {
		dotsRemaining = maze.LoadLevel(nextLevel, isPixelLit)
	} else {
		maze.Level = nextLevel
	}

	gameState.ResetForNextLevel(nextLevel, dotsRemaining)
	component.MustFindPRNG(w).Reset()
	ResetActors(w)
	ResetGhostDotCounters(w)
	return nextLevel
}

// ConsumeLifeAndRespawn decrements Lives after the player is eaten. If lives
// remain, it resets round timers, PRNG, and actor positions on the current
// level (keeping dots and score intact) and returns true. Otherwise it marks
// GameOver and returns false.
func ConsumeLifeAndRespawn(w donburi.World) bool {
	gameState := component.MustFindGameState(w)
	if gameState.Lives > 0 {
		gameState.Lives--
	}
	gameState.PlayerDying = false
	if gameState.Lives <= 0 {
		gameState.Lives = 0
		gameState.GameOver = true
		return false
	}

	gameState.ResetForNextLife()
	component.MustFindPRNG(w).Reset()
	ResetActors(w)
	return true
}

// RestartGame resets the world to Level 1 with full lives, zero score (keeping
// HighScore), a freshly loaded Level 1 maze, rewound PRNG, and spawned actors.
func RestartGame(w donburi.World, isPixelLit func(x, y int) bool) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	dotsRemaining := 0
	if isPixelLit != nil {
		dotsRemaining = maze.LoadLevel(1, isPixelLit)
	} else {
		maze.Level = 1
	}

	gameState.ResetForNewGame(dotsRemaining)
	component.MustFindPRNG(w).Reset()
	ResetActors(w)
	ResetGhostDotCounters(w)
}
