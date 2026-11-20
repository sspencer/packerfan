package component

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/config"
)

// GameStateData is the singleton world resource holding round metrics, timers,
// difficulty specs, and global ghost wave state.
type GameStateData struct {
	Zoom             float32
	PacmanMode       bool
	Paused           bool
	Lives            int
	Score            int
	HighScore        int
	Level            int
	FrameCount       int
	DotsEaten        int
	DotsRemaining    int
	IsFrightMode     bool
	FrightFlashes    int
	FrightFrames     int
	NumGhostsEaten   int
	GhostEatenPause  int
	GhostEatenX      int
	GhostEatenY      int
	GhostEatenEntity donburi.Entity
	GhostEatenScore  int
	Spec             config.LevelSpecTable
	GhostState       config.GhostState
}

// SetLevel updates the active level number and loads its speed/timing specification.
func (g *GameStateData) SetLevel(level int) {
	g.Level = level
	g.Spec = config.GetLevelSpec(level)
}

var GameState = donburi.NewComponentType[GameStateData]()

// MustFindGameState returns the singleton GameState component in the world.
func MustFindGameState(w donburi.World) *GameStateData {
	return GameState.Get(GameState.MustFirst(w))
}
