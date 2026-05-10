package main

import (
	"flag"
	"runtime/debug"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	GameWidth       = 28
	GameHeight      = 31
	TopOffset       = 3
	ScreenWidth     = GameWidth
	ScreenHeight    = GameHeight + 5
	Tile            = 8
	Sprite          = 16
	Half            = Tile / 2
	PelletScore     = 10
	PelletPause     = 1
	PowerScore      = 50
	PowerPause      = 3
	GhostEatenPause = 61
)

var (
	art        *Art
	game       *Game
	maze       *Maze
	player     *Player
	blinky     *Ghost
	inky       *Ghost
	pinky      *Ghost
	clyde      *Ghost
	eventStack *EventStack
)

func main() {
	// less GC
	debug.SetGCPercent(200)

	var zoom int
	var pacman bool
	flag.IntVar(&zoom, "zoom", 3, "zoom level")
	flag.BoolVar(&pacman, "pacman", false, "pacman mode")
	flag.Parse()

	zoom = max(1, min(zoom, 10))

	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(ScreenWidth*Tile*int32(zoom), ScreenHeight*Tile*int32(zoom), "Pacman")
	defer rl.CloseWindow()

	rl.SetTargetFPS(61) // arcade game was clocked to 60.606061
	initPrng()
	art = NewArt()
	game = NewGame(pacman, float32(zoom))
	maze = NewMaze(pacman)
	eventStack = NewEventStack()
	game.setLevel(1)
	maze.setLevel(1)
	player = NewPlayer()
	blinky = NewBlinky()
	inky = NewInky()
	pinky = NewPinky()
	clyde = NewClyde()

	//fmt.Println(maze.String())

	for !rl.WindowShouldClose() {
		updateGame()
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		drawGame()
		rl.EndDrawing()
	}

	art.unload()
}
