package main

//go:generate stringer -type=Direction
type Direction int

const (
	Right Direction = iota
	Left
	Up
	Down
)

func (d Direction) IsOpposite(other Direction) bool {
	return (d == Right && other == Left) ||
		(d == Left && other == Right) ||
		(d == Up && other == Down) ||
		(d == Down && other == Up)
}

func (d Direction) Opposite() Direction {
	switch d {
	case Right:
		return Left
	case Left:
		return Right
	case Up:
		return Down
	default:
		return Up
	}
}
