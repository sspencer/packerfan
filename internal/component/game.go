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
	LevelComplete    bool
	PlayerDying      bool
	GameOver         bool
	Spec             config.LevelSpecTable
	GhostState       config.GhostState
}

// SetLevel updates the active level number and loads its speed/timing specification.
func (g *GameStateData) SetLevel(level int) {
	g.Level = level
	g.Spec = config.GetLevelSpec(level)
}

// ResetForNextLevel advances to the given level and resets round-scoped timers,
// counters, frightened state, and ghost-eaten popup state while preserving score,
// high score, lives, zoom, and mode flags.
func (g *GameStateData) ResetForNextLevel(level, dotsRemaining int) {
	g.SetLevel(level)
	g.FrameCount = 0
	g.DotsEaten = 0
	g.DotsRemaining = dotsRemaining
	g.IsFrightMode = false
	g.FrightFlashes = 0
	g.FrightFrames = 0
	g.NumGhostsEaten = 0
	g.GhostEatenPause = 0
	g.GhostEatenX = 0
	g.GhostEatenY = 0
	g.GhostEatenEntity = donburi.Null
	g.GhostEatenScore = 0
	g.GhostState = config.Scatter
	g.LevelComplete = false
	g.PlayerDying = false
	g.GameOver = false
}

// ResetForNextLife resets wave timing, frightened state, and transient popup
// state when respawning on the same level while preserving level, dots, score,
// high score, and remaining lives.
func (g *GameStateData) ResetForNextLife() {
	g.FrameCount = 0
	g.IsFrightMode = false
	g.FrightFlashes = 0
	g.FrightFrames = 0
	g.NumGhostsEaten = 0
	g.GhostEatenPause = 0
	g.GhostEatenX = 0
	g.GhostEatenY = 0
	g.GhostEatenEntity = donburi.Null
	g.GhostEatenScore = 0
	g.GhostState = config.Scatter
	g.PlayerDying = false
}

// ResetForNewGame resets the session back to level 1 with full lives and zero
// score while preserving HighScore, Zoom, and PacmanMode.
func (g *GameStateData) ResetForNewGame(dotsRemaining int) {
	g.Lives = config.InitialLives
	g.Score = 0
	g.Paused = false
	g.ResetForNextLevel(1, dotsRemaining)
}

var GameState = donburi.NewComponentType[GameStateData]()

// MustFindGameState returns the singleton GameState component in the world.
func MustFindGameState(w donburi.World) *GameStateData {
	return GameState.Get(GameState.MustFirst(w))
}
