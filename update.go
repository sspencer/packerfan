package main

func (g *Game) Update() {
	g.updatePlayer(g.player)
	g.handleEvents()
}

func (g *Game) handleEvents() {
	for {
		event, ok := g.eventStack.Pop()
		if !ok {
			return
		}
		g.handleEvent(event)
	}
}

func (g *Game) handleEvent(event Event) {
	if event.X < 0 || event.X >= GameWidth || event.Y < 0 || event.Y >= GameHeight {
		return
	}

	switch event.Type {
	case PlayerAteDot:
		g.maze[event.Y][event.X] = Empty
		g.score += PelletScore
		if g.score > g.highScore {
			g.highScore = g.score
		}
		g.dotsEaten++
		g.dotsRemaining--
		g.player.dotPause += PelletPause

	case PlayerAtePowerUp:
		g.maze[event.Y][event.X] = Empty
		g.score += PowerScore
		if g.score > g.highScore {
			g.highScore = g.score
		}
		g.dotsEaten++
		g.dotsRemaining--
		g.player.dotPause += PowerPause
	}
}
