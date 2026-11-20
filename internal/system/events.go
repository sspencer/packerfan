package system

import (
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

	recordDotForGhostHouse := func(w donburi.World, gameState *component.GameStateData) {
		gameState.FramesSinceLastDot = 0
		if gameState.UseGlobalDotCounter {
			gameState.GlobalDotCounter++
			return
		}
		if preferred := preferredInHouseGhost(w, ghostQuery); preferred != nil {
			component.Ghost.Get(preferred).DotCounter++
		}
	}

	maybeSpawnFruit := func(w donburi.World, gameState *component.GameStateData, maze *component.MazeData) {
		firstDots := config.MsFruitFirstSpawnDots
		secondDots := config.MsFruitSecondSpawnDots
		if gameState.PacmanMode {
			firstDots = config.FruitFirstSpawnDots
			secondDots = config.FruitSecondSpawnDots
		}
		shouldSpawn := (gameState.DotsEaten == firstDots && gameState.FruitSpawnedCount == 0) ||
			(gameState.DotsEaten == secondDots && gameState.FruitSpawnedCount == 1)
		if !shouldSpawn {
			return
		}
		house := maze.GhostHouse()
		prng := component.MustFindPRNG(w)
		centerFruitPos := config.Vector{X: house.Center.X, Y: house.Center.Y + 3*config.Tile}

		gameState.FruitActive = true
		gameState.FruitSpawnedCount++
		gameState.FruitExiting = false
		gameState.FruitMotion = component.MotionData{}
		gameState.FruitBounceFrame = 0

		if gameState.PacmanMode {
			span := config.FruitMaxFrames - config.FruitMinFrames + 1
			gameState.FruitMoving = false
			gameState.FruitSymbol = gameState.Spec.BonusSymbol
			gameState.FruitPoints = int(gameState.Spec.BonusPoints)
			gameState.FruitTimer = config.FruitMinFrames + int(prng.NextByte())%span
			gameState.FruitPos = centerFruitPos
			return
		}

		// Ms. Pac-Man: bouncing fruit enters from a random tunnel, visits the maze
		// center below the ghost house, and exits out a random tunnel.
		sym, pts := config.MsPacmanBonus(gameState.Level, prng.NextByte())
		tunnels := maze.TunnelRows()
		entryRow := tunnels[int(prng.NextByte())%len(tunnels)]
		exitRow := tunnels[int(prng.NextByte())%len(tunnels)]
		entryLeft := prng.NextByte()%2 == 0
		exitLeft := prng.NextByte()%2 == 0

		entryX := (config.GameWidth - 1) * config.Tile
		entryDir := config.Left
		if entryLeft {
			entryX = 0
			entryDir = config.Right
		}
		exitX := (config.GameWidth - 1) * config.Tile
		if exitLeft {
			exitX = 0
		}

		gameState.FruitMoving = true
		gameState.FruitSymbol = sym
		gameState.FruitPoints = pts
		gameState.FruitTimer = config.MsFruitMaxFrames
		gameState.FruitPos = config.Vector{X: entryX, Y: entryRow * config.Tile}
		gameState.FruitDir = entryDir
		gameState.FruitTarget = centerFruitPos
		gameState.FruitExitPos = config.Vector{X: exitX, Y: exitRow * config.Tile}
	}

	addScore := func(w donburi.World, gameState *component.GameStateData, points int) {
		if gameState.AddScore(points) {
			event.ExtraLifeAwarded.Publish(w, event.ExtraLifeAwardedEvent{})
		}
	}

	event.DotEaten.Subscribe(w, func(w donburi.World, ev event.DotEatenEvent) {
		maze := component.MustFindMaze(w)
		if !maze.InBounds(ev.X, ev.Y) {
			return
		}
		gameState := component.MustFindGameState(w)

		maze.SetCell(ev.X, ev.Y, component.Empty)
		addScore(w, gameState, config.PelletScore)
		gameState.DotsEaten++
		gameState.DotsRemaining--
		recordDotForGhostHouse(w, gameState)
		maybeSpawnFruit(w, gameState, maze)
		if gameState.DotsRemaining == 0 {
			event.LevelCompleted.Publish(w, event.LevelCompletedEvent{})
		}

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
		addScore(w, gameState, config.PowerScore)
		gameState.DotsEaten++
		gameState.DotsRemaining--
		recordDotForGhostHouse(w, gameState)
		maybeSpawnFruit(w, gameState, maze)
		gameState.NumGhostsEaten = 0
		if gameState.DotsRemaining == 0 {
			event.LevelCompleted.Publish(w, event.LevelCompletedEvent{})
		}

		if playerEntry, ok := playerQuery.First(w); ok {
			component.Motion.Get(playerEntry).DotPause += config.PowerPause
		}

		if gameState.Spec.FrightenedFrames > 0 {
			gameState.IsFrightMode = true
			gameState.FrightFrames = gameState.Spec.FrightenedFrames
			gameState.FrightFlashes = gameState.Spec.FrightenedFlashes
			for ghostEntry := range ghostQuery.Iter(w) {
				g := component.Ghost.Get(ghostEntry)
				switch g.State {
				case config.Chase, config.Scatter, config.Frightened:
					g.State = config.Frightened
					g.Reverse = true
					g.FrightenedInHouse = false
				case config.InHouse, config.LeavingHouse:
					g.FrightenedInHouse = true
				}
			}
		} else {
			// On no-fright levels (17, 19+), energizers still reverse active ghosts
			// without turning them blue or edible.
			for ghostEntry := range ghostQuery.Iter(w) {
				g := component.Ghost.Get(ghostEntry)
				if g.State == config.Chase || g.State == config.Scatter {
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
		ghost.HasNextDir = false

		gameState := component.MustFindGameState(w)
		gameState.NumGhostsEaten++
		points := config.GhostScore(gameState.NumGhostsEaten)
		addScore(w, gameState, points)
		gameState.GhostEatenPause = config.GhostEatenPause
		gameState.GhostEatenX = ev.X
		gameState.GhostEatenY = ev.Y
		gameState.GhostEatenEntity = ev.Entity
		gameState.GhostEatenScore = points
	})

	event.PlayerEaten.Subscribe(w, func(w donburi.World, ev event.PlayerEatenEvent) {
		if !w.Valid(ev.Entity) {
			return
		}
		entry := w.Entry(ev.Entity)
		actor := component.Actor.Get(entry)
		if actor.IsEaten {
			return
		}
		actor.IsEaten = true

		gameState := component.MustFindGameState(w)
		gameState.PlayerDying = true
	})

	event.GhostStateChanged.Subscribe(w, func(w donburi.World, ev event.GhostStateChangedEvent) {
		gameState := component.MustFindGameState(w)
		gameState.GhostState = ev.State
		for ghostEntry := range ghostQuery.Iter(w) {
			g := component.Ghost.Get(ghostEntry)
			switch g.State {
			case config.Chase, config.Scatter:
				g.State = gameState.GhostState
				g.Reverse = true
			case config.InHouse, config.LeavingHouse:
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
			g.FrightenedInHouse = false
			if g.State == config.Frightened {
				g.State = gameState.GhostState
				g.HasNextDir = false
			}
		}
	})

	event.LevelCompleted.Subscribe(w, func(w donburi.World, _ event.LevelCompletedEvent) {
		gameState := component.MustFindGameState(w)
		if gameState.LevelComplete {
			return
		}
		gameState.LevelComplete = true
	})

	event.FruitEaten.Subscribe(w, func(w donburi.World, ev event.FruitEatenEvent) {
		gameState := component.MustFindGameState(w)
		if !gameState.FruitActive {
			return
		}
		gameState.FruitActive = false
		gameState.FruitMoving = false
		gameState.FruitExiting = false
		gameState.FruitTimer = 0
		addScore(w, gameState, ev.Points)
		gameState.FruitScoreTimer = config.FruitScorePopupFrames
		gameState.FruitScoreValue = ev.Points
	})
}
