package component

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/config"
)

// PlayerSession stores per-player round state for 1-player or 2-player alternating play.
type PlayerSession struct {
	Score               int
	Lives               int
	ExtraLifeAwarded    bool
	Level               int
	DotsEaten           int
	DotsRemaining       int
	FruitSpawnedCount   int
	ElroySuspended      bool
	UseGlobalDotCounter bool
	GlobalDotCounter    int
	GhostDotCounters    [4]int
	MazeGrid            Cells
	Started             bool
}

// CloneCells returns a deep copy of a 2D maze cell grid.
func CloneCells(grid Cells) Cells {
	if len(grid) == 0 {
		return nil
	}
	cp := make(Cells, len(grid))
	for y := range grid {
		cp[y] = make([]Cell, len(grid[y]))
		copy(cp[y], grid[y])
	}
	return cp
}

// GameStateData is the singleton world resource holding round metrics, timers,
// difficulty specs, and global ghost wave state.
type GameStateData struct {
	Zoom                float32
	PacmanMode          bool
	Paused              bool
	PlayerCount         int
	ActivePlayer        int
	PlayerScores        [2]int
	Players             [2]PlayerSession
	Lives               int
	ExtraLifeAwarded    bool
	Score               int
	HighScore           int
	Level               int
	FrameCount          int
	WaveFrames          int
	DotsEaten           int
	DotsRemaining       int
	IsFrightMode        bool
	FrightFlashes       int
	FrightFrames        int
	NumGhostsEaten      int
	GhostEatenPause     int
	GhostEatenX         int
	GhostEatenY         int
	GhostEatenEntity    donburi.Entity
	GhostEatenScore     int
	LevelComplete       bool
	PlayerDying         bool
	GameOver            bool
	ElroySuspended      bool
	UseGlobalDotCounter bool
	GlobalDotCounter    int
	FramesSinceLastDot  int
	FruitActive         bool
	FruitMoving         bool
	FruitExiting        bool
	FruitTimer          int
	FruitSpawnedCount   int
	FruitSymbol         config.BonusSymbol
	FruitPoints         int
	FruitScoreTimer     int
	FruitScoreValue     int
	FruitPos            config.Vector
	FruitDir            config.Direction
	FruitTarget         config.Vector
	FruitExitPos        config.Vector
	FruitMotion         MotionData
	FruitBounceFrame    int
	Spec                config.LevelSpecTable
	GhostState          config.GhostState
}

// AddScore increments Score, updates HighScore, and awards a one-time extra
// life upon reaching config.ExtraLifeScore (10,000 points). It returns true
// when this score increment triggered the extra life.
func (g *GameStateData) AddScore(points int) bool {
	if points <= 0 {
		return false
	}
	g.Score += points
	if g.ActivePlayer >= 0 && g.ActivePlayer < len(g.PlayerScores) {
		g.PlayerScores[g.ActivePlayer] = g.Score
	}
	if g.Score > g.HighScore {
		g.HighScore = g.Score
	}
	if !g.ExtraLifeAwarded && g.Score >= config.ExtraLifeScore {
		g.ExtraLifeAwarded = true
		g.Lives++
		return true
	}
	return false
}

// ActivePlayerName returns "PLAYER ONE" or "PLAYER TWO" for banner overlays.
func (g *GameStateData) ActivePlayerName() string {
	if g.ActivePlayer == 1 {
		return "PLAYER TWO"
	}
	return "PLAYER ONE"
}

// SaveActivePlayer snapshots the active player's progress into Players[ActivePlayer].
func (g *GameStateData) SaveActivePlayer(maze *MazeData, ghostDotCounters [4]int) {
	idx := g.ActivePlayer
	if idx < 0 || idx >= len(g.Players) {
		return
	}
	g.PlayerScores[idx] = g.Score
	g.Players[idx] = PlayerSession{
		Score:               g.Score,
		Lives:               g.Lives,
		ExtraLifeAwarded:    g.ExtraLifeAwarded,
		Level:               g.Level,
		DotsEaten:           g.DotsEaten,
		DotsRemaining:       g.DotsRemaining,
		FruitSpawnedCount:   g.FruitSpawnedCount,
		ElroySuspended:      g.ElroySuspended,
		UseGlobalDotCounter: g.UseGlobalDotCounter,
		GlobalDotCounter:    g.GlobalDotCounter,
		GhostDotCounters:    ghostDotCounters,
		MazeGrid:            CloneCells(maze.Grid),
		Started:             true,
	}
}

// LoadActivePlayer restores Players[idx] into the live GameState and Maze.
func (g *GameStateData) LoadActivePlayer(idx int, maze *MazeData) [4]int {
	g.ActivePlayer = idx
	slot := g.Players[idx]
	g.Score = slot.Score
	g.PlayerScores[idx] = slot.Score
	g.Lives = slot.Lives
	g.ExtraLifeAwarded = slot.ExtraLifeAwarded
	g.SetLevel(max(1, slot.Level))
	g.DotsEaten = slot.DotsEaten
	g.DotsRemaining = slot.DotsRemaining
	g.FruitSpawnedCount = slot.FruitSpawnedCount
	g.ElroySuspended = slot.ElroySuspended
	g.UseGlobalDotCounter = slot.UseGlobalDotCounter
	g.GlobalDotCounter = slot.GlobalDotCounter
	g.FrameCount = 0
	g.WaveFrames = 0
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
	g.FramesSinceLastDot = 0
	g.FruitActive = false
	g.FruitTimer = 0
	g.FruitScoreTimer = 0
	g.FruitScoreValue = 0

	if len(slot.MazeGrid) > 0 {
		maze.Level = g.Level
		maze.Grid = CloneCells(slot.MazeGrid)
		maze.InvalidateHouseCache()
	}
	return slot.GhostDotCounters
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
	g.WaveFrames = 0
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
	g.ElroySuspended = false
	g.UseGlobalDotCounter = false
	g.GlobalDotCounter = 0
	g.FramesSinceLastDot = 0
	g.FruitActive = false
	g.FruitMoving = false
	g.FruitExiting = false
	g.FruitTimer = 0
	g.FruitSpawnedCount = 0
	g.FruitSymbol = g.Spec.BonusSymbol
	g.FruitPoints = int(g.Spec.BonusPoints)
	g.FruitScoreTimer = 0
	g.FruitScoreValue = 0
	g.FruitPos = config.Vector{}
	g.FruitTarget = config.Vector{}
	g.FruitExitPos = config.Vector{}
	g.FruitMotion = MotionData{}
	g.FruitBounceFrame = 0
}

// ResetForNextLife resets wave timing, frightened state, and transient popup
// state when respawning on the same level while preserving level, dots, score,
// high score, and remaining lives.
func (g *GameStateData) ResetForNextLife() {
	g.FrameCount = 0
	g.WaveFrames = 0
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
	g.ElroySuspended = true
	g.UseGlobalDotCounter = true
	g.GlobalDotCounter = 0
	g.FramesSinceLastDot = 0
	g.FruitActive = false
	g.FruitMoving = false
	g.FruitExiting = false
	g.FruitTimer = 0
	g.FruitScoreTimer = 0
	g.FruitScoreValue = 0
	g.FruitMotion = MotionData{}
	g.FruitBounceFrame = 0
}

// ResetForNewGame resets the session back to level 1 with full lives and zero
// score while preserving HighScore, Zoom, PacmanMode, and PlayerCount.
func (g *GameStateData) ResetForNewGame(dotsRemaining int) {
	if g.PlayerCount < 1 || g.PlayerCount > 2 {
		g.PlayerCount = 1
	}
	g.ActivePlayer = 0
	g.PlayerScores = [2]int{}
	g.Players = [2]PlayerSession{}
	for i := range g.PlayerCount {
		g.Players[i] = PlayerSession{
			Lives: config.InitialLives,
			Level: 1,
		}
	}
	g.Players[0].Started = true
	g.Lives = config.InitialLives
	g.ExtraLifeAwarded = false
	g.Score = 0
	g.Paused = false
	g.ResetForNextLevel(1, dotsRemaining)
}

var GameState = donburi.NewComponentType[GameStateData]()

// MustFindGameState returns the singleton GameState component in the world.
func MustFindGameState(w donburi.World) *GameStateData {
	return GameState.Get(GameState.MustFirst(w))
}
