package config

import "fmt"

type Vector struct {
	X, Y int
}

func (v Vector) ToTile() Vector {
	return Vector{X: v.X / Tile, Y: v.Y / Tile}
}

// ToCenterTile converts an actor's pixel position to the tile containing its center.
func (v Vector) ToCenterTile() Vector {
	return Vector{X: (v.X + Half) / Tile, Y: (v.Y + Half) / Tile}
}

func (v Vector) ToPixel() Vector {
	return Vector{X: v.X * Tile, Y: v.Y * Tile}
}

func (v Vector) Add(x, y int) Vector {
	return Vector{X: v.X + x, Y: v.Y + y}
}

func (v Vector) Sub(x, y int) Vector {
	return Vector{X: v.X - x, Y: v.Y - y}
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
	return fmt.Sprintf("(%d, %d)", v.X, v.Y)
}

func (v Vector) DistanceSq(b Vector) int {
	dx := v.X - b.X
	dy := v.Y - b.Y
	return dx*dx + dy*dy
}
