package main

import rl "github.com/gen2brain/raylib-go/raylib"

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

	if rl.IsKeyPressed(rl.KeySpace) {
		g.paused = !g.paused
	}
}
