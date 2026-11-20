package scene

// GameOverScene freezes gameplay permanently and displays the GAME OVER banner
// over the maze while keeping the HUD visible.
type GameOverScene struct {
	ctx *Context
}

func NewGameOverScene(ctx *Context) *GameOverScene {
	return &GameOverScene{ctx: ctx}
}

func (s *GameOverScene) Enter()  {}
func (s *GameOverScene) Update() {}

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
