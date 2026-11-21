package main

import (
	"fmt"
	"image/color"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	flag "github.com/spf13/pflag"

	"pacfan/assets"
	"pacfan/internal/audio"
	"pacfan/internal/config"
	"pacfan/internal/game"
	"pacfan/internal/render"
)

const defaultWindowedZoom = 3

func main() {
	// Reduce GC frequency during the 61 FPS arcade loop.
	debug.SetGCPercent(200)

	var zoom int
	var fullscreen bool
	var msPacman bool
	var players int
	var mute bool
	var replayPath string
	var replaySpeed float64
	var artFile string
	flag.IntVarP(&zoom, "zoom", "z", 0, "zoom level (1..10; 0 = auto-fit fullscreen or 3x windowed)")
	flag.BoolVar(&fullscreen, "fullscreen", true, "run in fullscreen mode")
	flag.BoolVarP(&msPacman, "ms", "m", false, "Ms. Pac-Man mode (default: classic Pac-Man)")
	flag.IntVarP(&players, "players", "p", 1, "default number of players (1 or 2)")
	flag.BoolVar(&mute, "mute", false, "disable audio playback")
	flag.StringVarP(&replayPath, "replay", "r", "", "play a recorded .pfr replay file or directory")
	flag.Float64VarP(&replaySpeed, "replay-speed", "s", 1.0, "replay playback speed multiplier (0.25..16)")
	flag.StringVarP(&artFile, "art", "a", "", "override game spritesheet PNG path for --replay mode")
	flag.Parse()
	pacman := !msPacman

	var maxZoom float32
	windowZoom := defaultWindowedZoom
	if zoom > 0 {
		windowZoom = max(1, min(zoom, 10))
		if fullscreen {
			maxZoom = float32(windowZoom)
		}
	}
	players = max(1, min(players, 2))

	if replayPath != "" {
		runReplay(replayPath, artFile, fullscreen, windowZoom, maxZoom, replaySpeed)
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
and rebuild to enable missing sounds, or pass --mute to silence this warning.
`, strings.Join(missing, "\n  "))
		}
	}

	configureWindow(game.WindowTitle(pacman), fullscreen, windowZoom)

	canvas := render.NewCanvas(maxZoom)
	defer canvas.Unload()

	engine := game.NewEngine(game.Options{
		PacmanMode: pacman,
		Zoom:       1, // Renderer draws into the native 224x288 canvas; Canvas scales to fit screen.
		Players:    players,
		Mute:       mute,
	}, assets.FS)
	defer engine.Unload()

	app := &arcadeApp{
		canvas: canvas,
		engine: engine,
		clock:  config.NewArcadeClock(),
	}
	if err := ebiten.RunGame(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// maxFrameDT caps the wall-clock time a single Update may account for, so a
// stall (window drag, sleep, debugger break) doesn't trigger a burst of
// catch-up work on the next frame.
const maxFrameDT = 0.25

// windowApp holds the window controls and frame timing shared by the arcade
// and replay ebiten.Game implementations.
type windowApp struct {
	lastTick time.Time
}

// beginFrame handles quit and fullscreen keys and returns the seconds elapsed
// since the previous frame (0 on the first frame, capped at maxFrameDT).
// quit reports that the game should return ebiten.Termination.
func (w *windowApp) beginFrame() (dt float64, quit bool) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return 0, true
	}
	if fullscreenTogglePressed() {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	now := time.Now()
	if !w.lastTick.IsZero() {
		dt = min(now.Sub(w.lastTick).Seconds(), maxFrameDT)
	}
	w.lastTick = now
	return dt, false
}

func (w *windowApp) Layout(outsideWidth, outsideHeight int) (int, int) {
	return max(1, outsideWidth), max(1, outsideHeight)
}

type arcadeApp struct {
	windowApp
	canvas *render.Canvas
	engine *game.Engine
	clock  *config.FixedStepClock
}

func (a *arcadeApp) Update() error {
	dt, quit := a.beginFrame()
	if quit {
		return ebiten.Termination
	}

	a.engine.PollInput()
	steps := a.clock.Advance(dt)
	for range steps {
		a.engine.Update()
	}
	return nil
}

func (a *arcadeApp) Draw(screen *ebiten.Image) {
	a.canvas.Clear()
	a.engine.Draw(a.canvas.Image())

	screen.Fill(color.Black)
	a.canvas.Present(screen)
}

func runReplay(replayPath, artFile string, fullscreen bool, windowZoom int, maxZoom float32, speed float64) {
	if err := render.CheckAssets(assets.FS); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	player, err := game.NewReplayPlayer(assets.FS, replayPath, artFile, 1, speed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	title := "Ms. Pac-Man [Replay]"
	if player.Replay().Header.Pacman {
		title = "Pac-Man [Replay]"
	}

	configureWindow(title, fullscreen, windowZoom)

	canvas := render.NewCanvas(maxZoom)
	defer canvas.Unload()

	viewer := game.NewReplayViewer(assets.FS, player)
	defer viewer.Unload()

	app := &replayApp{
		canvas: canvas,
		viewer: viewer,
	}
	if err := ebiten.RunGame(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type replayApp struct {
	windowApp
	canvas *render.Canvas
	viewer *game.ReplayViewer
}

func (r *replayApp) Update() error {
	dt, quit := r.beginFrame()
	if quit {
		return ebiten.Termination
	}

	r.viewer.Update(dt)
	return nil
}

func (r *replayApp) Draw(screen *ebiten.Image) {
	r.canvas.Clear()
	r.viewer.DrawWorld(r.canvas.Image())

	screen.Fill(color.Black)
	vp := r.canvas.Present(screen)
	r.viewer.DrawOverlay(screen, vp)
}

func configureWindow(title string, fullscreen bool, windowZoom int) {
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(render.NativeWidth*windowZoom, render.NativeHeight*windowZoom)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(fullscreen)
	ebiten.SetTPS(ebiten.SyncWithFPS)
}

func fullscreenTogglePressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		return true
	}
	altDown := ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight)
	return altDown && inpututil.IsKeyJustPressed(ebiten.KeyEnter)
}
