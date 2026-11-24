package scene

import (
	"cheddarhead/internal/audio"
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
	"cheddarhead/internal/system"
)

// RoundClearedScene freezes the cleared maze, flashes the maze walls, advances
// the world to the next level, and routes to an optional IntermissionScene or
// directly to ReadyScene.
type RoundClearedScene struct {
	ctx            *Context
	frame          int
	completedLevel int
	holdFrames     int
	totalFrames    int
}

func NewRoundClearedScene(ctx *Context) *RoundClearedScene {
	return &RoundClearedScene{ctx: ctx}
}

func (s *RoundClearedScene) Enter() {
	s.ctx.audio().SetLoop(audio.LoopNone)
	s.frame = 0
	s.completedLevel = component.MustFindGameState(s.ctx.World).Level
	s.holdFrames = s.ctx.roundClearedHoldFrames()
	flashFrames := 2 * s.ctx.roundClearedFlashes() * config.RoundClearedFlashHalfFrames
	s.totalFrames = s.holdFrames + flashFrames
}

func (s *RoundClearedScene) Update() {
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		s.ctx.Input.ApplyPause(s.ctx.World)
	}

	if s.ctx.syncAudioPause() {
		return
	}
	gameState := component.MustFindGameState(s.ctx.World)

	s.frame++
	if s.frame < s.totalFrames {
		return
	}

	system.AdvanceToNextLevel(s.ctx.World, s.ctx.IsPixelLit)

	if act := config.IntermissionForLevel(s.completedLevel, gameState.PacmanMode); act > 0 {
		if intermission := s.ctx.newIntermissionScene(act); intermission != nil {
			s.ctx.Manager.Transition(intermission)
			return
		}
	}

	s.ctx.Manager.Transition(NewReadyScene(s.ctx))
}

func (s *RoundClearedScene) Draw() {
	if s.ctx.Drawer == nil {
		return
	}

	if s.frame < s.holdFrames {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{})
		return
	}

	flashElapsed := s.frame - s.holdFrames
	flashWhite := (flashElapsed/config.RoundClearedFlashHalfFrames)%2 == 0
	s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
		HideGhosts: true,
		FlashMaze:  flashWhite,
	})
}

func (s *RoundClearedScene) Exit() {}
