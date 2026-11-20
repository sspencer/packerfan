package game

import (
	"io/fs"

	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/component"
	"pacfan/internal/render"
	"pacfan/internal/scene"
	"pacfan/internal/system"
)

// Options configures how a new game session is initialized.
type Options struct {
	PacmanMode bool
	Zoom       float32
}

// Engine ties together the Donburi ECS World, simulation system pipeline,
// scene manager, loaded Raylib art assets, and renderer without any package-level globals.
type Engine struct {
	World    donburi.World
	art      *render.Art
	renderer *render.Renderer
	pipeline *system.Pipeline
	scenes   *scene.Manager
}

// NewEngine constructs a fully initialized game world, loads level 1, spawns
// entities, registers event subscribers, and wires up rendering, input, and scenes.
func NewEngine(opts Options, assetsFS fs.FS) *Engine {
	art := render.NewArt(assetsFS)
	world := donburi.NewWorld()

	gameEntry := archetype.CreateGameState(world, opts.PacmanMode, opts.Zoom)
	mazeEntry := archetype.CreateMaze(world, opts.PacmanMode)
	archetype.CreatePRNG(world)

	gameState := component.GameState.Get(gameEntry)
	maze := component.Maze.Get(mazeEntry)
	gameState.SetLevel(1)
	gameState.DotsEaten = 0
	gameState.DotsRemaining = maze.LoadLevel(1, art.IsPixelLit)

	archetype.SpawnPlayer(world)
	archetype.SpawnAllGhosts(world)

	system.RegisterEventHandlers(world)

	renderer := render.NewRenderer(art)
	pipeline := system.NewPipeline(render.NewRaylibInput())

	sceneCtx := &scene.Context{
		World:      world,
		Input:      pipeline.Input(),
		Pipeline:   pipeline,
		Drawer:     renderer,
		IsPixelLit: art.IsPixelLit,
	}
	manager := scene.NewManager(nil)
	sceneCtx.Manager = manager
	manager.Start(scene.NewGameStartReadyScene(sceneCtx))

	return &Engine{
		World:    world,
		art:      art,
		renderer: renderer,
		pipeline: pipeline,
		scenes:   manager,
	}
}

// Update advances the active scene by one frame.
func (e *Engine) Update() {
	e.scenes.Update()
}

// Draw renders the active scene.
func (e *Engine) Draw() {
	e.scenes.Draw()
}

// Unload releases GPU textures, images, and shaders.
func (e *Engine) Unload() {
	e.art.Unload()
}
