package game

import (
	"io/fs"

	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/component"
	"pacfan/internal/render"
	"pacfan/internal/system"
)

// Options configures how a new game session is initialized.
type Options struct {
	PacmanMode bool
	Zoom       float32
}

// Engine ties together the Donburi ECS World, simulation system pipeline,
// loaded Raylib art assets, and renderer without any package-level globals.
type Engine struct {
	World    donburi.World
	art      *render.Art
	renderer *render.Renderer
	pipeline *system.Pipeline
}

// NewEngine constructs a fully initialized game world, loads level 1, spawns
// entities, registers event subscribers, and wires up rendering and input.
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

	return &Engine{
		World:    world,
		art:      art,
		renderer: render.NewRenderer(art),
		pipeline: system.NewPipeline(render.NewRaylibInput()),
	}
}

// Update advances the simulation by one frame.
func (e *Engine) Update() {
	e.pipeline.Update(e.World)
}

// Draw renders the current ECS world state.
func (e *Engine) Draw() {
	e.renderer.Draw(e.World)
}

// Unload releases GPU textures, images, and shaders.
func (e *Engine) Unload() {
	e.art.Unload()
}
