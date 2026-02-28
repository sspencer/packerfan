package main

import rl "github.com/gen2brain/raylib-go/raylib"

type Game struct {
	font          rl.Texture2D
	texture       rl.Texture2D
	image         *rl.Image
	shader        rl.Shader
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
	maze          Maze
	player        *Player
	eventStack    EventStack
	spec          LevelSpecTable
	//ghosts      [4]*Ghost
}

func NewGame(pacman bool, zoom float32, font, texture rl.Texture2D, image *rl.Image, shader rl.Shader) *Game {
	maze := make(Maze, GameHeight)
	for i := range GameHeight {
		maze[i] = make([]Cell, GameWidth)
	}

	game := Game{
		font:    font,
		texture: texture,
		image:   image,
		shader:  shader,
		zoom:    zoom,
		level:   1,
		lives:   3,
		maze:    maze,
		player:  NewPlayer(),
		pacman:  pacman,
	}

	game.spec = getLevelSpec(game.level) // eventually change pacman/mspacman spec
	game.eventStack.Init()

	return &game
}
