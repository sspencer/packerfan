package config

//go:generate stringer -type=Direction
type Direction int

const (
	Right Direction = iota
	Left
	Up
	Down
)

const DirCount = 4

// GhostPriorityDirs defines the tie-breaking order ghosts use when choosing
// between equally distant tiles at an intersection.
var GhostPriorityDirs = [DirCount]Direction{Up, Left, Down, Right}

func (d Direction) IsOpposite(other Direction) bool {
	return (d ^ other) == 1
}

func (d Direction) Opposite() Direction {
	if d >= Right && d <= Down {
		return d ^ 1
	}
	return Up
}

// ClockwiseNext returns the next direction clockwise: Up -> Right -> Down -> Left.
func (d Direction) ClockwiseNext() Direction {
	switch d {
	case Up:
		return Right
	case Right:
		return Down
	case Down:
		return Left
	case Left:
		return Up
	default:
		return Up
	}
}
