package main

import "fmt"

type Vector struct {
	x, y int
}

func (v Vector) ToTile() Vector {
	return Vector{v.x / Tile, v.y / Tile}
}

func (v Vector) ToPixel() Vector {
	return Vector{v.x * Tile, v.y * Tile}
}

func (v Vector) Add(x, y int) Vector {
	return Vector{v.x + x, v.y + y}
}

func (v Vector) Sub(x, y int) Vector {
	return Vector{v.x - x, v.y - y}
}

func (v Vector) AddDir(dir Direction) Vector {
	switch dir {
	case Up:
		return v.Add(0, -1)
	case Down:
		return v.Add(0, 1)
	case Left:

		return v.Add(-1, 0)
	default:
		return v.Add(1, 0)
	}
}

func (v Vector) String() string {
	return fmt.Sprintf("(%d, %d)", v.x, v.y)
}

func (v Vector) DistanceSq(b Vector) int {
	dx := v.x - b.x
	dy := v.y - b.y
	return dx*dx + dy*dy
}
