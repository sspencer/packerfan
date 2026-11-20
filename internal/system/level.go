package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
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

func snapshotGhostDotCounters(w donburi.World) [4]int {
	var counters [4]int
	ghostQuery := donburi.NewQuery(
		filter.Contains(component.GhostTag, component.Ghost),
	)
	for entry := range ghostQuery.Iter(w) {
		g := component.Ghost.Get(entry)
		if idx := int(g.AIKind); idx >= 0 && idx < len(counters) {
			counters[idx] = g.DotCounter
		}
	}
	return counters
}

func restoreGhostDotCounters(w donburi.World, counters [4]int) {
	ghostQuery := donburi.NewQuery(
		filter.Contains(component.GhostTag, component.Ghost),
	)
	for entry := range ghostQuery.Iter(w) {
		g := component.Ghost.Get(entry)
		if idx := int(g.AIKind); idx >= 0 && idx < len(counters) {
			g.DotCounter = counters[idx]
		}
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
	alive, _ := ConsumeLifeOrSwitchPlayer(w, nil)
	return alive
}

// ConsumeLifeOrSwitchPlayer decrements the active player's lives after death.
// In 2-player mode, if the other player still has lives remaining, it snapshots
// the active player's board/progress and switches to the other player (loading
// a fresh Level 1 board on Player 2's first turn or restoring their saved board).
// Returns (alive, switchedPlayer).
func ConsumeLifeOrSwitchPlayer(w donburi.World, isPixelLit func(x, y int) bool) (bool, bool) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	if gameState.Lives > 0 {
		gameState.Lives--
	}
	gameState.PlayerDying = false

	// Save the outgoing player's post-death state (with post-death Elroy/global-counter flags).
	if gameState.Lives > 0 {
		gameState.ElroySuspended = true
		gameState.UseGlobalDotCounter = true
		gameState.GlobalDotCounter = 0
	}
	gameState.SaveActivePlayer(maze, snapshotGhostDotCounters(w))

	// In 2-player mode, alternate to the other player if they still have lives.
	if gameState.PlayerCount == 2 {
		other := 1 - gameState.ActivePlayer
		if gameState.Players[other].Lives > 0 {
			if !gameState.Players[other].Started {
				gameState.ActivePlayer = other
				gameState.Players[other].Started = true
				dotsRemaining := 0
				if isPixelLit != nil {
					dotsRemaining = maze.LoadLevel(1, isPixelLit)
				} else {
					maze.Level = 1
					dotsRemaining = gameState.DotsEaten + gameState.DotsRemaining
				}
				gameState.Score = 0
				gameState.PlayerScores[other] = 0
				gameState.Lives = config.InitialLives
				gameState.ExtraLifeAwarded = false
				gameState.ResetForNextLevel(1, dotsRemaining)
				component.MustFindPRNG(w).Reset()
				ResetActors(w)
				ResetGhostDotCounters(w)
				return true, true
			}

			counters := gameState.LoadActivePlayer(other, maze)
			component.MustFindPRNG(w).Reset()
			ResetActors(w)
			restoreGhostDotCounters(w, counters)
			return true, true
		}
	}

	if gameState.Lives <= 0 {
		gameState.Lives = 0
		gameState.GameOver = true
		return false, false
	}

	gameState.ResetForNextLife()
	component.MustFindPRNG(w).Reset()
	ResetActors(w)
	return true, false
}

// StartNewGame initializes a 1-player or 2-player game from Level 1 while
// preserving HighScore, Zoom, and PacmanMode.
func StartNewGame(w donburi.World, playerCount int, isPixelLit func(x, y int) bool) {
	gameState := component.MustFindGameState(w)
	gameState.PlayerCount = max(1, min(playerCount, 2))
	RestartGame(w, isPixelLit)
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
