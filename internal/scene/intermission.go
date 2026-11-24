package scene

import (
	"fmt"

	"cheddarhead/internal/audio"
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
)

const (
	intermissionAct1Frames = 366 // ~6.0 seconds at 61 FPS
	intermissionAct2Frames = 336 // ~5.5 seconds at 61 FPS
	intermissionAct3Frames = 336 // ~5.5 seconds at 61 FPS

	msClapperFrames = 90
	msAct1MeetFrame = msClapperFrames + 112 // frame when Ms. Pac-Man and Pac-Man meet in Act 1
)

// IntermissionScene plays a scripted non-gameplay cutscene between selected
// levels and transitions to ReadyScene when finished.
type IntermissionScene struct {
	ctx      *Context
	act      int
	frame    int
	duration int
}

func NewIntermissionScene(ctx *Context, act int) *IntermissionScene {
	duration := intermissionAct1Frames
	switch act {
	case 2:
		duration = intermissionAct2Frames
	case 3:
		duration = intermissionAct3Frames
	}
	return &IntermissionScene{
		ctx:      ctx,
		act:      act,
		duration: duration,
	}
}

// Act returns the 1-based intermission act number being played.
func (s *IntermissionScene) Act() int {
	return s.act
}

func (s *IntermissionScene) Enter() {
	s.frame = 0
	s.ctx.audio().SetLoop(audio.LoopNone)
	s.ctx.audio().Play(audio.IntermissionSound(s.act))
}

func (s *IntermissionScene) Update() {
	if s.ctx.Input != nil {
		s.ctx.Input.Sample()
		s.ctx.Input.ApplyPause(s.ctx.World)
	}

	if s.ctx.syncAudioPause() {
		return
	}

	s.frame++
	gs := component.MustFindGameState(s.ctx.World)
	if !gs.PacmanMode && s.act == 1 && s.frame == msAct1MeetFrame {
		s.ctx.audio().Play(audio.SoundIntermission1Bump)
	}
	if s.frame >= s.duration {
		s.ctx.Manager.Transition(NewReadyScene(s.ctx))
	}
}

func (s *IntermissionScene) Draw() {
	if s.ctx.Drawer == nil {
		return
	}
	gameState := component.MustFindGameState(s.ctx.World)
	s.ctx.Drawer.DrawIntermission(s.ctx.World, s.buildView(gameState.PacmanMode))
}

func (s *IntermissionScene) Exit() {}

func (s *IntermissionScene) buildView(pacmanMode bool) IntermissionView {
	if pacmanMode {
		return s.buildPacmanView()
	}
	return s.buildMsPacmanView()
}

func (s *IntermissionScene) buildPacmanView() IntermissionView {
	const (
		laneY     = 128
		mazeRight = config.GameWidth * config.Tile
	)
	mouthFrame := []int{2, 1, 0, 1}[(s.frame/4)%4]
	ghostFrame := (s.frame / 8) % 2

	switch s.act {
	case 1:
		// Act 1: Blinky chases Pac-Man right-to-left, then Giant Pac-Man chases frightened Blinky back.
		if s.frame < 180 {
			pacX := mazeRight + 8 - (s.frame * 3 / 2)
			blinkyX := pacX + 26
			return IntermissionView{
				Sprites: []IntermissionSprite{
					{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: 1008}, Size: config.Sprite, Frame: mouthFrame},
					{Pos: config.Vector{X: blinkyX, Y: laneY}, Src: config.Vector{X: 488, Y: 64}, Size: config.Sprite, Frame: ghostFrame},
				},
			}
		}
		f := s.frame - 180
		frightX := -20 + (f * 3 / 2)
		giantX := frightX - 42
		giantFrame := []int{2, 1, 0, 1}[(s.frame/5)%4]
		return IntermissionView{
			Sprites: []IntermissionSprite{
				{Pos: config.Vector{X: frightX, Y: laneY}, Src: config.Vector{X: 584, Y: 64}, Size: config.Sprite, Frame: ghostFrame},
				{Pos: config.Vector{X: giantX, Y: laneY - 8}, Src: config.Vector{X: 504, Y: 1008}, Size: 2 * config.Sprite, Frame: giantFrame},
			},
		}

	case 2:
		// Act 2: Blinky chases Pac-Man across a floor nail, snags and tears his sheet,
		// and stares down at the tear while Pac-Man escapes.
		const nailX = 112
		pacX := mazeRight + 8 - (s.frame * 3 / 2)
		rawBlinkyX := pacX + 28
		if rawBlinkyX > nailX {
			return IntermissionView{
				Sprites: []IntermissionSprite{
					{Pos: config.Vector{X: nailX, Y: laneY}, Src: config.Vector{X: 584, Y: 1088}, Size: config.Sprite, Frame: 0},
					{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: 1008}, Size: config.Sprite, Frame: mouthFrame},
					{Pos: config.Vector{X: rawBlinkyX, Y: laneY}, Src: config.Vector{X: 488, Y: 64}, Size: config.Sprite, Frame: ghostFrame},
				},
			}
		}

		// Snagged on the nail: stretch sheet, rip free, then stare at torn sheet.
		snagElapsed := nailX - rawBlinkyX
		sprites := []IntermissionSprite{
			{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: 1008}, Size: config.Sprite, Frame: mouthFrame},
		}
		switch {
		case snagElapsed < 24:
			// Sheet stretching on the nail.
			sprites = append(sprites,
				IntermissionSprite{Pos: config.Vector{X: nailX, Y: laneY}, Src: config.Vector{X: 600, Y: 1088}, Size: config.Sprite, Frame: 0},
				IntermissionSprite{Pos: config.Vector{X: nailX - 4, Y: laneY}, Src: config.Vector{X: 488, Y: 64}, Size: config.Sprite, Frame: ghostFrame},
			)
		case snagElapsed < 56:
			// Sheet stretched further before tearing.
			sprites = append(sprites,
				IntermissionSprite{Pos: config.Vector{X: nailX, Y: laneY}, Src: config.Vector{X: 616, Y: 1088}, Size: config.Sprite, Frame: 0},
				IntermissionSprite{Pos: config.Vector{X: nailX - 10, Y: laneY}, Src: config.Vector{X: 584, Y: 1104}, Size: config.Sprite, Frame: 0},
			)
		default:
			// Sheet ripped; Blinky stops and looks at his torn sheet.
			lookFrame := 0
			if (s.frame/24)%2 == 1 {
				lookFrame = 1
			}
			sprites = append(sprites,
				IntermissionSprite{Pos: config.Vector{X: nailX, Y: laneY}, Src: config.Vector{X: 632, Y: 1088}, Size: config.Sprite, Frame: 0},
				IntermissionSprite{Pos: config.Vector{X: nailX - 16, Y: laneY}, Src: config.Vector{X: 584, Y: 1104}, Size: config.Sprite, Frame: lookFrame},
			)
		}
		return IntermissionView{Sprites: sprites}

	default:
		// Act 3: Patched Blinky chases Pac-Man right-to-left, then sheetless Blinky
		// sprints back left-to-right dragging his red sheet behind him.
		if s.frame < 168 {
			pacX := mazeRight + 8 - (s.frame * 3 / 2)
			blinkyX := pacX + 26
			return IntermissionView{
				Sprites: []IntermissionSprite{
					{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: 1008}, Size: config.Sprite, Frame: mouthFrame},
					{Pos: config.Vector{X: blinkyX, Y: laneY}, Src: config.Vector{X: 616, Y: 1104}, Size: config.Sprite, Frame: ghostFrame},
				},
			}
		}
		f := s.frame - 168
		blinkyX := -16 + (f * 3 / 2)
		sheetSrcX := 584 + ghostFrame*16
		bareSrcX := 616 + ghostFrame*16
		return IntermissionView{
			Sprites: []IntermissionSprite{
				{Pos: config.Vector{X: blinkyX - config.Sprite, Y: laneY}, Src: config.Vector{X: sheetSrcX, Y: 1136}, Size: config.Sprite, Frame: 0},
				{Pos: config.Vector{X: blinkyX, Y: laneY}, Src: config.Vector{X: bareSrcX, Y: 1136}, Size: config.Sprite, Frame: 0},
			},
		}
	}
}

func (s *IntermissionScene) buildMsPacmanView() IntermissionView {
	const (
		laneY     = 136
		mazeRight = config.GameWidth * config.Tile
	)
	subtitles := map[int]string{
		1: "THEY MEET",
		2: "THE CHASE",
		3: "JUNIOR",
	}
	subtitle := subtitles[s.act]
	title := fmt.Sprintf("ACT %d", s.act)

	// Opening clapperboard slate.
	if s.frame < msClapperFrames {
		clapFrame := 0
		if s.frame >= 30 && s.frame < 45 {
			clapFrame = 1
		} else if s.frame >= 45 {
			clapFrame = 2
		}
		return IntermissionView{
			Title:    title,
			Subtitle: subtitle,
			Sprites: []IntermissionSprite{
				{Pos: config.Vector{X: 56, Y: 104}, Src: config.Vector{X: 456, Y: 208}, Size: 2 * config.Sprite, Frame: clapFrame},
			},
		}
	}

	f := s.frame - msClapperFrames
	mouthFrame := []int{2, 1, 0, 1}[(f/4)%4]

	switch s.act {
	case 1:
		// Ms. Pac-Man and Pac-Man enter from opposite sides, meet in the middle, and a heart appears.
		msX := min(96, -16+f)
		pacX := max(112, mazeRight-f)
		sprites := []IntermissionSprite{
			{Pos: config.Vector{X: msX, Y: laneY}, Src: config.Vector{X: 456, Y: 0}, Size: config.Sprite, Frame: mouthFrame},
			{Pos: config.Vector{X: pacX, Y: laneY}, Src: config.Vector{X: 456, Y: 160}, Size: config.Sprite, Frame: mouthFrame},
		}
		if msX == 96 && pacX == 112 {
			sprites[0].Frame = 1
			sprites[1].Frame = 1
			sprites = append(sprites, IntermissionSprite{
				Pos:   config.Vector{X: 104, Y: laneY - 20},
				Src:   config.Vector{X: 488, Y: 160},
				Size:  config.Sprite,
				Frame: 0,
			})
		}
		return IntermissionView{Sprites: sprites}

	case 2:
		// Pac-Man and Ms. Pac-Man race across the screen and back.
		if f < 123 {
			leadX := -16 + f*2
			followX := leadX - 28
			return IntermissionView{
				Sprites: []IntermissionSprite{
					{Pos: config.Vector{X: leadX, Y: laneY}, Src: config.Vector{X: 456, Y: 144}, Size: config.Sprite, Frame: mouthFrame},
					{Pos: config.Vector{X: followX, Y: laneY}, Src: config.Vector{X: 456, Y: 0}, Size: config.Sprite, Frame: mouthFrame},
				},
			}
		}
		f2 := f - 123
		leadX := mazeRight + 16 - f2*2
		followX := leadX + 28
		return IntermissionView{
			Sprites: []IntermissionSprite{
				{Pos: config.Vector{X: leadX, Y: laneY}, Src: config.Vector{X: 456, Y: 16}, Size: config.Sprite, Frame: mouthFrame},
				{Pos: config.Vector{X: followX, Y: laneY}, Src: config.Vector{X: 456, Y: 160}, Size: config.Sprite, Frame: mouthFrame},
			},
		}

	default:
		// Act 3 ("JUNIOR"): Pac-Man and Ms. Pac-Man stand together while the stork
		// flies across overhead, drops the blue bundle, and Junior appears.
		const (
			storkY   = laneY - 56
			dropX    = 104
			groundY  = laneY
			dropRate = 2
		)
		storkX := mazeRight + 8 - f
		storkFrame := (f / 8) % 2
		storkLeftSrcX := 488 + storkFrame*32
		storkRightSrcX := storkLeftSrcX + config.Sprite

		sprites := []IntermissionSprite{
			{Pos: config.Vector{X: 80, Y: groundY}, Src: config.Vector{X: 456, Y: 0}, Size: config.Sprite, Frame: 1},
			{Pos: config.Vector{X: 128, Y: groundY}, Src: config.Vector{X: 456, Y: 160}, Size: config.Sprite, Frame: 1},
			{Pos: config.Vector{X: storkX, Y: storkY}, Src: config.Vector{X: storkLeftSrcX, Y: 176}, Size: config.Sprite, Frame: 0},
			{Pos: config.Vector{X: storkX + config.Sprite, Y: storkY}, Src: config.Vector{X: storkRightSrcX, Y: 176}, Size: config.Sprite, Frame: 0},
		}

		if storkX > dropX {
			// Stork carries the blue bundle beneath its beak.
			sprites = append(sprites, IntermissionSprite{
				Pos:   config.Vector{X: storkX - 4, Y: storkY + 12},
				Src:   config.Vector{X: 488, Y: 192},
				Size:  config.Sprite,
				Frame: 0,
			})
		} else {
			dropFrames := dropX - storkX
			bundleY := min(groundY, storkY+12+dropFrames*dropRate)
			if bundleY < groundY {
				// Blue bundle falling toward the ground.
				sprites = append(sprites, IntermissionSprite{
					Pos:   config.Vector{X: dropX, Y: bundleY},
					Src:   config.Vector{X: 488, Y: 192},
					Size:  config.Sprite,
					Frame: 0,
				})
			} else {
				// Bundle landed: Junior appears between Ms. Pac-Man and Pac-Man.
				sprites = append(sprites, IntermissionSprite{
					Pos:   config.Vector{X: dropX, Y: groundY},
					Src:   config.Vector{X: 504, Y: 192},
					Size:  config.Sprite,
					Frame: 0,
				})
			}
		}
		return IntermissionView{Sprites: sprites}
	}
}
