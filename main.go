package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	rl "github.com/gen2brain/raylib-go/raylib"

	"pacfan/internal/config"
	"pacfan/internal/game"
	"pacfan/internal/render"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	// Reduce GC frequency during the 61 FPS arcade loop.
	debug.SetGCPercent(200)

	var zoom int
	var pacman bool
	var players int
	flag.IntVar(&zoom, "zoom", 3, "zoom level")
	flag.BoolVar(&pacman, "pacman", false, "pacman mode")
	flag.IntVar(&players, "players", 1, "default number of players (1 or 2)")
	flag.Parse()

	zoom = max(1, min(zoom, 10))
	players = max(1, min(players, 2))

	// Assets are embedded at build time, so check them before opening a window.
	if err := render.CheckAssets(assetsFS); err != nil {
		fmt.Fprintf(os.Stderr, `error: %v

The game artwork is not included in this repository for copyright reasons.
Place the required PNG files in the assets/ directory (see assets/README.md
for the expected format), then rebuild: the assets are embedded into the
binary at build time, so adding them without rebuilding has no effect.
`, err)
		os.Exit(1)
	}

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
		Players:    players,
	}, assetsFS)
	defer engine.Unload()

	clock := config.NewArcadeClock()

	for !rl.WindowShouldClose() {
		steps := clock.Advance(float64(rl.GetFrameTime()))
		for range steps {
			engine.Update()
		}
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		engine.Draw()
		rl.EndDrawing()
	}
}
