package main

import (
	"flag"
	"runtime/debug"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	GameWidth    = 28
	GameHeight   = 31
	TopOffset    = 3
	ScreenWidth  = GameWidth
	ScreenHeight = GameHeight + 5
	Tile         = 8
	Sprite       = 16
	Half         = Tile / 2
	PelletScore  = 10
	PelletPause  = 1
	PowerScore   = 50
	PowerPause   = 3
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

	fontTexture := rl.LoadTexture("assets/text.png")
	defer rl.UnloadTexture(fontTexture)
	gameTexture := rl.LoadTexture("assets/game.png")
	defer rl.UnloadTexture(gameTexture)
	gameImage := rl.LoadImageFromTexture(gameTexture)
	defer rl.UnloadImage(gameImage)
	shader := chromaShader("assets/chroma_key.fs")
	defer rl.UnloadShader(shader)

	game := NewGame(pacman, float32(zoom), fontTexture, gameTexture, gameImage, shader)
	game.initMaze()
	//fmt.Println(game.maze.String())

	for !rl.WindowShouldClose() {
		game.Update()
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		game.Draw()
		rl.EndDrawing()
	}
}
