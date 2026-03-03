package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func (g *Game) Draw() {
	zeroV := rl.Vector2{}
	halfV := rl.Vector2{X: Half, Y: Half}

	// Camera params (offset, target, rotation, zoom)
	mazeCamera := rl.NewCamera2D(rl.Vector2{Y: TopOffset * Tile * g.zoom}, zeroV, 0, g.zoom)
	rl.BeginMode2D(mazeCamera)
	g.drawMaze()
	rl.EndMode2D()

	textCamera := rl.NewCamera2D(rl.Vector2{X: 0, Y: 2 * g.zoom}, zeroV, 0, g.zoom)
	rl.BeginMode2D(textCamera)
	g.drawText("1UP", 3, 0, rl.White)
	g.drawText("HIGH SCORE", 9, 0, rl.White)
	rl.EndMode2D()

	textCamera.Offset.Y = 3 * g.zoom
	rl.BeginMode2D(textCamera)
	g.drawText(fmt.Sprintf("%7d", g.score), 0, 1, rl.White)
	g.drawText(fmt.Sprintf("%7d", g.highScore), 9, 1, rl.White)
	rl.EndMode2D()

	textCamera.Offset.Y = ((GameHeight+3)*Tile)*g.zoom + 2
	rl.BeginMode2D(textCamera)
	g.drawPlayerLives()
	rl.EndMode2D()

	spriteCamera := rl.NewCamera2D(rl.Vector2{Y: TopOffset * Tile * g.zoom}, halfV, 0, g.zoom)
	rl.BeginMode2D(spriteCamera)
	rl.BeginShaderMode(g.shader)
	g.drawPlayer()
	g.drawGhost(g.blinky)
	g.drawGhost(g.inky)
	g.drawGhost(g.pinky)
	g.drawGhost(g.clyde)
	rl.EndShaderMode()
	rl.EndMode2D()
}

func (g *Game) drawSprite(pos, sprite Vector, frameIndex int) {
	src := rl.Rectangle{
		X:      float32(sprite.x) + float32(frameIndex)*Sprite,
		Y:      float32(sprite.y),
		Width:  Sprite,
		Height: Sprite,
	}

	dst := rl.Rectangle{
		X:      float32(pos.x),
		Y:      float32(pos.y),
		Width:  Sprite,
		Height: Sprite,
	}

	rl.DrawTexturePro(g.texture, src, dst, rl.Vector2{}, 0, rl.White)
}
