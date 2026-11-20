package render

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
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
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	zeroV := rl.Vector2{}
	halfV := rl.Vector2{X: config.Half, Y: config.Half}
	zoom := gameState.Zoom

	// Maze + edible dots/power pellets
	mazeCamera := rl.NewCamera2D(rl.Vector2{Y: config.TopOffset * config.Tile * zoom}, zeroV, 0, zoom)
	rl.BeginMode2D(mazeCamera)
	r.drawMaze(maze)
	rl.EndMode2D()

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

	// Animated sprites with chroma-key transparency shader
	spriteCamera := rl.NewCamera2D(rl.Vector2{Y: config.TopOffset * config.Tile * zoom}, halfV, 0, zoom)
	rl.BeginMode2D(spriteCamera)
	rl.BeginShaderMode(r.art.Shader)

	for entry := range r.playerQuery.Iter(w) {
		r.drawPlayer(gameState, component.Transform.Get(entry), component.Animation.Get(entry))
	}

	for entry := range r.ghostQuery.Iter(w) {
		r.drawGhost(
			gameState,
			component.Transform.Get(entry),
			component.Animation.Get(entry),
			component.Ghost.Get(entry),
		)
	}

	rl.EndShaderMode()
	rl.EndMode2D()
}

func (r *Renderer) drawMaze(maze *component.MazeData) {
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
	rl.DrawTexturePro(r.art.Texture, src, dst, rl.Vector2{}, 0, rl.White)

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

func (r *Renderer) drawGhost(
	gameState *component.GameStateData,
	transform *component.TransformData,
	anim *component.AnimationData,
	ghost *component.GhostData,
) {
	var sprite config.Vector

	switch ghost.State {
	case config.Eaten:
		sprite.X = []int{616, 584, 632, 600}[transform.Dir]
		sprite.Y = 80
	case config.Frightened:
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

	r.drawSprite(transform.Pos, sprite, anim.FrameIndex)
}

func (r *Renderer) drawSprite(pos, sprite config.Vector, frameIndex int) {
	src := rl.Rectangle{
		X:      float32(sprite.X) + float32(frameIndex)*config.Sprite,
		Y:      float32(sprite.Y),
		Width:  config.Sprite,
		Height: config.Sprite,
	}

	dst := rl.Rectangle{
		X:      float32(pos.X),
		Y:      float32(pos.Y),
		Width:  config.Sprite,
		Height: config.Sprite,
	}

	rl.DrawTexturePro(r.art.Texture, src, dst, rl.Vector2{}, 0, rl.White)
}
