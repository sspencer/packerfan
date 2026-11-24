package scene

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/audio"
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
	"cheddarhead/internal/system"
)

// DrawOptions controls presentation overlays and visibility flags passed from
// the active scene to the renderer.
type DrawOptions struct {
	ShowPlayerOne    bool
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

// AttractCastRow describes one ghost entry on the arcade attract roll-call screen.
type AttractCastRow struct {
	SpriteRow int
	Label     string
	Nickname  string
	ColorID   int // 0=Red, 1=Pink, 2=Cyan, 3=Orange
}

// AttractView describes the visual state of one attract/title screen frame.
type AttractView struct {
	Frame          int
	SelectedPlayer int // 1 or 2
	ShowLegend     bool
	ShowPellet     bool
	PelletBlink    bool
	PelletPos      config.Vector
	Cast           []AttractCastRow
	Sprites        []IntermissionSprite
	// SwitchModeLabel names the game the "3" key switches to ("MS PAC-MAN" or
	// "PAC-MAN"); empty when switching is unavailable.
	SwitchModeLabel string
}

// Drawer decouples scene presentation from Ebitengine so scenes can be tested headlessly.
type Drawer interface {
	DrawWorld(w donburi.World, opts DrawOptions)
	DrawIntermission(w donburi.World, view IntermissionView)
	DrawAttract(w donburi.World, view AttractView)
}

// Context bundles the shared ECS world, systems, renderer, and level-loading
// callbacks used by all scenes in a game session.
type Context struct {
	Manager        *Manager
	World          donburi.World
	Input          *system.InputSystem
	Pipeline       *system.Pipeline
	Drawer         Drawer
	Audio          audio.Player
	IsPixelLit     func(x, y int) bool
	DefaultPlayers int
	// SwitchMode, when non-nil, is called from the attract screen to toggle
	// between Pac-Man and Ms. Pac-Man. The owner should rebuild the session
	// after the current frame rather than inside the call.
	SwitchMode func()

	// PlayerOneDuration overrides config.ReadyDuration for the initial
	// "PLAYER ONE" stage of NewGameStartReadyScene when > 0 (useful in tests).
	PlayerOneDuration int
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
	// GameOverRestartDelay overrides config.GameOverRestartDelay when > 0.
	GameOverRestartDelay int
	// GameOverDuration overrides config.GameOverDuration when > 0.
	GameOverDuration int
	// IntermissionFactory optionally overrides how intermission scenes are created.
	// Return nil to skip an intermission and proceed directly to ReadyScene.
	IntermissionFactory func(ctx *Context, act int) Scene
}

func (c *Context) audio() audio.Player {
	if c.Audio != nil {
		return c.Audio
	}
	return audio.NoopPlayer{}
}

func (c *Context) syncAudioMute() {
	if c.Input != nil && c.Input.MutePressed() {
		a := c.audio()
		a.SetMuted(!a.Muted())
	}
}

func (c *Context) syncAudioPause() bool {
	c.syncAudioMute()
	paused := component.MustFindGameState(c.World).Paused
	c.audio().SetPaused(paused)
	return paused
}

func (c *Context) playerOneDuration() int {
	if c.PlayerOneDuration > 0 {
		return c.PlayerOneDuration
	}
	return config.ReadyDuration
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

func (c *Context) gameOverRestartDelay() int {
	if c.GameOverRestartDelay > 0 {
		return c.GameOverRestartDelay
	}
	return config.GameOverRestartDelay
}

func (c *Context) gameOverDuration() int {
	if c.GameOverDuration > 0 {
		return c.GameOverDuration
	}
	return config.GameOverDuration
}

func (c *Context) newIntermissionScene(act int) Scene {
	if c.IntermissionFactory != nil {
		return c.IntermissionFactory(c, act)
	}
	return NewIntermissionScene(c, act)
}
