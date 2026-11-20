package main

import (
	"embed"
	"flag"
	"runtime/debug"

	rl "github.com/gen2brain/raylib-go/raylib"

	"pacfan/internal/config"
	"pacfan/internal/game"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	// Reduce GC frequency during the 61 FPS arcade loop.
	debug.SetGCPercent(200)

	var zoom int
	var pacman bool
	flag.IntVar(&zoom, "zoom", 3, "zoom level")
	flag.BoolVar(&pacman, "pacman", false, "pacman mode")
	flag.Parse()

	zoom = max(1, min(zoom, 10))

	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(
		config.ScreenWidth*config.Tile*int32(zoom),
		config.ScreenHeight*config.Tile*int32(zoom),
		"Pacman",
	)
	defer rl.CloseWindow()

	rl.SetTargetFPS(config.TargetFPS) // arcade game was clocked to 60.606061

	engine := game.NewEngine(game.Options{
		PacmanMode: pacman,
		Zoom:       float32(zoom),
	}, assetsFS)
	defer engine.Unload()

	for !rl.WindowShouldClose() {
		engine.Update()
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		engine.Draw()
		rl.EndDrawing()
	}
}
