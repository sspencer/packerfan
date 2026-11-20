package scene

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/audio"
	"pacfan/internal/component"
)

// GhostEatenFreezeScene briefly freezes simulation after Pac-Man eats a
// frightened ghost while displaying the bonus score sprite in the maze.
type GhostEatenFreezeScene struct {
	ctx *Context
}

func NewGhostEatenFreezeScene(ctx *Context) *GhostEatenFreezeScene {
	return &GhostEatenFreezeScene{ctx: ctx}
}

func (s *GhostEatenFreezeScene) Enter() {}

func (s *GhostEatenFreezeScene) Update() {
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		s.ctx.Input.ApplyPause(s.ctx.World)
	}

	gameState := component.MustFindGameState(s.ctx.World)
	if gameState.Paused {
		s.ctx.audio().SetLoop(audio.LoopNone)
		return
	}

	if s.ctx.Input != nil {
		s.ctx.Input.ApplyPlayerInput(s.ctx.World)
	}
	if s.ctx.Pipeline != nil {
		s.ctx.Pipeline.UpdateEatenGhosts(s.ctx.World)
	}
	if audio.HasOtherRetreatingEyes(s.ctx.World) {
		s.ctx.audio().SetLoop(audio.LoopEyes)
	} else {
		s.ctx.audio().SetLoop(audio.LoopNone)
	}

	if gameState.GhostEatenPause > 0 {
		gameState.GhostEatenPause--
	}
	if gameState.GhostEatenPause <= 0 {
		gameState.GhostEatenEntity = donburi.Null
		gameState.GhostEatenScore = 0
		s.ctx.Manager.Transition(NewPlayingScene(s.ctx))
	}
}

func (s *GhostEatenFreezeScene) Draw() {
	if s.ctx.Drawer != nil {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
			HidePlayer:     true,
			ShowGhostScore: true,
		})
	}
}

func (s *GhostEatenFreezeScene) Exit() {}
