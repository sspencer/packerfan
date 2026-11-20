package game

import (
	"io/fs"

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
	HighScorePath string // Optional override for high-score persistence file
}

// Engine ties together the Donburi ECS World, simulation system pipeline,
// scene manager, loaded Raylib art/audio assets, and renderer without any package-level globals.
type Engine struct {
	World              donburi.World
	art                *render.Art
	audio              audio.Player
	renderer           *render.Renderer
	pipeline           *system.Pipeline
	scenes             *scene.Manager
	highScorePath      string
	lastSavedHighScore int
}

// NewEngine constructs a fully initialized game world, loads level 1, spawns
// entities, registers event subscribers, loads persisted high score, and wires
// up rendering, input, and scenes starting at AttractScene.
func NewEngine(opts Options, assetsFS fs.FS) *Engine {
	art := render.NewArt(assetsFS)
	world := donburi.NewWorld()

	players := max(1, min(opts.Players, 2))
	if opts.Players <= 0 {
		players = 1
	}

	gameEntry := archetype.CreateGameState(world, opts.PacmanMode, opts.Zoom)
	mazeEntry := archetype.CreateMaze(world, opts.PacmanMode)
	archetype.CreatePRNG(world)

	hsPath := opts.HighScorePath
	if hsPath == "" {
		hsPath = DefaultHighScorePath()
	}
	savedHS := LoadHighScore(hsPath)

	gameState := component.GameState.Get(gameEntry)
	maze := component.Maze.Get(mazeEntry)
	gameState.PlayerCount = players
	gameState.HighScore = savedHS
	gameState.SetLevel(1)
	gameState.DotsEaten = 0
	gameState.DotsRemaining = maze.LoadLevel(1, art.IsPixelLit)

	archetype.SpawnPlayer(world)
	archetype.SpawnAllGhosts(world)

	var audioPlayer audio.Player = audio.NoopPlayer{}
	if !opts.Mute {
		audioPlayer = audio.NewRaylibPlayer(assetsFS, opts.PacmanMode)
	}

	system.RegisterEventHandlers(world)
	system.RegisterAudioEventHandlers(world, audioPlayer)

	renderer := render.NewRenderer(art)
	pipeline := system.NewPipeline(render.NewRaylibInput())

	sceneCtx := &scene.Context{
		World:          world,
		Input:          pipeline.Input(),
		Pipeline:       pipeline,
		Drawer:         renderer,
		Audio:          audioPlayer,
		IsPixelLit:     art.IsPixelLit,
		DefaultPlayers: players,
	}
	manager := scene.NewManager(nil)
	sceneCtx.Manager = manager
	manager.Start(scene.NewAttractScene(sceneCtx))

	return &Engine{
		World:              world,
		art:                art,
		audio:              audioPlayer,
		renderer:           renderer,
		pipeline:           pipeline,
		scenes:             manager,
		highScorePath:      hsPath,
		lastSavedHighScore: savedHS,
	}
}

// Update advances the active scene by one frame and persists HighScore changes.
func (e *Engine) Update() {
	e.scenes.Update()
	e.persistHighScoreIfNeeded()
}

// Draw renders the active scene.
func (e *Engine) Draw() {
	e.scenes.Draw()
}

// Unload persists any new high score and releases audio, GPU textures, images, and shaders.
func (e *Engine) Unload() {
	e.persistHighScoreIfNeeded()
	e.audio.Unload()
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
