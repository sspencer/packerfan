package config

import "fmt"

type Vector struct {
	X, Y int
}

var dirOffsets = [DirCount]Vector{
	Right: {X: 1, Y: 0},
	Left:  {X: -1, Y: 0},
	Up:    {X: 0, Y: -1},
	Down:  {X: 0, Y: 1},
}

func (v Vector) ToTile() Vector {
	if (v.X | v.Y) >= 0 {
		return Vector{X: v.X >> 3, Y: v.Y >> 3}
	}
	return Vector{X: v.X / Tile, Y: v.Y / Tile}
}

// ToCenterTile converts an actor's pixel position to the tile containing its center.
func (v Vector) ToCenterTile() Vector {
	if (v.X | v.Y) >= 0 {
		return Vector{X: (v.X + Half) >> 3, Y: (v.Y + Half) >> 3}
	}
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
	if uint(dir) < DirCount {
		o := dirOffsets[dir]
		return Vector{X: v.X + o.X, Y: v.Y + o.Y}
	}
	return Vector{X: v.X + 1, Y: v.Y}
}

func (v Vector) String() string {
	return fmt.Sprintf("(%d, %d)", v.X, v.Y)
}

func (v Vector) DistanceSq(b Vector) int {
	dx := v.X - b.X
	dy := v.Y - b.Y
	return dx*dx + dy*dy
}
