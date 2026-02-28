package main

import rl "github.com/gen2brain/raylib-go/raylib"

type Player struct {
	Actor
	requested Direction
}

func NewPlayer() *Player {
	return &Player{Actor: Actor{x: 108, y: 184, dir: Left}, requested: Left}
}

func (g *Game) drawPlayer() {
	// r l u d -- graphic
	// u d l r -- dir
	p := g.player
	var sy float32
	if g.pacman {
		sprites := []float32{992, 1008, 1024, 1040}
		sy = sprites[p.dir]
	} else {
		sprites := []float32{0, 16, 32, 48}
		sy = sprites[p.dir]
	}

	g.drawActor(float32(p.x), float32(p.y), 456, sy, p.frameIndex)

}
func (g *Game) drawPlayerLives() {
	x := float32(2) * Tile
	y := float32(0)
	for _ = range g.lives - 1 {
		if g.pacman {
			g.drawActor(x, y, 456, 992, 1)
		} else {
			g.drawActor(x, y, 456, 0, 1)
		}
		x += Sprite
	}
}

func (g *Game) canPlayerMove(tx, ty int, dir Direction) bool {
	nx := tx
	ny := ty

	switch dir {
	case Up:
		ny--
	case Down:
		ny++
	case Left:
		nx--
	case Right:
		nx++
	}

	// Tunnel wrap-around (Pacman-specific left/right tunnels)
	if nx < 0 {
		nx = GameWidth - 1
	}
	if nx >= GameWidth {
		nx = 0
	}

	if ny < 0 || ny >= GameHeight {
		return false
	}

	tile := g.maze[ny][nx]

	return tile != Wall && tile != Door
}

// updateFrame sets the frame ID for Pac-Man's 4-frame animation.
// Frame IDs: 0 (fully open), 1 (half-open), 2 (closed).
// Animation cycle: closed (2) -> half-open (1) -> fully open (0) -> half-open (1).
// Each animation state lasts 5 frames at 60 FPS (20 frames per full cycle).
func (p *Player) updateFrame() {
	framesPerState := 5
	cycleLength := 4 * framesPerState // 20 frames for full cycle
	frameInCycle := p.frameCount % cycleLength
	switch {
	case frameInCycle < framesPerState:
		p.frameIndex = 2 // Closed
	case frameInCycle < 2*framesPerState:
		p.frameIndex = 1 // Half-open
	case frameInCycle < 3*framesPerState:
		p.frameIndex = 0 // Fully open
	default:
		p.frameIndex = 1 // Half-open (returning)
	}

	p.frameCount++
	if p.frameCount > cycleLength {
		p.frameCount = 0
	}
}

func (g *Game) updatePlayer(p *Player) {
	p.updateFrame()

	requestedDir := false
	if rl.IsKeyDown(rl.KeyUp) {
		requestedDir = true
		p.requested = Up
	} else if rl.IsKeyDown(rl.KeyDown) {
		requestedDir = true
		p.requested = Down
	} else if rl.IsKeyDown(rl.KeyLeft) {
		requestedDir = true
		p.requested = Left
	} else if rl.IsKeyDown(rl.KeyRight) {
		requestedDir = true
		p.requested = Right
	}

	const threshold = 3

	tx := p.x / Tile
	ty := p.y / Tile
	ox := p.x % Tile
	oy := p.y % Tile

	// direction change
	if requestedDir || p.requested != p.dir {
		if p.dir.IsOpposite(p.requested) {
			p.dir = p.requested
		} else {
			// Perpendicular turn: check if near intersection
			nearX := ox <= threshold || ox >= Tile-threshold
			nearY := oy <= threshold || oy >= Tile-threshold

			if p.dir == Up || p.dir == Down {
				if nearY {
					targetTy := ty
					if oy > Half {
						targetTy++
					}

					if g.canPlayerMove(tx, targetTy, p.requested) {
						p.dir = p.requested
					}
				}
			} else {
				if nearX {
					targetTx := tx
					if ox > Half {
						targetTx++
					}

					if g.canPlayerMove(targetTx, ty, p.requested) {
						p.dir = p.requested
					}
				}
			}
		}
	}

	// move if not blocked
	canMove := false
	if p.dir == Up || p.dir == Down {
		if oy != 0 {
			canMove = true
		} else {
			targetTx := tx
			if ox > Half {
				targetTx++
			}
			canMove = g.canPlayerMove(targetTx, ty, p.dir)
		}
	} else {
		if ox != 0 {
			canMove = true
		} else {
			targetTy := ty
			if oy > Half {
				targetTy++
			}
			canMove = g.canPlayerMove(tx, targetTy, p.dir)
		}
	}

	if canMove {
		p.update(g.getPlayerSpeed())
	}

	// Eat dots/powerups using center-aligned tile coordinates
	ctx := (p.x + Half) / Tile
	cty := (p.y + Half) / Tile
	if ctx >= 0 && ctx < GameWidth && cty >= 0 && cty < GameHeight {
		tile := g.maze[cty][ctx]
		if tile == Dot {
			g.eventStack.Push(Event{Type: PlayerAteDot, ID: 0, X: ctx, Y: cty})
		} else if tile == Power {
			g.eventStack.Push(Event{Type: PlayerAtePowerUp, ID: 0, X: ctx, Y: cty})
		}
	}

	// tunnel warp
	if p.x < 0 {
		p.x += GameWidth * Tile
	}

	if p.x >= GameWidth*Tile {
		p.x -= GameWidth * Tile
	}

}
