package main

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
