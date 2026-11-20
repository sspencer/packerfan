package render

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/scene"
)

// Renderer is the ECS rendering system that draws the maze, HUD, lives,
// player, and ghosts from the world state using Raylib 2D cameras and shaders.
type Renderer struct {
	art         *Art
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

func (r *Renderer) Draw(w donburi.World) {
	r.DrawWorld(w, scene.DrawOptions{})
}

// DrawWorld renders the maze, HUD, optional READY! overlay, and actors according to opts.
func (r *Renderer) DrawWorld(w donburi.World, opts scene.DrawOptions) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	zeroV := rl.Vector2{}
	halfV := rl.Vector2{X: config.Half, Y: config.Half}
	zoom := gameState.Zoom

	// Maze + edible dots/power pellets
	mazeCamera := rl.NewCamera2D(rl.Vector2{Y: config.TopOffset * config.Tile * zoom}, zeroV, 0, zoom)
	rl.BeginMode2D(mazeCamera)
	r.drawMaze(maze, opts.FlashMaze)
	if opts.ShowReady {
		rl.BeginShaderMode(r.art.Shader)
		DrawText(r.art.Font, "READY!", 11, 17, rl.Yellow)
		rl.EndShaderMode()
	} else if opts.ShowGameOver {
		rl.BeginShaderMode(r.art.Shader)
		DrawText(r.art.Font, "GAME OVER", 9, 17, rl.Red)
		rl.EndShaderMode()
	}
	rl.EndMode2D()

	r.drawHUD(gameState)

	// Animated sprites with chroma-key transparency shader
	spriteCamera := rl.NewCamera2D(rl.Vector2{Y: config.TopOffset * config.Tile * zoom}, halfV, 0, zoom)
	rl.BeginMode2D(spriteCamera)
	rl.BeginShaderMode(r.art.Shader)

	showGhostScore := opts.ShowGhostScore || gameState.GhostEatenPause > 0
	if !opts.HidePlayer && !showGhostScore {
		for entry := range r.playerQuery.Iter(w) {
			if opts.PlayerDying {
				r.drawPlayerDeath(gameState, component.Transform.Get(entry), opts.PlayerDeathFrame)
			} else {
				r.drawPlayer(gameState, component.Transform.Get(entry), component.Animation.Get(entry))
			}
		}
	}

	if !opts.HideGhosts {
		for entry := range r.ghostQuery.Iter(w) {
			if showGhostScore && entry.Entity() == gameState.GhostEatenEntity {
				continue
			}
			r.drawGhost(
				gameState,
				component.Transform.Get(entry),
				component.Animation.Get(entry),
				component.Ghost.Get(entry),
			)
		}
	}

	if showGhostScore {
		r.drawGhostScore(gameState)
	}

	rl.EndShaderMode()
	rl.EndMode2D()
}

// DrawIntermission renders the HUD and scripted intermission text and sprites.
func (r *Renderer) DrawIntermission(w donburi.World, view scene.IntermissionView) {
	gameState := component.MustFindGameState(w)

	zeroV := rl.Vector2{}
	halfV := rl.Vector2{X: config.Half, Y: config.Half}
	zoom := gameState.Zoom

	r.drawHUD(gameState)

	if view.Title != "" || view.Subtitle != "" {
		mazeCamera := rl.NewCamera2D(rl.Vector2{Y: config.TopOffset * config.Tile * zoom}, zeroV, 0, zoom)
		rl.BeginMode2D(mazeCamera)
		rl.BeginShaderMode(r.art.Shader)
		if view.Title != "" {
			tx := max(0, (config.GameWidth-len(view.Title))/2)
			DrawText(r.art.Font, view.Title, tx, 9, rl.Yellow)
		}
		if view.Subtitle != "" {
			sx := max(0, (config.GameWidth-len(view.Subtitle))/2)
			DrawText(r.art.Font, view.Subtitle, sx, 11, rl.White)
		}
		rl.EndShaderMode()
		rl.EndMode2D()
	}

	if len(view.Sprites) > 0 {
		spriteCamera := rl.NewCamera2D(rl.Vector2{Y: config.TopOffset * config.Tile * zoom}, halfV, 0, zoom)
		rl.BeginMode2D(spriteCamera)
		rl.BeginShaderMode(r.art.Shader)
		for _, sp := range view.Sprites {
			size := sp.Size
			if size <= 0 {
				size = config.Sprite
			}
			r.drawSizedSprite(sp.Pos, sp.Src, sp.Frame, size)
		}
		rl.EndShaderMode()
		rl.EndMode2D()
	}
}

func (r *Renderer) drawHUD(gameState *component.GameStateData) {
	zeroV := rl.Vector2{}
	zoom := gameState.Zoom

	// Top HUD labels
	textCamera := rl.NewCamera2D(rl.Vector2{X: 0, Y: 2 * zoom}, zeroV, 0, zoom)
	rl.BeginMode2D(textCamera)
	DrawText(r.art.Font, "1UP", 3, 0, rl.White)
	DrawText(r.art.Font, "HIGH SCORE", 9, 0, rl.White)
	rl.EndMode2D()

	// Top HUD scores
	textCamera.Offset.Y = 3 * zoom
	rl.BeginMode2D(textCamera)
	DrawText(r.art.Font, fmt.Sprintf("%7d", gameState.Score), 0, 1, rl.White)
	DrawText(r.art.Font, fmt.Sprintf("%7d", gameState.HighScore), 9, 1, rl.White)
	rl.EndMode2D()

	// Bottom HUD remaining lives
	textCamera.Offset.Y = ((config.GameHeight+3)*config.Tile)*zoom + 2
	rl.BeginMode2D(textCamera)
	r.drawLives(gameState)
	rl.EndMode2D()
}

func (r *Renderer) drawMaze(maze *component.MazeData, flashWhite bool) {
	const mazeW = float32(config.GameWidth) * config.Tile
	const mazeH = float32(config.GameHeight) * config.Tile
	index := maze.GetIndex()
	src := rl.Rectangle{
		X:      config.GameWidth*config.Tile + config.Half,
		Y:      float32(index) * config.GameHeight * config.Tile,
		Width:  mazeW,
		Height: mazeH,
	}
	dst := rl.NewRectangle(0, 0, mazeW, mazeH)
	if flashWhite {
		rl.BeginShaderMode(r.art.FlashShader)
		rl.DrawTexturePro(r.art.Texture, src, dst, rl.Vector2{}, 0, rl.White)
		rl.EndShaderMode()
	} else {
		rl.DrawTexturePro(r.art.Texture, src, dst, rl.Vector2{}, 0, rl.White)
	}

	dots := []float32{1, 36, 63, 94, 125}
	powerUps := []float32{2, 35, 65, 96, 127}

	dotSrc := rl.Rectangle{X: config.Tile, Y: dots[index] * config.Tile, Width: config.Tile, Height: config.Tile}
	powerSrc := rl.Rectangle{X: config.Tile, Y: powerUps[index] * config.Tile, Width: config.Tile, Height: config.Tile}

	for y := range config.GameHeight {
		for x := range config.GameWidth {
			tile := maze.Grid[y][x]
			switch tile {
			case component.Dot:
				rec := rl.Rectangle{X: float32(x) * config.Tile, Y: float32(y) * config.Tile, Width: config.Tile, Height: config.Tile}
				rl.DrawTexturePro(r.art.Texture, dotSrc, rec, rl.Vector2{}, 0, rl.White)
			case component.Power:
				rec := rl.Rectangle{X: float32(x) * config.Tile, Y: float32(y) * config.Tile, Width: config.Tile, Height: config.Tile}
				rl.DrawTexturePro(r.art.Texture, powerSrc, rec, rl.Vector2{}, 0, rl.White)
			}
		}
	}
}

func (r *Renderer) drawLives(gameState *component.GameStateData) {
	if gameState.Lives <= 1 {
		return
	}

	pos := config.Vector{X: 2 * config.Tile, Y: 0}
	sprite := config.Vector{X: 456, Y: 0}
	if gameState.PacmanMode {
		sprite.Y = 992
	}

	for range gameState.Lives - 1 {
		r.drawSprite(pos, sprite, 1)
		pos.X += config.Sprite
	}
}

func (r *Renderer) drawPlayer(
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

	r.drawSprite(transform.Pos, sprite, anim.FrameIndex)
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
	gameState *component.GameStateData,
	transform *component.TransformData,
	frame int,
) {
	sprite, ok := PlayerDeathSprite(gameState.PacmanMode, frame)
	if !ok {
		return
	}
	r.drawSprite(transform.Pos, sprite, 0)
}

func (r *Renderer) drawGhost(
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
		const flashFrames = 14
		const blueX = 584
		const whiteX = 616

		sprite.X = whiteX
		sprite.Y = 64

		n := gameState.FrightFrames/flashFrames + 1
		if n%2 == 0 || gameState.FrightFrames >= gameState.FrightFlashes*flashFrames {
			sprite.X = blueX
		}
	default:
		sprite.Y = ghost.SpriteRow
		sprite.X = []int{456, 488, 520, 552}[transform.Dir]
	}

	r.drawSprite(transform.Pos, sprite, frameIndex)
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

func (r *Renderer) drawGhostScore(gameState *component.GameStateData) {
	sprite, ok := GhostScoreSprite(gameState.GhostEatenScore)
	if !ok {
		return
	}
	pos := config.Vector{X: gameState.GhostEatenX, Y: gameState.GhostEatenY}
	r.drawSprite(pos, sprite, 0)
}

func (r *Renderer) drawSprite(pos, sprite config.Vector, frameIndex int) {
	r.drawSizedSprite(pos, sprite, frameIndex, config.Sprite)
}

func (r *Renderer) drawSizedSprite(pos, sprite config.Vector, frameIndex, size int) {
	s := float32(size)
	src := rl.Rectangle{
		X:      float32(sprite.X) + float32(frameIndex)*s,
		Y:      float32(sprite.Y),
		Width:  s,
		Height: s,
	}

	dst := rl.Rectangle{
		X:      float32(pos.X),
		Y:      float32(pos.Y),
		Width:  s,
		Height: s,
	}

	rl.DrawTexturePro(r.art.Texture, src, dst, rl.Vector2{}, 0, rl.White)
}
