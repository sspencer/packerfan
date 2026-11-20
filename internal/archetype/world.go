package archetype

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

// CreateGameState creates the singleton GameState entity in the world.
func CreateGameState(w donburi.World, pacmanMode bool, zoom float32) *donburi.Entry {
	entry := w.Entry(w.Create(component.GameState))
	state := component.GameStateData{
		Zoom:       zoom,
		PacmanMode: pacmanMode,
		Level:      1,
		Lives:      config.InitialLives,
		GhostState: config.Scatter,
	}
	state.SetLevel(1)
	component.GameState.SetValue(entry, state)
	return entry
}

// CreateMaze creates the singleton Maze entity in the world.
func CreateMaze(w donburi.World, pacmanMode bool) *donburi.Entry {
	entry := w.Entry(w.Create(component.Maze))
	component.Maze.SetValue(entry, component.NewMazeData(pacmanMode))
	return entry
}

// CreatePRNG creates the singleton arcade PRNG entity in the world.
func CreatePRNG(w donburi.World) *donburi.Entry {
	entry := w.Entry(w.Create(component.PRNG))
	component.PRNG.SetValue(entry, component.NewPRNGData())
	return entry
}
