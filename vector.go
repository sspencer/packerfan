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

func (v Vector) String() string {
	return fmt.Sprintf("(%d, %d)", v.x, v.y)
}
