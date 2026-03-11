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
	dotsEaten     int
	dotsRemaining int
	isFrightMode  bool
	spec          LevelSpecTable
	ghostState    GhostState
}

func NewGame(pacman bool, zoom float32) *Game {
	game := &Game{
		zoom:   zoom,
		level:  1,
		lives:  3,
		pacman: pacman,
	}

	return game
}

func (g *Game) update() {
	if rl.IsKeyPressed(rl.KeySpace) {
		g.paused = !g.paused
	}
}
