package scene

import "pacfan/internal/audio"

// GameOverScene freezes gameplay, displays the GAME OVER banner over the maze,
// and returns to AttractScene (where the player can choose 1 or 2 player mode)
// either automatically after GameOverDuration (~3s) or on any key press after
// GameOverRestartDelay (~1s).
type GameOverScene struct {
	ctx          *Context
	elapsed      int
	restartDelay int
	duration     int
}

func NewGameOverScene(ctx *Context) *GameOverScene {
	return &GameOverScene{ctx: ctx}
}

func (s *GameOverScene) Enter() {
	s.ctx.audio().SetLoop(audio.LoopNone)
	s.elapsed = 0
	s.restartDelay = s.ctx.gameOverRestartDelay()
	s.duration = s.ctx.gameOverDuration()
}

func (s *GameOverScene) Update() {
	anyKey := false
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		anyKey = s.ctx.Input.AnyKeyPressed()
	}

	s.elapsed++
	if s.elapsed >= s.duration || (s.elapsed >= s.restartDelay && anyKey) {
		s.ctx.Manager.Transition(NewAttractScene(s.ctx))
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
