package sim

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211
)

type fnv64a uint64

func (h *fnv64a) writeByte(b byte) {
	*h ^= fnv64a(b)
	*h *= fnv64a(fnvPrime64)
}

func (h *fnv64a) writeBool(v bool) {
	if v {
		h.writeByte(1)
	} else {
		h.writeByte(0)
	}
}

func (h *fnv64a) writeInt(v int) {
	u := uint64(v)
	for range 8 {
		h.writeByte(byte(u))
		u >>= 8
	}
}

func (h *fnv64a) writeVec(v config.Vector) {
	h.writeInt(v.X)
	h.writeInt(v.Y)
}

func (h *fnv64a) writeMotion(m *component.MotionData) {
	h.writeInt(m.SpeedIndex)
	h.writeInt(m.DotPause)
	h.writeBool(m.CornerSnap)
}

var (
	hashPlayerQuery = donburi.NewQuery(
		filter.Contains(component.PlayerTag, component.Actor, component.Transform, component.Motion, component.Animation, component.PlayerInput),
	)
	hashGhostQuery = donburi.NewQuery(
		filter.Contains(component.GhostTag, component.Actor, component.Transform, component.Motion, component.Animation, component.Ghost),
	)
)

// HashWorld computes a deterministic 64-bit FNV-1a digest over all gameplay
// state in w (GameState, Maze grid, PRNG cursor, player, and ghosts in
// canonical ActorID order). Cosmetic/meta fields Zoom and HighScore are excluded.
func HashWorld(w donburi.World) uint64 {
	h := fnv64a(fnvOffset64)

	gs := component.MustFindGameState(w)
	h.writeBool(gs.PacmanMode)
	h.writeBool(gs.Paused)
	h.writeInt(gs.PlayerCount)
	h.writeInt(gs.ActivePlayer)
	h.writeInt(gs.Lives)
	h.writeBool(gs.ExtraLifeAwarded)
	h.writeInt(gs.Score)
	h.writeInt(gs.Level)
	h.writeInt(gs.FrameCount)
	h.writeInt(gs.WaveFrames)
	h.writeInt(gs.DotsEaten)
	h.writeInt(gs.DotsRemaining)
	h.writeBool(gs.IsFrightMode)
	h.writeInt(gs.FrightFlashes)
	h.writeInt(gs.FrightFrames)
	h.writeInt(gs.NumGhostsEaten)
	h.writeInt(gs.GhostEatenPause)
	h.writeInt(gs.GhostEatenX)
	h.writeInt(gs.GhostEatenY)
	h.writeInt(gs.GhostEatenScore)
	h.writeBool(gs.LevelComplete)
	h.writeBool(gs.PlayerDying)
	h.writeBool(gs.GameOver)
	h.writeBool(gs.ElroySuspended)
	h.writeBool(gs.UseGlobalDotCounter)
	h.writeInt(gs.GlobalDotCounter)
	h.writeInt(gs.FramesSinceLastDot)
	h.writeInt(int(gs.GhostState))

	// Fruit state.
	h.writeBool(gs.FruitActive)
	h.writeBool(gs.FruitMoving)
	h.writeBool(gs.FruitExiting)
	h.writeInt(gs.FruitTimer)
	h.writeInt(gs.FruitSpawnedCount)
	h.writeInt(int(gs.FruitSymbol))
	h.writeInt(gs.FruitPoints)
	h.writeInt(gs.FruitScoreTimer)
	h.writeInt(gs.FruitScoreValue)
	h.writeVec(gs.FruitPos)
	h.writeInt(int(gs.FruitDir))
	h.writeVec(gs.FruitTarget)
	h.writeVec(gs.FruitExitPos)
	h.writeMotion(&gs.FruitMotion)
	h.writeInt(gs.FruitBounceFrame)

	// Maze grid.
	maze := component.MustFindMaze(w)
	h.writeInt(maze.Level)
	h.writeBool(maze.PacmanMode)
	for y := range maze.Grid {
		row := maze.Grid[y]
		for x := range row {
			h.writeByte(byte(row[x]))
		}
	}

	// PRNG cursor.
	prng := component.MustFindPRNG(w)
	h.writeInt(int(prng.Index))

	// Player state.
	if entry, ok := hashPlayerQuery.First(w); ok {
		h.writeBool(component.Actor.Get(entry).IsEaten)
		tr := component.Transform.Get(entry)
		h.writeVec(tr.Pos)
		h.writeInt(int(tr.Dir))
		h.writeMotion(component.Motion.Get(entry))
		anim := component.Animation.Get(entry)
		h.writeInt(anim.FrameCount)
		h.writeInt(anim.FrameIndex)
		in := component.PlayerInput.Get(entry)
		h.writeInt(int(in.Requested))
		h.writeBool(in.Active)
	}

	// Ghosts ordered by canonical ActorID (Blinky, Pinky, Inky, Clyde).
	var ordered [4]*donburi.Entry
	for entry := range hashGhostQuery.Iter(w) {
		id := int(component.Actor.Get(entry).ID)
		if id >= 0 && id < len(ordered) {
			ordered[id] = entry
		}
	}
	for _, entry := range ordered {
		if entry == nil {
			h.writeByte(0)
			continue
		}
		h.writeByte(1)
		h.writeBool(component.Actor.Get(entry).IsEaten)
		tr := component.Transform.Get(entry)
		h.writeVec(tr.Pos)
		h.writeInt(int(tr.Dir))
		h.writeMotion(component.Motion.Get(entry))
		anim := component.Animation.Get(entry)
		h.writeInt(anim.FrameCount)
		h.writeInt(anim.FrameIndex)
		g := component.Ghost.Get(entry)
		h.writeInt(int(g.AIKind))
		h.writeInt(int(g.State))
		h.writeVec(g.Target)
		h.writeInt(g.PixelsMoved)
		h.writeBool(g.Reverse)
		h.writeBool(g.FrightenedInHouse)
		h.writeInt(int(g.NextDir))
		h.writeVec(g.PlannedTile)
		h.writeBool(g.HasNextDir)
		h.writeInt(g.DotCounter)
	}

	return uint64(h)
}
