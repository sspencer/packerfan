package scene

import (
	"pacfan/internal/audio"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/system"
)

// PacmanDyingScene freezes normal gameplay after a ghost catches the player,
// plays the non-looping death animation, decrements lives, and routes to
// ReadyScene (if lives remain) or GameOverScene (when lives reach 0).
type PacmanDyingScene struct {
	ctx        *Context
	frame      int
	holdFrames int
	frameTicks int
	animFrames int
	total      int
}

func NewPacmanDyingScene(ctx *Context) *PacmanDyingScene {
	return &PacmanDyingScene{ctx: ctx}
}

func (s *PacmanDyingScene) Enter() {
	s.ctx.audio().SetLoop(audio.LoopNone)
	s.frame = 0
	s.holdFrames = s.ctx.playerDyingHoldFrames()
	s.frameTicks = s.ctx.playerDyingFrameTicks()
	s.animFrames = config.PlayerDyingFrameCount * s.frameTicks
	s.total = s.holdFrames + s.animFrames + s.ctx.playerDyingPostFrames()
}

func (s *PacmanDyingScene) Update() {
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		s.ctx.Input.ApplyPause(s.ctx.World)
	}

	if component.MustFindGameState(s.ctx.World).Paused {
		return
	}

	s.frame++
	if s.frame == s.holdFrames {
		s.ctx.audio().Play(audio.SoundDeath)
	}
	if s.frame < s.total {
		return
	}

	alive, switched := system.ConsumeLifeOrSwitchPlayer(s.ctx.World, s.ctx.IsPixelLit)
	if switched {
		s.ctx.Manager.Transition(NewGameStartReadyScene(s.ctx))
		return
	}
	if alive {
		s.ctx.Manager.Transition(NewReadyScene(s.ctx))
		return
	}

	s.ctx.Manager.Transition(NewGameOverScene(s.ctx))
}

func (s *PacmanDyingScene) Draw() {
	if s.ctx.Drawer == nil {
		return
	}

	// Phase 1: brief freeze showing the collision with ghosts still visible.
	if s.frame < s.holdFrames {
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{})
		return
	}

	// Phase 2: hide ghosts and play the 11-frame non-looping death animation.
	animElapsed := s.frame - s.holdFrames
	if animElapsed < s.animFrames {
		deathFrame := min(animElapsed/s.frameTicks, config.PlayerDyingFrameCount-1)
		s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
			HideGhosts:       true,
			PlayerDying:      true,
			PlayerDeathFrame: deathFrame,
		})
		return
	}

	// Phase 3: brief empty-maze beat before respawning or entering Game Over.
	s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
		HidePlayer: true,
		HideGhosts: true,
	})
}

func (s *PacmanDyingScene) Exit() {}
