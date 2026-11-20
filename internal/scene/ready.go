package scene

import "pacfan/internal/component"

// ReadyScene displays the maze, reset actors, and "READY!" prompt while
// holding gameplay until the start delay elapses.
type ReadyScene struct {
	ctx       *Context
	remaining int
}

func NewReadyScene(ctx *Context) *ReadyScene {
	return &ReadyScene{ctx: ctx}
}

func (s *ReadyScene) Enter() {
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

	if s.ctx.Input != nil {
		s.ctx.Input.ApplyPlayerInput(s.ctx.World)
	}

	s.remaining--
	if s.remaining <= 0 {
		s.ctx.Manager.Transition(NewPlayingScene(s.ctx))
	}
}

func (s *ReadyScene) Draw() {
	if s.ctx.Drawer != nil {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{ShowReady: true})
	}
}

func (s *ReadyScene) Exit() {}
