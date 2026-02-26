package main

import rl "github.com/gen2brain/raylib-go/raylib"

func (g *Game) Draw() {
	zeroV := rl.Vector2{}
	halfV := rl.Vector2{X: Half, Y: Half}
	// Camera params (offset, target, rotation, zoom)
	mazeCamera := rl.NewCamera2D(rl.Vector2{Y: TopOffset * Tile * g.zoom}, zeroV, 0, g.zoom)
	rl.BeginMode2D(mazeCamera)
	g.drawMaze()
	rl.EndMode2D()

	actorCamera := rl.NewCamera2D(rl.Vector2{Y: TopOffset * Tile * g.zoom}, halfV, 0, g.zoom)
	rl.BeginMode2D(actorCamera)
	rl.BeginShaderMode(g.shader)
	g.drawPlayer()
	rl.EndShaderMode()
	rl.EndMode2D()
}
