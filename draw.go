package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func drawGame() {
	zeroV := rl.Vector2{}
	halfV := rl.Vector2{X: Half, Y: Half}
	zoom := game.zoom
	// Camera params (offset, target, rotation, zoom)
	mazeCamera := rl.NewCamera2D(rl.Vector2{Y: TopOffset * Tile * zoom}, zeroV, 0, zoom)
	rl.BeginMode2D(mazeCamera)
	maze.draw()
	rl.EndMode2D()

	textCamera := rl.NewCamera2D(rl.Vector2{X: 0, Y: 2 * zoom}, zeroV, 0, zoom)
	rl.BeginMode2D(textCamera)
	drawText("1UP", 3, 0, rl.White)
	drawText("HIGH SCORE", 9, 0, rl.White)
	rl.EndMode2D()

	textCamera.Offset.Y = 3 * game.zoom
	rl.BeginMode2D(textCamera)
	drawText(fmt.Sprintf("%7d", game.score), 0, 1, rl.White)
	drawText(fmt.Sprintf("%7d", game.highScore), 9, 1, rl.White)
	rl.EndMode2D()

	textCamera.Offset.Y = ((GameHeight+3)*Tile)*zoom + 2
	rl.BeginMode2D(textCamera)
	player.drawLives()
	rl.EndMode2D()

	spriteCamera := rl.NewCamera2D(rl.Vector2{Y: TopOffset * Tile * zoom}, halfV, 0, zoom)
	rl.BeginMode2D(spriteCamera)
	rl.BeginShaderMode(art.shader)
	player.draw()
	blinky.draw()
	inky.draw()
	pinky.draw()
	clyde.draw()
	rl.EndShaderMode()
	rl.EndMode2D()
}

func drawSprite(pos, sprite Vector, frameIndex int) {
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

	rl.DrawTexturePro(art.texture, src, dst, rl.Vector2{}, 0, rl.White)
}
