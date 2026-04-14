package main

type Game struct {
	zoom          float32
	pacman        bool
	paused        bool
	lives         int
	score         int
	highScore     int
	level         int
	frameCount    int
	dotsEaten     int
	dotsRemaining int
	isFrightMode  bool
	frightFlashes int
	frightFrames  int
	spec          LevelSpecTable
	ghostState    GhostState
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

}
