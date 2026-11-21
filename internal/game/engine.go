package game

import (
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/audio"
	"pacfan/internal/component"
	"pacfan/internal/render"
	"pacfan/internal/scene"
	"pacfan/internal/system"
)

// Options configures how a new game session is initialized.
type Options struct {
	PacmanMode    bool
	Zoom          float32
	Players       int    // Default player count selected on the attract screen (1 or 2)
	Mute          bool   // Disable audio playback when true
	HighScorePath string // Optional override for high-score persistence file (used as-is for either mode)
}

// Engine ties together the Donburi ECS World, simulation system pipeline,
// scene manager, loaded Ebitengine art/audio assets, and renderer without any package-level globals.
type Engine struct {
	World              donburi.World
	art                *render.Art
	audio              audio.Player
	input              *render.EbitenInput
	renderer           *render.Renderer
	pipeline           *system.Pipeline
	scenes             *scene.Manager
	highScorePath      string
	lastSavedHighScore int

	assetsFS     fs.FS
	opts         Options
	audioByMode  map[bool]audio.Player // loaded per mode so switching back is instant
	switchToggle bool                  // set by the attract screen; applied after the frame
}

// WindowTitle returns the window title for the given game mode.
func WindowTitle(pacmanMode bool) string {
	if pacmanMode {
		return "Pac-Man"
	}
	return "Ms. Pac-Man"
}

// NewEngine constructs a fully initialized game world, loads level 1, spawns
// entities, registers event subscribers, loads persisted high score, and wires
// up rendering, input, and scenes starting at AttractScene.
func NewEngine(opts Options, assetsFS fs.FS) *Engine {
	e := &Engine{
		art:         render.NewArt(assetsFS),
		input:       render.NewEbitenInput(),
		assetsFS:    assetsFS,
		audioByMode: make(map[bool]audio.Player),
	}
	e.start(opts)
	return e
}

// start builds a fresh session (world, scenes, high score, audio) for opts,
// reusing the engine's art and input.
func (e *Engine) start(opts Options) {
	world := donburi.NewWorld()

	players := max(1, min(opts.Players, 2))
	if opts.Players <= 0 {
		players = 1
	}

	gameEntry := archetype.CreateGameState(world, opts.PacmanMode, opts.Zoom)
	mazeEntry := archetype.CreateMaze(world, opts.PacmanMode)
	archetype.CreatePRNG(world)

	hsPath := opts.HighScorePath
	var savedHS int
	if hsPath == "" {
		hsPath = DefaultHighScorePath(opts.PacmanMode)
		savedHS = loadModeHighScore(hsPath, opts.PacmanMode)
	} else {
		savedHS = LoadHighScore(hsPath)
	}

	gameState := component.GameState.Get(gameEntry)
	maze := component.Maze.Get(mazeEntry)
	gameState.PlayerCount = players
	gameState.HighScore = savedHS
	gameState.SetLevel(1)
	gameState.DotsEaten = 0
	gameState.DotsRemaining = maze.LoadLevel(1, e.art.IsPixelLit)

	archetype.SpawnPlayer(world)
	archetype.SpawnAllGhosts(world)

	audioPlayer := e.audioFor(opts)

	system.RegisterEventHandlers(world)
	audio.RegisterAudioEventHandlers(world, audioPlayer)

	renderer := render.NewRenderer(e.art)
	pipeline := system.NewPipeline(e.input)

	sceneCtx := &scene.Context{
		World:          world,
		Input:          pipeline.Input(),
		Pipeline:       pipeline,
		Drawer:         renderer,
		Audio:          audioPlayer,
		IsPixelLit:     e.art.IsPixelLit,
		DefaultPlayers: players,
		SwitchMode:     func() { e.switchToggle = true },
	}
	manager := scene.NewManager(nil)
	sceneCtx.Manager = manager
	manager.Start(scene.NewAttractScene(sceneCtx))

	e.World = world
	e.audio = audioPlayer
	e.renderer = renderer
	e.pipeline = pipeline
	e.scenes = manager
	e.highScorePath = hsPath
	e.lastSavedHighScore = savedHS
	e.opts = opts
}

// audioFor returns the audio player for opts' mode, loading it on first use.
func (e *Engine) audioFor(opts Options) audio.Player {
	if opts.Mute {
		return audio.NoopPlayer{}
	}
	if p, ok := e.audioByMode[opts.PacmanMode]; ok {
		return p
	}
	p := audio.NewEbitenPlayer(e.assetsFS, opts.PacmanMode)
	e.audioByMode[opts.PacmanMode] = p
	return p
}

// PacmanMode reports whether the engine is currently running classic Pac-Man.
func (e *Engine) PacmanMode() bool {
	return e.opts.PacmanMode
}

// switchMode saves the current high score and restarts at the attract screen
// in the other game mode, carrying over the mute setting.
func (e *Engine) switchMode() {
	e.persistHighScoreIfNeeded()
	muted := e.audio.Muted()
	e.audio.StopAll()

	opts := e.opts
	opts.PacmanMode = !opts.PacmanMode
	e.start(opts)
	e.audio.SetMuted(muted)
	ebiten.SetWindowTitle(WindowTitle(opts.PacmanMode))
}

// PollInput latches edge-triggered hardware key presses for the current frame.
// Call once per Ebitengine Update before stepping the fixed-timestep clock.
func (e *Engine) PollInput() {
	e.input.Poll()
}

// Update advances the active scene by one frame and persists HighScore changes.
func (e *Engine) Update() {
	e.scenes.Update()
	e.persistHighScoreIfNeeded()
	if e.switchToggle {
		e.switchToggle = false
		e.switchMode()
	}
}

// Draw renders the active scene into dst.
func (e *Engine) Draw(dst *ebiten.Image) {
	e.renderer.SetTarget(dst)
	e.scenes.Draw()
}

// Unload persists any new high score and releases audio, GPU images, and shaders.
func (e *Engine) Unload() {
	e.persistHighScoreIfNeeded()
	for _, p := range e.audioByMode {
		p.Unload()
	}
	e.art.Unload()
}

func (e *Engine) persistHighScoreIfNeeded() {
	gs := component.MustFindGameState(e.World)
	if gs.HighScore > e.lastSavedHighScore {
		if err := SaveHighScore(e.highScorePath, gs.HighScore); err == nil {
			e.lastSavedHighScore = gs.HighScore
		}
	}
}
