package scene

import (
	"pacfan/internal/audio"
	"pacfan/internal/system"
)

// GameOverScene freezes gameplay, displays the GAME OVER banner over the maze,
// and after at least one second allows any key press to restart from Level 1.
type GameOverScene struct {
	ctx          *Context
	elapsed      int
	restartDelay int
}

func NewGameOverScene(ctx *Context) *GameOverScene {
	return &GameOverScene{ctx: ctx}
}

func (s *GameOverScene) Enter() {
	s.ctx.audio().SetLoop(audio.LoopNone)
	s.elapsed = 0
	s.restartDelay = s.ctx.gameOverRestartDelay()
}

func (s *GameOverScene) Update() {
	anyKey := false
	start1P := false
	start2P := false
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		anyKey = s.ctx.Input.AnyKeyPressed()
		start1P = s.ctx.Input.Start1PPressed()
		start2P = s.ctx.Input.Start2PPressed()
	}

	s.elapsed++
	if s.elapsed < s.restartDelay {
		return
	}

	switch {
	case start1P:
		system.StartNewGame(s.ctx.World, 1, s.ctx.IsPixelLit)
		s.ctx.Manager.Transition(NewGameStartReadyScene(s.ctx))
	case start2P:
		system.StartNewGame(s.ctx.World, 2, s.ctx.IsPixelLit)
		s.ctx.Manager.Transition(NewGameStartReadyScene(s.ctx))
	case anyKey:
		system.RestartGame(s.ctx.World, s.ctx.IsPixelLit)
		s.ctx.Manager.Transition(NewGameStartReadyScene(s.ctx))
	}
}

func (s *GameOverScene) Draw() {
	if s.ctx.Drawer != nil {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
			ShowGameOver: true,
			HidePlayer:   true,
			HideGhosts:   true,
		})
	}
}

func (s *GameOverScene) Exit() {}
