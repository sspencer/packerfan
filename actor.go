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
	x, y       int
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
	a.x = a.StartPos.x
	a.y = a.StartPos.y
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
			a.y -= 1
			break
		case Right:
			a.x += 1
			break
		case Down:
			a.y += 1
			break
		case Left:
			a.x -= 1
			break
		default:
			break
		}

		// cornering / snapping for player
		if !a.isGhost {
			if a.dir == Up || a.dir == Down {
				targetX := (a.x + Half) / Tile * Tile
				if a.x < targetX {
					a.x++
				} else if a.x > targetX {
					a.x--
				}
			} else if a.dir == Left || a.dir == Right {
				targetY := (a.y + Half) / Tile * Tile
				if a.y < targetY {
					a.y++
				} else if a.y > targetY {
					a.y--
				}
			}
		}

		if move > 1 && isCentered(a.x, a.y) {
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
