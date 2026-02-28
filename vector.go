package main

type Vector struct {
	x, y int
}

func (v Vector) ToTile() Vector {
	return Vector{v.x / Tile, v.y / Tile}
}

func (v Vector) ToPixel() Vector {
	return Vector{v.x * Tile, v.y * Tile}
}
