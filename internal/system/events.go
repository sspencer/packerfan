package system

import (
	"fmt"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/event"
)

// RegisterEventHandlers subscribes all gameplay event handlers to the given ECS World.
func RegisterEventHandlers(w donburi.World) {
	playerQuery := donburi.NewQuery(
		filter.Contains(component.PlayerTag, component.Motion),
	)
	ghostQuery := donburi.NewQuery(
		filter.Contains(component.GhostTag, component.Ghost),
	)

	event.DotEaten.Subscribe(w, func(w donburi.World, ev event.DotEatenEvent) {
		maze := component.MustFindMaze(w)
		if !maze.InBounds(ev.X, ev.Y) {
			return
		}
		gameState := component.MustFindGameState(w)

		maze.SetCell(ev.X, ev.Y, component.Empty)
		gameState.Score += config.PelletScore
		if gameState.Score > gameState.HighScore {
			gameState.HighScore = gameState.Score
		}
		gameState.DotsEaten++
		gameState.DotsRemaining--

		if playerEntry, ok := playerQuery.First(w); ok {
			component.Motion.Get(playerEntry).DotPause += config.PelletPause
		}
	})

	event.PowerUpEaten.Subscribe(w, func(w donburi.World, ev event.PowerUpEatenEvent) {
		maze := component.MustFindMaze(w)
		if !maze.InBounds(ev.X, ev.Y) {
			return
		}
		gameState := component.MustFindGameState(w)

		maze.SetCell(ev.X, ev.Y, component.Empty)
		gameState.Score += config.PowerScore
		if gameState.Score > gameState.HighScore {
			gameState.HighScore = gameState.Score
		}
		gameState.DotsEaten++
		gameState.DotsRemaining--
		gameState.NumGhostsEaten = 0

		if playerEntry, ok := playerQuery.First(w); ok {
			component.Motion.Get(playerEntry).DotPause += config.PowerPause
		}

		if gameState.Spec.FrightenedFrames > 0 {
			gameState.IsFrightMode = true
			gameState.FrightFrames = gameState.Spec.FrightenedFrames
			gameState.FrightFlashes = gameState.Spec.FrightenedFlashes
			for ghostEntry := range ghostQuery.Iter(w) {
				g := component.Ghost.Get(ghostEntry)
				if g.State == config.Chase || g.State == config.Scatter || g.State == config.Frightened {
					g.State = config.Frightened
					g.Reverse = true
				}
			}
		}
	})

	event.GhostEaten.Subscribe(w, func(w donburi.World, ev event.GhostEatenEvent) {
		if !w.Valid(ev.Entity) {
			return
		}
		entry := w.Entry(ev.Entity)
		actor := component.Actor.Get(entry)
		if actor.IsEaten {
			return
		}
		actor.IsEaten = true
		ghost := component.Ghost.Get(entry)
		ghost.State = config.Eaten
		ghost.Reverse = false

		gameState := component.MustFindGameState(w)
		gameState.NumGhostsEaten++
		points := config.GhostScore(gameState.NumGhostsEaten)
		gameState.Score += points
		if gameState.Score > gameState.HighScore {
			gameState.HighScore = gameState.Score
		}
		gameState.GhostEatenPause = config.GhostEatenPause
		gameState.GhostEatenX = ev.X
		gameState.GhostEatenY = ev.Y
		gameState.GhostEatenEntity = ev.Entity
		gameState.GhostEatenScore = points

		fmt.Println("Player ate ghost: ", ev.ID.String())
	})

	event.PlayerEaten.Subscribe(w, func(_ donburi.World, _ event.PlayerEatenEvent) {
		fmt.Println("Ghost ate player")
	})

	event.GhostStateChanged.Subscribe(w, func(w donburi.World, ev event.GhostStateChangedEvent) {
		gameState := component.MustFindGameState(w)
		gameState.GhostState = ev.State
		for ghostEntry := range ghostQuery.Iter(w) {
			g := component.Ghost.Get(ghostEntry)
			if g.State == config.Chase || g.State == config.Scatter {
				g.State = gameState.GhostState
				g.Reverse = true
			}
		}
	})

	event.FrightModeEnded.Subscribe(w, func(w donburi.World, _ event.FrightModeEndedEvent) {
		gameState := component.MustFindGameState(w)
		gameState.IsFrightMode = false
		gameState.FrightFrames = 0
		gameState.FrightFlashes = 0

		for ghostEntry := range ghostQuery.Iter(w) {
			g := component.Ghost.Get(ghostEntry)
			if g.State == config.Frightened {
				g.State = gameState.GhostState
			}
		}
	})

	event.LevelCompleted.Subscribe(w, func(w donburi.World, _ event.LevelCompletedEvent) {
		fmt.Println("Level complete!")
		component.MustFindGameState(w).Paused = true
	})
}
