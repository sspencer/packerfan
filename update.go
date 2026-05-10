package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func inputUpdate() {
	if rl.IsKeyPressed(rl.KeySpace) {
		game.paused = !game.paused
	}
}

func updateGame() {
	inputUpdate()

	if game.paused {
		return
	}

	game.update()
	handleEvents()

	blinky.update()
	inky.update()
	pinky.update()
	clyde.update()
	handleEvents()

	player.update()
	handleEvents()
}

func handleEvents() {
	for {
		event, ok := eventStack.Pop()
		if !ok {
			return
		}
		handleEvent(event)
	}
}

func handleEvent(event Event) {
	switch event.Type {
	case PlayerAteDot:
		if event.X < 0 || event.X >= GameWidth || event.Y < 0 || event.Y >= GameHeight {
			return
		}
		maze.setCell(event.X, event.Y, Empty)
		game.score += PelletScore
		if game.score > game.highScore {
			game.highScore = game.score
		}
		game.dotsEaten++
		game.dotsRemaining--
		player.dotPause += PelletPause

	case PlayerAtePowerUp:
		if event.X < 0 || event.X >= GameWidth || event.Y < 0 || event.Y >= GameHeight {
			return
		}
		maze.setCell(event.X, event.Y, Empty)
		game.score += PowerScore
		if game.score > game.highScore {
			game.highScore = game.score
		}
		game.dotsEaten++
		game.dotsRemaining--
		game.numGhostsEaten = 0
		player.dotPause += PowerPause

		if game.spec.FrightenedFrames > 0 {
			game.isFrightMode = true
			game.frightFrames = game.spec.FrightenedFrames
			game.frightFlashes = game.spec.FrightenedFlashes
			for _, g := range []*Ghost{blinky, inky, pinky, clyde} {
				if g.state == Chase || g.state == Scatter || g.state == Frightened {
					g.state = Frightened
					g.reverse = true
				}
			}
		}

	case PlayerAteGhost:
		id := event.ID
		ghost := getGhostFromID(ActorID(id))
		ghost.isEaten = true
		ghost.state = Eaten
		fmt.Println("Player ate ghost: ", getGhostFromID(ActorID(id)))

	case GhostAtePlayer:
		fmt.Println("Ghost ate player")

	case GhostStateChange:
		game.ghostState = GhostState(event.ID)
		ghosts := []*Ghost{blinky, inky, pinky, clyde}
		for _, g := range ghosts {
			if g.state == Chase || g.state == Scatter {
				g.state = game.ghostState
				g.reverse = true
			}
		}

	case FrightModeEnd:
		game.isFrightMode = false
		game.frightFrames = 0
		game.frightFlashes = 0

		//TODO need to reset frightened flashes and frames too
		for _, g := range []*Ghost{blinky, inky, pinky, clyde} {
			if g.state == Frightened {
				g.state = game.ghostState
			}
		}

	case LevelComplete:
		fmt.Println("Level complete!")
		game.paused = true
	}
}
