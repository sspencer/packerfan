package render

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/scene"
)

var (
	colorWhite   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	colorRed     = color.RGBA{R: 230, G: 41, B: 55, A: 255}
	colorPink    = color.RGBA{R: 255, G: 109, B: 194, A: 255}
	colorSkyBlue = color.RGBA{R: 102, G: 191, B: 255, A: 255}
	colorOrange  = color.RGBA{R: 255, G: 161, B: 0, A: 255}
	colorYellow  = color.RGBA{R: 253, G: 249, B: 0, A: 255}
)

type camera2D struct {
	offsetX, offsetY float64
	targetX, targetY float64
	zoom             float64
}

func (c camera2D) geoM(x, y float64) ebiten.GeoM {
	var g ebiten.GeoM
	g.Translate(x-c.targetX, y-c.targetY)
	g.Scale(c.zoom, c.zoom)
	g.Translate(c.offsetX, c.offsetY)
	return g
}

// Renderer is the ECS rendering system that draws the maze, HUD, lives,
// player, and ghosts from the world state onto an Ebitengine target image.
type Renderer struct {
	art         *Art
	target      *ebiten.Image
	playerQuery *donburi.Query
	ghostQuery  *donburi.Query
}

func NewRenderer(art *Art) *Renderer {
	return &Renderer{
		art: art,
		playerQuery: donburi.NewQuery(
			filter.Contains(component.PlayerTag, component.Transform, component.Animation),
		),
		ghostQuery: donburi.NewQuery(
			filter.Contains(component.GhostTag, component.Transform, component.Animation, component.Ghost),
		),
	}
}

// sceneCams holds the camera setups shared by every draw entry point.
type sceneCams struct {
	zoom   float64
	text   camera2D // top HUD rows (labels at offset 2*zoom)
	maze   camera2D // maze-aligned tiles and text
	sprite camera2D // maze-aligned sprites, centered on their position
}

// sceneCameras derives the scene cameras from the game state's zoom,
// treating a non-positive zoom as 1.
func sceneCameras(gs *component.GameStateData) sceneCams {
	zoom := float64(gs.Zoom)
	if zoom <= 0 {
		zoom = 1
	}
	top := config.TopOffset * config.Tile * zoom
	return sceneCams{
		zoom:   zoom,
		text:   camera2D{offsetY: 2 * zoom, zoom: zoom},
		maze:   camera2D{offsetY: top, zoom: zoom},
		sprite: camera2D{offsetY: top, targetX: config.Half, targetY: config.Half, zoom: zoom},
	}
}

// SetTarget binds the destination image for subsequent Draw* calls.
func (r *Renderer) SetTarget(dst *ebiten.Image) {
	r.target = dst
}

func (r *Renderer) Draw(w donburi.World) {
	r.DrawWorld(w, scene.DrawOptions{})
}

// DrawWorld renders the maze, HUD, optional READY! overlay, and actors according to opts.
func (r *Renderer) DrawWorld(w donburi.World, opts scene.DrawOptions) {
	if r.target == nil {
		return
	}
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	cams := sceneCameras(gameState)

	// Maze + edible dots/power pellets
	mazeCam := cams.maze
	showPower := EnergizerVisible(gameState.FrameCount, opts.ShowReady || opts.ShowGameOver)
	r.drawMaze(mazeCam, maze, opts.FlashMaze, showPower)

	showPaused := gameState.Paused && !opts.ShowGameOver
	if opts.ShowPlayerOne || opts.ShowReady || opts.ShowGameOver || showPaused {
		if opts.ShowPlayerOne {
			drawTextWithCamera(r.target, r.art.Font, mazeCam, gameState.ActivePlayerName(), 9, 11, colorSkyBlue)
		}
		if opts.ShowReady {
			drawTextWithCamera(r.target, r.art.Font, mazeCam, "READY!", 11, 17, colorYellow)
		} else if opts.ShowGameOver {
			drawTextWithCamera(r.target, r.art.Font, mazeCam, "GAME OVER", 9, 17, colorRed)
		}
		if showPaused {
			pauseY := 17
			if opts.ShowReady {
				pauseY = 14
			}
			drawTextWithCamera(r.target, r.art.Font, mazeCam, "PAUSED", 11, pauseY, colorYellow)
		}
	}

	r.drawHUD(gameState)

	// Animated sprites
	spriteCam := cams.sprite

	showGhostScore := opts.ShowGhostScore || gameState.GhostEatenPause > 0
	if !opts.HidePlayer && !opts.HideGhosts {
		if gameState.FruitActive {
			r.drawBonusFruit(spriteCam, gameState)
		} else if gameState.FruitScoreTimer > 0 {
			r.drawBonusScore(spriteCam, gameState)
		}
	}
	if !opts.HidePlayer && !showGhostScore {
		for entry := range r.playerQuery.Iter(w) {
			if opts.PlayerDying {
				r.drawPlayerDeath(spriteCam, gameState, component.Transform.Get(entry), opts.PlayerDeathFrame)
			} else {
				r.drawPlayer(spriteCam, gameState, component.Transform.Get(entry), component.Animation.Get(entry))
			}
		}
	}

	if !opts.HideGhosts {
		for entry := range r.ghostQuery.Iter(w) {
			if showGhostScore && entry.Entity() == gameState.GhostEatenEntity {
				continue
			}
			r.drawGhost(
				spriteCam,
				gameState,
				component.Transform.Get(entry),
				component.Animation.Get(entry),
				component.Ghost.Get(entry),
			)
		}
	}

	if showGhostScore {
		r.drawGhostScore(spriteCam, gameState)
	}
}

// DrawIntermission renders the HUD and scripted intermission text and sprites.
func (r *Renderer) DrawIntermission(w donburi.World, view scene.IntermissionView) {
	if r.target == nil {
		return
	}
	gameState := component.MustFindGameState(w)

	cams := sceneCameras(gameState)

	r.drawHUD(gameState)

	if view.Title != "" || view.Subtitle != "" || gameState.Paused {
		mazeCam := cams.maze
		if view.Title != "" {
			tx := max(0, (config.GameWidth-len(view.Title))/2)
			drawTextWithCamera(r.target, r.art.Font, mazeCam, view.Title, tx, 9, colorYellow)
		}
		if view.Subtitle != "" {
			sx := max(0, (config.GameWidth-len(view.Subtitle))/2)
			drawTextWithCamera(r.target, r.art.Font, mazeCam, view.Subtitle, sx, 11, colorWhite)
		}
		if gameState.Paused {
			drawTextWithCamera(r.target, r.art.Font, mazeCam, "PAUSED", 11, 17, colorYellow)
		}
	}

	if len(view.Sprites) > 0 {
		spriteCam := cams.sprite
		for _, sp := range view.Sprites {
			size := sp.Size
			if size <= 0 {
				size = config.Sprite
			}
			r.drawSizedSprite(spriteCam, sp.Pos, sp.Src, sp.Frame, size)
		}
	}
}

// DrawAttract renders the arcade title / character roll-call, scoring legend,
// animated chase demo, and 1P/2P selection prompt.
func (r *Renderer) DrawAttract(w donburi.World, view scene.AttractView) {
	if r.target == nil {
		return
	}
	gameState := component.MustFindGameState(w)

	cams := sceneCameras(gameState)

	// Top HUD (shows both 1UP and 2UP on the attract screen).
	textCam := cams.text
	drawTextWithCamera(r.target, r.art.Font, textCam, "1UP", 3, 0, colorWhite)
	drawTextWithCamera(r.target, r.art.Font, textCam, "HIGH SCORE", 9, 0, colorWhite)
	drawTextWithCamera(r.target, r.art.Font, textCam, "2UP", 22, 0, colorWhite)

	textCam.offsetY = 3 * cams.zoom
	drawTextWithCamera(r.target, r.art.Font, textCam, fmt.Sprintf("%7d", gameState.PlayerScores[0]), 0, 1, colorWhite)
	drawTextWithCamera(r.target, r.art.Font, textCam, fmt.Sprintf("%7d", gameState.HighScore), 9, 1, colorWhite)
	drawTextWithCamera(r.target, r.art.Font, textCam, fmt.Sprintf("%7d", gameState.PlayerScores[1]), 20, 1, colorWhite)

	// Text overlay pass
	mazeCam := cams.maze

	drawTextWithCamera(r.target, r.art.Font, mazeCam, "CHARACTER / NICKNAME", 7, 2, colorWhite)
	castColors := [4]color.RGBA{colorRed, colorPink, colorSkyBlue, colorOrange}
	for i, row := range view.Cast {
		yTile := 5 + i*3
		c := castColors[row.ColorID%len(castColors)]
		drawTextWithCamera(r.target, r.art.Font, mazeCam, row.Label, 7, yTile, c)
		drawTextWithCamera(r.target, r.art.Font, mazeCam, row.Nickname, 18, yTile, c)
	}

	if view.ShowLegend {
		drawTextWithCamera(r.target, r.art.Font, mazeCam, "10 PTS", 12, 22, colorWhite)
		drawTextWithCamera(r.target, r.art.Font, mazeCam, "50 PTS", 12, 24, colorWhite)
	}

	p1Color := colorWhite
	p2Color := colorWhite
	if view.SelectedPlayer == 2 {
		p2Color = colorYellow
	} else {
		p1Color = colorYellow
	}
	promptY := 27
	if view.SwitchModeLabel != "" {
		promptY = 26
	}
	drawTextWithCamera(r.target, r.art.Font, mazeCam, "PRESS 1 - 1 PLAYER", 5, promptY, p1Color)
	drawTextWithCamera(r.target, r.art.Font, mazeCam, "PRESS 2 - 2 PLAYERS", 5, promptY+2, p2Color)
	if view.SwitchModeLabel != "" {
		drawTextWithCamera(r.target, r.art.Font, mazeCam, "PRESS 3 - "+view.SwitchModeLabel, 5, promptY+4, colorPink)
	}

	// Sprites pass (cast portraits, legend pellets, demo chase sprites)
	spriteCam := cams.sprite

	for i, row := range view.Cast {
		pos := config.Vector{X: 4 * config.Tile, Y: (5 + i*3) * config.Tile}
		r.drawSprite(spriteCam, pos, config.Vector{X: 456, Y: row.SpriteRow}, 0)
	}

	if view.ShowLegend {
		dotSub := r.subTexture(config.Tile, 125*config.Tile, config.Tile, config.Tile)
		powSub := r.subTexture(config.Tile, 127*config.Tile, config.Tile, config.Tile)
		r.drawSubImage(spriteCam, dotSub, 10*config.Tile+config.Half, 22*config.Tile+config.Half)
		r.drawSubImage(spriteCam, powSub, 10*config.Tile+config.Half, 24*config.Tile+config.Half)
	}

	if view.ShowPellet && view.PelletBlink {
		powSub := r.subTexture(config.Tile, 127*config.Tile, config.Tile, config.Tile)
		r.drawSubImage(spriteCam, powSub, float64(view.PelletPos.X), float64(view.PelletPos.Y))
	}

	for _, sp := range view.Sprites {
		size := sp.Size
		if size <= 0 {
			size = config.Sprite
		}
		r.drawSizedSprite(spriteCam, sp.Pos, sp.Src, sp.Frame, size)
	}
}

func (r *Renderer) drawHUD(gameState *component.GameStateData) {
	cams := sceneCameras(gameState)

	// Top HUD labels
	textCam := cams.text
	if OneUpVisible(gameState.FrameCount, gameState.GameOver || gameState.ActivePlayer != 0) {
		drawTextWithCamera(r.target, r.art.Font, textCam, "1UP", 3, 0, colorWhite)
	}
	drawTextWithCamera(r.target, r.art.Font, textCam, "HIGH SCORE", 9, 0, colorWhite)
	if gameState.PlayerCount == 2 {
		if OneUpVisible(gameState.FrameCount, gameState.GameOver || gameState.ActivePlayer != 1) {
			drawTextWithCamera(r.target, r.art.Font, textCam, "2UP", 22, 0, colorWhite)
		}
	}

	// Top HUD scores
	textCam.offsetY = 3 * cams.zoom
	p1Score := gameState.PlayerScores[0]
	if gameState.ActivePlayer == 0 {
		p1Score = gameState.Score
	}
	drawTextWithCamera(r.target, r.art.Font, textCam, fmt.Sprintf("%7d", p1Score), 0, 1, colorWhite)
	drawTextWithCamera(r.target, r.art.Font, textCam, fmt.Sprintf("%7d", gameState.HighScore), 9, 1, colorWhite)
	if gameState.PlayerCount == 2 {
		p2Score := gameState.PlayerScores[1]
		if gameState.ActivePlayer == 1 {
			p2Score = gameState.Score
		}
		drawTextWithCamera(r.target, r.art.Font, textCam, fmt.Sprintf("%7d", p2Score), 20, 1, colorWhite)
	}

	// Bottom HUD remaining lives and level bonus symbols
	textCam.offsetY = float64((config.GameHeight+3)*config.Tile) * cams.zoom
	r.drawLives(textCam, gameState)
	r.drawLevelSymbols(textCam, gameState)
}

func (r *Renderer) drawMaze(cam camera2D, maze *component.MazeData, flashWhite, showPower bool) {
	const mazeW = config.GameWidth * config.Tile
	const mazeH = config.GameHeight * config.Tile
	index := maze.GetIndex()
	srcX := config.GameWidth*config.Tile + config.Half
	srcY := index * config.GameHeight * config.Tile
	mazeSub := r.subTexture(srcX, srcY, mazeW, mazeH)

	if flashWhite {
		var op ebiten.DrawRectShaderOptions
		op.GeoM = cam.geoM(0, 0)
		op.Images[0] = mazeSub
		r.target.DrawRectShader(mazeW, mazeH, r.art.FlashShader, &op)
	} else {
		r.drawSubImage(cam, mazeSub, 0, 0)
	}

	dots := []int{1, 36, 63, 94, 125}
	powerUps := []int{2, 35, 65, 96, 127}

	dotSub := r.subTexture(config.Tile, dots[index]*config.Tile, config.Tile, config.Tile)
	powerSub := r.subTexture(config.Tile, powerUps[index]*config.Tile, config.Tile, config.Tile)

	for y := range config.GameHeight {
		for x := range config.GameWidth {
			tile := maze.Grid[y][x]
			switch tile {
			case component.Dot:
				r.drawSubImage(cam, dotSub, float64(x*config.Tile), float64(y*config.Tile))
			case component.Power:
				if !showPower {
					continue
				}
				r.drawSubImage(cam, powerSub, float64(x*config.Tile), float64(y*config.Tile))
			}
		}
	}
}

// EnergizerVisible reports whether power pellets should be drawn on frameCount.
// Pellets remain solid when solid is true (e.g., READY! and GAME OVER).
func EnergizerVisible(frameCount int, solid bool) bool {
	if solid {
		return true
	}
	return (frameCount/10)%2 == 0
}

// OneUpVisible reports whether the "1UP" HUD header should be drawn on frameCount.
func OneUpVisible(frameCount int, gameOver bool) bool {
	if gameOver {
		return true
	}
	return (frameCount/16)%2 == 0
}

func (r *Renderer) drawLives(cam camera2D, gameState *component.GameStateData) {
	if gameState.Lives <= 1 {
		return
	}

	pos := config.Vector{X: 2 * config.Tile, Y: 0}
	sprite := config.Vector{X: 456, Y: 0}
	if gameState.PacmanMode {
		sprite.Y = 992
	}

	for range gameState.Lives - 1 {
		r.drawSprite(cam, pos, sprite, 1)
		pos.X += config.Sprite
	}
}

// LevelBonusSymbols returns up to the 7 most recent level bonus symbols ordered
// from rightmost (oldest in the window) to leftmost (current level).
func LevelBonusSymbols(level int) []config.BonusSymbol {
	if level < 1 {
		return nil
	}
	if level > 255 {
		level = 255
	}
	start := max(1, level-6)
	symbols := make([]config.BonusSymbol, 0, level-start+1)
	for lvl := start; lvl <= level; lvl++ {
		symbols = append(symbols, config.GetLevelSpec(lvl).BonusSymbol)
	}
	return symbols
}

func (r *Renderer) drawLevelSymbols(cam camera2D, gameState *component.GameStateData) {
	pos := config.Vector{X: (config.GameWidth - 4) * config.Tile, Y: 0}
	if gameState.PacmanMode {
		for _, sym := range LevelBonusSymbols(gameState.Level) {
			if sprite, ok := BonusFruitSprite(true, sym); ok {
				r.drawSprite(cam, pos, sprite, 0)
			}
			pos.X -= config.Sprite
		}
		return
	}
	// Ms. Pac-Man HUD shows up to 7 sequential fruit symbols (Levels 1..7, capped at Banana).
	count := min(max(1, gameState.Level), 7)
	for i := range count {
		sym, _ := config.MsPacmanBonus(i+1, 0)
		if sprite, ok := BonusFruitSprite(false, sym); ok {
			r.drawSprite(cam, pos, sprite, 0)
		}
		pos.X -= config.Sprite
	}
}

func (r *Renderer) drawPlayer(
	cam camera2D,
	gameState *component.GameStateData,
	transform *component.TransformData,
	anim *component.AnimationData,
) {
	sprite := config.Vector{X: 456}
	if gameState.PacmanMode {
		sprite.Y = []int{992, 1008, 1024, 1040}[transform.Dir]
	} else {
		sprite.Y = []int{0, 16, 32, 48}[transform.Dir]
	}

	r.drawSprite(cam, transform.Pos, sprite, anim.FrameIndex)
}

// PlayerDeathSprite returns the top-left spritesheet pixel coordinates in
// assets/game.png for a non-looping player death frame (0..PlayerDyingFrameCount-1).
// Classic Pac-Man uses the 11-frame collapse+starburst row at Y=992; Ms. Pac-Man
// spins through her 4 directional sprites (Down, Left, Up, Right).
func PlayerDeathSprite(pacmanMode bool, frame int) (config.Vector, bool) {
	if frame < 0 || frame >= config.PlayerDyingFrameCount {
		return config.Vector{}, false
	}
	if pacmanMode {
		return config.Vector{X: 504 + frame*config.Sprite, Y: 992}, true
	}
	spinRows := [4]int{48, 16, 32, 0} // Down, Left, Up, Right
	return config.Vector{X: 472, Y: spinRows[frame%len(spinRows)]}, true
}

func (r *Renderer) drawPlayerDeath(
	cam camera2D,
	gameState *component.GameStateData,
	transform *component.TransformData,
	frame int,
) {
	sprite, ok := PlayerDeathSprite(gameState.PacmanMode, frame)
	if !ok {
		return
	}
	r.drawSprite(cam, transform.Pos, sprite, 0)
}

func (r *Renderer) drawGhost(
	cam camera2D,
	gameState *component.GameStateData,
	transform *component.TransformData,
	anim *component.AnimationData,
	ghost *component.GhostData,
) {
	var sprite config.Vector
	frameIndex := anim.FrameIndex

	switch {
	case ghost.State == config.Eaten || ghost.State == config.EnteringHouse:
		// Eye sprites are single-frame (16px apart: Right, Left, Up, Down).
		sprite.X = []int{584, 600, 616, 632}[transform.Dir]
		sprite.Y = 80
		frameIndex = 0
	case ghost.State == config.Frightened ||
		(ghost.FrightenedInHouse && gameState.IsFrightMode && (ghost.State == config.InHouse || ghost.State == config.LeavingHouse)):
		const blueX = 584
		const whiteX = 616

		sprite.Y = 64
		if FrightenedGhostFlashWhite(gameState.FrightFrames, gameState.FrightFlashes) {
			sprite.X = whiteX
		} else {
			sprite.X = blueX
		}
	default:
		sprite.Y = ghost.SpriteRow
		sprite.X = []int{456, 488, 520, 552}[transform.Dir]
	}

	r.drawSprite(cam, transform.Pos, sprite, frameIndex)
}

// FrightenedGhostFlashWhite reports whether a frightened ghost should render
// in its white warning sprite on the given remaining fright frame count.
// Each flash consists of a 14-frame blue half-cycle and a 14-frame white half-cycle,
// producing exactly frightFlashes white pulses as frightFrames counts down to 1.
func FrightenedGhostFlashWhite(frightFrames, frightFlashes int) bool {
	const flashHalfFrames = 14
	if frightFrames <= 0 || frightFlashes <= 0 {
		return false
	}
	if frightFrames > 2*frightFlashes*flashHalfFrames {
		return false
	}
	slot := (frightFrames - 1) / flashHalfFrames
	return slot%2 == 0
}

// GhostScoreSprite returns the top-left spritesheet pixel coordinates in
// assets/game.png for a ghost-eaten bonus score (200, 400, 800, 1600).
func GhostScoreSprite(score int) (config.Vector, bool) {
	const scoreRowY = 128
	switch score {
	case 200:
		return config.Vector{X: 456, Y: scoreRowY}, true
	case 400:
		return config.Vector{X: 472, Y: scoreRowY}, true
	case 800:
		return config.Vector{X: 488, Y: scoreRowY}, true
	case 1600:
		return config.Vector{X: 504, Y: scoreRowY}, true
	default:
		return config.Vector{}, false
	}
}

func (r *Renderer) drawGhostScore(cam camera2D, gameState *component.GameStateData) {
	sprite, ok := GhostScoreSprite(gameState.GhostEatenScore)
	if !ok {
		return
	}
	pos := config.Vector{X: gameState.GhostEatenX, Y: gameState.GhostEatenY}
	r.drawSprite(cam, pos, sprite, 0)
}

// BonusFruitSprite returns the top-left spritesheet pixel coordinates in
// assets/game.png for the given bonus fruit symbol.
func BonusFruitSprite(pacmanMode bool, symbol config.BonusSymbol) (config.Vector, bool) {
	idx := int(symbol)
	if idx < 0 || idx > int(config.BonusKey) {
		return config.Vector{}, false
	}
	if pacmanMode {
		return config.Vector{X: 504 + idx*config.Sprite, Y: 1040}, true
	}
	// Ms. Pac-Man spritesheet has 7 bonus items across Y=0 (X=504..600).
	if idx > 6 {
		idx = 6
	}
	return config.Vector{X: 504 + idx*config.Sprite, Y: 0}, true
}

// BonusScoreSprite returns the spritesheet origin and pixel width for a bonus
// fruit score popup in assets/game.png.
func BonusScoreSprite(pacmanMode bool, points int) (config.Vector, int, bool) {
	if pacmanMode {
		switch points {
		case 100:
			return config.Vector{X: 456, Y: 1136}, 16, true
		case 300:
			return config.Vector{X: 472, Y: 1136}, 16, true
		case 500:
			return config.Vector{X: 488, Y: 1136}, 16, true
		case 700:
			return config.Vector{X: 504, Y: 1136}, 16, true
		case 1000:
			return config.Vector{X: 520, Y: 1136}, 20, true
		case 2000:
			return config.Vector{X: 516, Y: 1152}, 24, true
		case 3000:
			return config.Vector{X: 516, Y: 1168}, 24, true
		case 5000:
			return config.Vector{X: 516, Y: 1184}, 24, true
		default:
			return config.Vector{}, 0, false
		}
	}
	// Ms. Pac-Man bonus score row at Y=16 (100, 200, 500, 700, 1000, 2000, 5000).
	switch points {
	case 100:
		return config.Vector{X: 504, Y: 16}, 16, true
	case 200, 300:
		return config.Vector{X: 520, Y: 16}, 16, true
	case 500:
		return config.Vector{X: 536, Y: 16}, 16, true
	case 700:
		return config.Vector{X: 552, Y: 16}, 16, true
	case 1000:
		return config.Vector{X: 568, Y: 16}, 16, true
	case 2000, 3000:
		return config.Vector{X: 584, Y: 16}, 16, true
	case 5000:
		return config.Vector{X: 600, Y: 16}, 16, true
	default:
		return config.Vector{}, 0, false
	}
}

// FruitBounceOffset returns the vertical pixel offset for Ms. Pac-Man's
// bouncing bonus fruit animation across its 16-frame bounce cycle.
func FruitBounceOffset(frame int) int {
	curve := [16]int{0, -1, -2, -2, -2, -1, 0, 0, 0, -1, -2, -2, -2, -1, 0, 0}
	return curve[frame%len(curve)]
}

func (r *Renderer) drawBonusFruit(cam camera2D, gameState *component.GameStateData) {
	sym := gameState.FruitSymbol
	if !gameState.FruitMoving && gameState.PacmanMode {
		sym = gameState.Spec.BonusSymbol
	}
	sprite, ok := BonusFruitSprite(gameState.PacmanMode, sym)
	if !ok {
		return
	}
	pos := gameState.FruitPos
	if gameState.FruitMoving {
		pos.Y += FruitBounceOffset(gameState.FruitBounceFrame)
	}
	r.drawSprite(cam, pos, sprite, 0)
}

func (r *Renderer) drawBonusScore(cam camera2D, gameState *component.GameStateData) {
	origin, width, ok := BonusScoreSprite(gameState.PacmanMode, gameState.FruitScoreValue)
	if !ok {
		return
	}
	offsetX := (width - config.Sprite) / 2
	sub := r.subTexture(origin.X, origin.Y, width, config.Sprite)
	r.drawSubImage(cam, sub, float64(gameState.FruitPos.X-offsetX), float64(gameState.FruitPos.Y))
}

func (r *Renderer) drawSprite(cam camera2D, pos, sprite config.Vector, frameIndex int) {
	r.drawSizedSprite(cam, pos, sprite, frameIndex, config.Sprite)
}

func (r *Renderer) drawSizedSprite(cam camera2D, pos, sprite config.Vector, frameIndex, size int) {
	sub := r.subTexture(sprite.X+frameIndex*size, sprite.Y, size, size)
	r.drawSubImage(cam, sub, float64(pos.X), float64(pos.Y))
}

func (r *Renderer) subTexture(x, y, w, h int) *ebiten.Image {
	return r.art.Texture.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image)
}

func (r *Renderer) drawSubImage(cam camera2D, sub *ebiten.Image, x, y float64) {
	var op ebiten.DrawImageOptions
	op.GeoM = cam.geoM(x, y)
	r.target.DrawImage(sub, &op)
}
