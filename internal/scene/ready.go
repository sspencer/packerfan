package scene

import (
	"pacfan/internal/audio"
	"pacfan/internal/component"
)

// ReadyScene displays the maze and "READY!" prompt while holding gameplay until
// the pre-round delay elapses. On a fresh game start (NewGameStartReadyScene),
// it first runs an intro stage showing "PLAYER ONE" + "READY!" with actors hidden
// before revealing actors for the second "READY!" beat.
type ReadyScene struct {
	ctx            *Context
	showPlayerOne  bool
	introRemaining int
	remaining      int
}

// NewReadyScene creates a single-stage "READY!" scene (used for respawns and
// subsequent levels) with actors visible immediately.
func NewReadyScene(ctx *Context) *ReadyScene {
	return &ReadyScene{ctx: ctx}
}

// NewGameStartReadyScene creates the two-stage arcade game-start sequence:
// Stage 1 shows "PLAYER ONE" + "READY!" with actors hidden; Stage 2 reveals
// actors with "READY!" before entering PlayingScene.
func NewGameStartReadyScene(ctx *Context) *ReadyScene {
	return &ReadyScene{
		ctx:           ctx,
		showPlayerOne: true,
	}
}

func (s *ReadyScene) Enter() {
	s.ctx.audio().SetLoop(audio.LoopNone)
	if s.showPlayerOne {
		s.introRemaining = s.ctx.playerOneDuration()
		s.ctx.audio().Play(audio.SoundStart)
	} else {
		s.introRemaining = 0
	}
	s.remaining = s.ctx.readyDuration()
}

func (s *ReadyScene) Update() {
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		s.ctx.Input.ApplyPause(s.ctx.World)
	}

	if component.MustFindGameState(s.ctx.World).Paused {
		return
	}

	if s.introRemaining > 0 {
		s.introRemaining--
		return
	}

	if s.ctx.Input != nil {
		s.ctx.Input.ApplyPlayerInput(s.ctx.World)
	}

	s.remaining--
	if s.remaining <= 0 {
		s.ctx.Manager.Transition(NewPlayingScene(s.ctx))
	}
}

func (s *ReadyScene) Draw() {
	if s.ctx.Drawer == nil {
		return
	}
	if s.introRemaining > 0 {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
			ShowPlayerOne: true,
			ShowReady:     true,
			HidePlayer:    true,
			HideGhosts:    true,
		})
		return
	}
	s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{ShowReady: true})
}

func (s *ReadyScene) Exit() {}
