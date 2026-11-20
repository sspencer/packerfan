package scene

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/config"
	"pacfan/internal/system"
)

// DrawOptions controls presentation overlays and visibility flags passed from
// the active scene to the renderer.
type DrawOptions struct {
	ShowReady        bool
	ShowGameOver     bool
	HidePlayer       bool
	HideGhosts       bool
	FlashMaze        bool
	ShowGhostScore   bool
	PlayerDying      bool
	PlayerDeathFrame int
}

// IntermissionSprite describes a single spritesheet sprite drawn during an intermission.
type IntermissionSprite struct {
	Pos   config.Vector
	Src   config.Vector
	Size  int
	Frame int
}

// IntermissionView describes the visual state of one intermission frame.
type IntermissionView struct {
	Title    string
	Subtitle string
	Sprites  []IntermissionSprite
}

// Drawer decouples scene presentation from Raylib so scenes can be tested headlessly.
type Drawer interface {
	DrawWorld(w donburi.World, opts DrawOptions)
	DrawIntermission(w donburi.World, view IntermissionView)
}

// Context bundles the shared ECS world, systems, renderer, and level-loading
// callbacks used by all scenes in a game session.
type Context struct {
	Manager    *Manager
	World      donburi.World
	Input      *system.InputSystem
	Pipeline   *system.Pipeline
	Drawer     Drawer
	IsPixelLit func(x, y int) bool

	// ReadyDuration overrides config.ReadyDuration when > 0 (useful in tests).
	ReadyDuration int
	// RoundClearedHoldFrames overrides config.RoundClearedHoldFrames when > 0.
	RoundClearedHoldFrames int
	// RoundClearedFlashes overrides config.RoundClearedFlashes when > 0.
	RoundClearedFlashes int
	// PlayerDyingHoldFrames overrides config.PlayerDyingHoldFrames when > 0.
	PlayerDyingHoldFrames int
	// PlayerDyingFrameTicks overrides config.PlayerDyingFrameTicks when > 0.
	PlayerDyingFrameTicks int
	// PlayerDyingPostFrames overrides config.PlayerDyingPostFrames when > 0.
	PlayerDyingPostFrames int
	// IntermissionFactory optionally overrides how intermission scenes are created.
	// Return nil to skip an intermission and proceed directly to ReadyScene.
	IntermissionFactory func(ctx *Context, act int) Scene
}

func (c *Context) readyDuration() int {
	if c.ReadyDuration > 0 {
		return c.ReadyDuration
	}
	return config.ReadyDuration
}

func (c *Context) roundClearedHoldFrames() int {
	if c.RoundClearedHoldFrames > 0 {
		return c.RoundClearedHoldFrames
	}
	return config.RoundClearedHoldFrames
}

func (c *Context) roundClearedFlashes() int {
	if c.RoundClearedFlashes > 0 {
		return c.RoundClearedFlashes
	}
	return config.RoundClearedFlashes
}

func (c *Context) playerDyingHoldFrames() int {
	if c.PlayerDyingHoldFrames > 0 {
		return c.PlayerDyingHoldFrames
	}
	return config.PlayerDyingHoldFrames
}

func (c *Context) playerDyingFrameTicks() int {
	if c.PlayerDyingFrameTicks > 0 {
		return c.PlayerDyingFrameTicks
	}
	return config.PlayerDyingFrameTicks
}

func (c *Context) playerDyingPostFrames() int {
	if c.PlayerDyingPostFrames > 0 {
		return c.PlayerDyingPostFrames
	}
	return config.PlayerDyingPostFrames
}

func (c *Context) newIntermissionScene(act int) Scene {
	if c.IntermissionFactory != nil {
		return c.IntermissionFactory(c, act)
	}
	return NewIntermissionScene(c, act)
}
