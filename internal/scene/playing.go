package scene

import (
	"pacfan/internal/audio"
	"pacfan/internal/component"
)

// PlayingScene runs the normal per-frame ECS simulation pipeline and routes
// transitions when a ghost is eaten or all dots and pellets are cleared.
type PlayingScene struct {
	ctx *Context
}

func NewPlayingScene(ctx *Context) *PlayingScene {
	return &PlayingScene{ctx: ctx}
}

func (s *PlayingScene) Enter() {}

func (s *PlayingScene) Update() {
	s.ctx.Pipeline.Update(s.ctx.World)
	if s.ctx.syncAudioPause() {
		return
	}

	gameState := component.MustFindGameState(s.ctx.World)
	if gameState.PlayerDying {
		s.ctx.Manager.Transition(NewPacmanDyingScene(s.ctx))
		return
	}
	if gameState.LevelComplete {
		s.ctx.Manager.Transition(NewRoundClearedScene(s.ctx))
		return
	}
	if gameState.GhostEatenPause > 0 {
		s.ctx.Manager.Transition(NewGhostEatenFreezeScene(s.ctx))
		return
	}

	s.ctx.audio().SetLoop(audio.SelectBackgroundLoop(s.ctx.World))
}

func (s *PlayingScene) Draw() {
	if s.ctx.Drawer != nil {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{})
	}
}

func (s *PlayingScene) Exit() {
	s.ctx.audio().SetLoop(audio.LoopNone)
}
