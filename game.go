package main

type Game struct {
	zoom            float32
	pacman          bool
	paused          bool
	lives           int
	score           int
	highScore       int
	level           int
	frameCount      int
	dotsEaten       int
	dotsRemaining   int
	isFrightMode    bool
	frightFlashes   int
	frightFrames    int
	numGhostsEaten  int
	ghostEatenPause int
	ghostEatenX     int
	ghostEatenY     int
	spec            LevelSpecTable
	ghostState      GhostState
}

func NewGame(pacman bool, zoom float32) *Game {
	game := &Game{
		zoom:       zoom,
		level:      1,
		lives:      3,
		pacman:     pacman,
		ghostState: Scatter,
	}

	return game
}

func (g *Game) update() {
	g.frameCount++

	if game.dotsRemaining == 0 {
		eventStack.Push(Event{Type: LevelComplete})
	}
	ghostState := g.getGhostState()
	if ghostState != g.ghostState {
		eventStack.Push(Event{Type: GhostStateChange, ID: int(ghostState)})
	}

	// Tick frightened mode timer; revert ghosts when it expires
	if game.isFrightMode {
		game.frightFrames--
		if game.frightFrames <= 0 {
			eventStack.Push(Event{Type: FrightModeEnd})
		}
	}

	g.checkForCollisions()
}

func (g *Game) checkForCollisions() {
	ghosts := []*Ghost{blinky, inky, pinky, clyde}

	px := float32(player.pos.x)
	py := float32(player.pos.y)
	for _, ghost := range ghosts {
		gx := float32(ghost.pos.x)
		gy := float32(ghost.pos.y)

		if !collide(px, py, gx, gy) {
			continue
		}

		if ghost.state == Frightened && !ghost.isEaten {
			eventStack.Push(Event{Type: PlayerAteGhost, ID: int(ghost.ID), X: ghost.pos.x, Y: ghost.pos.y})
			break
		}

		if (ghost.state == Chase || ghost.state == Scatter) && !player.isEaten {
			eventStack.Push(Event{Type: GhostAtePlayer, ID: int(player.ID), X: player.pos.x, Y: player.pos.y})
			break
		}
	}
}

func collide(px, py, gx, gy float32) bool {
	// Shrink each sprite hitbox to make collisions feel less unfair.
	const inset = float32(Sprite) * 0.20
	const box = float32(Sprite) - 2.0*inset

	pLeft := px + inset
	pTop := py + inset
	pRight := pLeft + box
	pBottom := pTop + box

	gLeft := gx + inset
	gTop := gy + inset
	gRight := gLeft + box
	gBottom := gTop + box

	// AABB overlap: any overlap counts.
	return pLeft < gRight && pRight > gLeft && pTop < gBottom && pBottom > gTop
}
