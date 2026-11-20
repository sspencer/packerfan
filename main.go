package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"pacfan/assets"
	"pacfan/internal/audio"
	"pacfan/internal/config"
	"pacfan/internal/game"
	"pacfan/internal/render"
)

func main() {
	// Reduce GC frequency during the 61 FPS arcade loop.
	debug.SetGCPercent(200)

	var zoom int
	var pacman bool
	var players int
	var mute bool
	var replayPath string
	var replaySpeed float64
	var artFile string
	flag.IntVar(&zoom, "zoom", 3, "zoom level")
	flag.BoolVar(&pacman, "pacman", false, "pacman mode")
	flag.IntVar(&players, "players", 1, "default number of players (1 or 2)")
	flag.BoolVar(&mute, "mute", false, "disable audio playback")
	flag.StringVar(&replayPath, "replay", "", "play a recorded .pfr replay file or directory")
	flag.Float64Var(&replaySpeed, "replay-speed", 1.0, "replay playback speed multiplier (0.25..16)")
	flag.StringVar(&artFile, "art", "", "override game spritesheet PNG path for -replay mode")
	flag.Parse()

	zoom = max(1, min(zoom, 10))
	players = max(1, min(players, 2))

	if replayPath != "" {
		runReplay(replayPath, artFile, zoom, replaySpeed)
		return
	}

	// Assets are embedded at build time, so check them before opening a window.
	if err := render.CheckAssets(assets.FS); err != nil {
		fmt.Fprintf(os.Stderr, `error: %v

The placeholder artwork assets/game.png and assets/text.png is tracked in the
repository; regenerate it with
  go run ./tools/genexample assets/game.png assets/text.png
To use the original artwork instead, place assets/game-real.png and
assets/text-real.png locally (see assets/README.md for the expected format).
Rebuild afterwards: the assets are embedded into the binary at build time.
`, err)
		os.Exit(1)
	}

	// Verify optional audio assets before initializing audio; warn clearly if any are absent.
	if !mute {
		if missing := audio.MissingAssets(assets.FS, pacman); len(missing) > 0 {
			fmt.Fprintf(os.Stderr, `warning: missing optional audio file(s):
  %s

Audio files are not included in this repository for copyright reasons.
Place WAV files under assets/pacman/ or assets/ms-pacman/ (see assets/README.md)
and rebuild to enable missing sounds, or pass -mute to silence this warning.
`, strings.Join(missing, "\n  "))
		}
	}

	title := "Ms. Pac-Man"
	if pacman {
		title = "Pac-Man"
	}

	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(
		config.ScreenWidth*config.Tile*int32(zoom),
		config.ScreenHeight*config.Tile*int32(zoom),
		title,
	)
	defer rl.CloseWindow()

	rl.SetTargetFPS(config.TargetFPS) // arcade game was clocked to 60.606061

	engine := game.NewEngine(game.Options{
		PacmanMode: pacman,
		Zoom:       float32(zoom),
		Players:    players,
		Mute:       mute,
	}, assets.FS)
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

func runReplay(replayPath, artFile string, zoom int, speed float64) {
	if err := render.CheckAssets(assets.FS); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	player, err := game.NewReplayPlayer(assets.FS, replayPath, artFile, float32(zoom), speed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	title := "Ms. Pac-Man [Replay]"
	if player.Replay().Header.Pacman {
		title = "Pac-Man [Replay]"
	}

	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(
		config.ScreenWidth*config.Tile*int32(zoom),
		config.ScreenHeight*config.Tile*int32(zoom),
		title,
	)
	defer rl.CloseWindow()

	rl.SetTargetFPS(config.TargetFPS)

	viewer := game.NewReplayViewer(assets.FS, player, float32(zoom))
	defer viewer.Unload()

	for !rl.WindowShouldClose() {
		viewer.Update(float64(rl.GetFrameTime()))
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		viewer.Draw()
		rl.EndDrawing()
	}
}
