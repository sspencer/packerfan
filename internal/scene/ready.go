package scene

import "cheddarhead/internal/audio"

// ReadyScene displays the maze and "READY!" prompt while holding gameplay until
// the pre-round delay elapses.
//   - NewReadyScene: single-stage "READY!" with actors visible (same-player respawn / next level).
//   - NewPlayerTurnReadyScene: single-stage "PLAYER ONE"/"PLAYER TWO" + "READY!" with actors
//     visible and no start jingle (2-player turn switches).
//   - NewGameStartReadyScene: two-stage arcade intro — Stage 1 plays SoundStart with
//     "PLAYER ONE" + "READY!" and actors hidden, then Stage 2 reveals actors with "READY!".
type ReadyScene struct {
	ctx              *Context
	playStartIntro   bool
	showPlayerBanner bool
	introRemaining   int
	remaining        int
}

// NewReadyScene creates a single-stage "READY!" scene (used for respawns and
// subsequent levels) with actors visible immediately.
func NewReadyScene(ctx *Context) *ReadyScene {
	return &ReadyScene{ctx: ctx}
}

// NewPlayerTurnReadyScene creates a single-stage ready screen showing the active
// player banner ("PLAYER ONE" / "PLAYER TWO") and "READY!" with actors visible
// and without replaying the game-start jingle (used on 2-player turn switches).
func NewPlayerTurnReadyScene(ctx *Context) *ReadyScene {
	return &ReadyScene{
		ctx:              ctx,
		showPlayerBanner: true,
	}
}

// NewGameStartReadyScene creates the two-stage arcade game-start sequence:
// Stage 1 shows "PLAYER ONE" + "READY!" with actors hidden and plays SoundStart;
// Stage 2 reveals actors with "READY!" before entering PlayingScene.
func NewGameStartReadyScene(ctx *Context) *ReadyScene {
	return &ReadyScene{
		ctx:            ctx,
		playStartIntro: true,
	}
}

func (s *ReadyScene) Enter() {
	s.ctx.audio().SetLoop(audio.LoopNone)
	if s.playStartIntro {
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

	if s.ctx.syncAudioPause() {
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
	s.ctx.Drawer.DrawWorld(s.ctx.World, DrawOptions{
		ShowPlayerOne: s.showPlayerBanner,
		ShowReady:     true,
	})
}

func (s *ReadyScene) Exit() {}
