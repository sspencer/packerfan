package main

func updateGame() {
	game.update()
	if game.paused {
		return
	}

	player.update()
	blinky.update()
	inky.update()
	pinky.update()
	clyde.update()
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
	if event.X < 0 || event.X >= GameWidth || event.Y < 0 || event.Y >= GameHeight {
		return
	}

	switch event.Type {
	case PlayerAteDot:
		maze.setCell(event.X, event.Y, Empty)
		game.score += PelletScore
		if game.score > game.highScore {
			game.highScore = game.score
		}
		game.dotsEaten++
		game.dotsRemaining--
		player.dotPause += PelletPause

	case PlayerAtePowerUp:
		maze.setCell(event.X, event.Y, Empty)
		game.score += PowerScore
		if game.score > game.highScore {
			game.highScore = game.score
		}
		game.dotsEaten++
		game.dotsRemaining--
		player.dotPause += PowerPause

	case GhostStateChange:
		game.ghostState = GhostState(event.ID)
		ghosts := []*Ghost{blinky, inky, pinky, clyde}
		for _, g := range ghosts {
			if g.state == Chase || g.state == Scatter {
				g.state = game.ghostState
				g.reverse = true
			}
		}

	}
}
