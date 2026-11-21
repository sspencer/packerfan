package scene

import (
	"pacfan/internal/audio"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/system"
)

const attractLoopFrames = 720 // ~12 seconds per full cast + demo chase cycle

// AttractScene displays the arcade title / character roll-call, scoring legend,
// looping demo chase animation, and 1P/2P player selection prompt.
// Pressing "1" starts a 1-player game, "2" starts a 2-player game, and Enter or
// Space starts using the currently selected player count (defaulting to Context.DefaultPlayers).
// Pressing "3" switches between Pac-Man and Ms. Pac-Man when Context.SwitchMode is set.
type AttractScene struct {
	ctx             *Context
	frame           int
	selectedPlayers int
}

func NewAttractScene(ctx *Context) *AttractScene {
	players := ctx.DefaultPlayers
	if players < 1 || players > 2 {
		players = 1
	}
	return &AttractScene{
		ctx:             ctx,
		selectedPlayers: players,
	}
}

func (s *AttractScene) Enter() {
	s.frame = 0
	s.ctx.audio().StopAll()
}

func (s *AttractScene) Update() {
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		s.ctx.syncAudioMute()
		switch {
		case s.ctx.Input.Start1PPressed():
			s.startGame(1)
			return
		case s.ctx.Input.Start2PPressed():
			s.startGame(2)
			return
		case s.ctx.Input.ConfirmPressed() || s.ctx.Input.PausePressed():
			s.startGame(s.selectedPlayers)
			return
		case s.ctx.Input.SwitchModePressed() && s.ctx.SwitchMode != nil:
			s.ctx.SwitchMode()
			return
		}
	}

	s.frame = (s.frame + 1) % attractLoopFrames
}

func (s *AttractScene) startGame(playerCount int) {
	s.ctx.audio().Play(audio.SoundCredit)
	system.StartNewGame(s.ctx.World, playerCount, s.ctx.IsPixelLit)
	s.ctx.Manager.Transition(NewGameStartReadyScene(s.ctx))
}

func (s *AttractScene) Draw() {
	if s.ctx.Drawer == nil {
		return
	}
	gs := component.MustFindGameState(s.ctx.World)
	s.ctx.Drawer.DrawAttract(s.ctx.World, s.buildView(gs.PacmanMode))
}

func (s *AttractScene) Exit() {}

func (s *AttractScene) buildView(pacmanMode bool) AttractView {
	clydeNick := `"CLYDE"`
	if !pacmanMode {
		clydeNick = `"SUE"`
	}
	fullCast := []AttractCastRow{
		{SpriteRow: 64, Label: "-SHADOW", Nickname: `"BLINKY"`, ColorID: 0},
		{SpriteRow: 80, Label: "-SPEEDY", Nickname: `"PINKY"`, ColorID: 1},
		{SpriteRow: 96, Label: "-BASHFUL", Nickname: `"INKY"`, ColorID: 2},
		{SpriteRow: 112, Label: "-POKEY", Nickname: clydeNick, ColorID: 3},
	}

	// Reveal cast one row every 30 frames (~0.5s).
	visibleRows := min(len(fullCast), s.frame/30+1)
	view := AttractView{
		Frame:          s.frame,
		SelectedPlayer: s.selectedPlayers,
		ShowLegend:     s.frame >= 120,
		Cast:           fullCast[:visibleRows],
		PelletPos:      config.Vector{X: 32, Y: 148},
		PelletBlink:    (s.frame/10)%2 == 0,
	}
	if s.ctx.SwitchMode != nil {
		view.SwitchModeLabel = "MS PAC-MAN"
		if !pacmanMode {
			view.SwitchModeLabel = "PAC-MAN"
		}
	}

	if s.frame < 150 {
		return view
	}

	// Demo chase animation (frames 150..719):
	// Phase A (150..360): ghosts chase player right-to-left toward the energizer at X=32.
	// Phase B (360..719): player eats energizer and chases blue ghosts left-to-right.
	const laneY = 144
	const mazeRight = config.GameWidth * config.Tile
	mouthFrame := []int{2, 1, 0, 1}[(s.frame/4)%4]
	ghostFrame := (s.frame / 8) % 2
	pacLeftY, pacRightY := 16, 0
	if pacmanMode {
		pacLeftY, pacRightY = 1008, 992
	}

	if s.frame < 360 {
		view.ShowPellet = true
		f := s.frame - 150
		pacX := mazeRight - f
		sprites := []IntermissionSprite{
			{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: pacLeftY}, Size: config.Sprite, Frame: mouthFrame},
		}
		for i, row := range []int{64, 80, 96, 112} {
			sprites = append(sprites, IntermissionSprite{
				Pos:   config.Vector{X: pacX + 24 + i*18, Y: laneY},
				Src:   config.Vector{X: 488, Y: row},
				Size:  config.Sprite,
				Frame: ghostFrame,
			})
		}
		view.Sprites = sprites
		return view
	}

	f := s.frame - 360
	pacX := 32 + f
	sprites := []IntermissionSprite{
		{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: pacRightY}, Size: config.Sprite, Frame: mouthFrame},
	}
	for i := range 4 {
		ghostX := 64 + i*22 + (f * 3 / 4)
		if pacX >= ghostX-4 {
			// Show score sprite briefly right after Pac-Man catches each ghost.
			if pacX <= ghostX+20 {
				sprites = append(sprites, IntermissionSprite{
					Pos:   config.Vector{X: ghostX, Y: laneY},
					Src:   config.Vector{X: 456 + i*16, Y: 128},
					Size:  config.Sprite,
					Frame: 0,
				})
			}
			continue
		}
		sprites = append(sprites, IntermissionSprite{
			Pos:   config.Vector{X: ghostX, Y: laneY},
			Src:   config.Vector{X: 584, Y: 64},
			Size:  config.Sprite,
			Frame: ghostFrame,
		})
	}
	view.Sprites = sprites
	return view
}
