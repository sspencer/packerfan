package main

import rl "github.com/gen2brain/raylib-go/raylib"

type ActorID int

const (
	Blinky ActorID = iota
	Pinky
	Inky
	Clyde
	Pacman
)

type Actor struct {
	ID         ActorID
	StartDir   Direction
	StartPos   Vector
	pos        Vector
	dir        Direction
	speedIndex int
	frameCount int
	frameIndex int
	dotPause   int
	isEaten    bool
	isGhost    bool
}

func (a *Actor) ResetToStart() {
	a.dir = a.StartDir
	a.pos = a.StartPos
}

func bitValue(n uint32, bit int) int {
	if n&(1<<bit) == 0 {
		return 0
	}
	return 1
}

func bitIsSet(value uint32, n int) bool {
	return ((value >> n) & 1) != 0
}

func (a *Actor) update(speed uint32) bool {
	const SpeedSize = 16
	if a.dotPause > 0 {
		a.dotPause--
		return false
	}

	base := bitValue(speed, 31)
	move := bitValue(speed, (SpeedSize-1)-a.speedIndex) + base

	for range move {
		switch a.dir {
		case Up:
			a.pos.y -= 1
			break
		case Right:
			a.pos.x += 1
			break
		case Down:
			a.pos.y += 1
			break
		case Left:
			a.pos.x -= 1
			break
		default:
			break
		}

		// cornering / snapping for player
		if !a.isGhost {
			if a.dir == Up || a.dir == Down {
				targetX := (a.pos.x + Half) / Tile * Tile
				if a.pos.x < targetX {
					a.pos.x++
				} else if a.pos.x > targetX {
					a.pos.x--
				}
			} else if a.dir == Left || a.dir == Right {
				targetY := (a.pos.y + Half) / Tile * Tile
				if a.pos.y < targetY {
					a.pos.y++
				} else if a.pos.y > targetY {
					a.pos.y--
				}
			}
		}

		if move > 1 && isCentered(a.pos.x, a.pos.y) {
			return true
		}
	}

	a.speedIndex = (a.speedIndex + 1) % SpeedSize
	return move > 0
}

func isCentered(x, y int) bool {
	return (x%Tile) == 0 && (y%Tile) == 0
}

func tileToPixel(x, y int) rl.Vector2 {
	return rl.Vector2{float32(x) * Tile, float32(y) * Tile}
}
