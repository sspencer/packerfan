package system

import (
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
)

const speedMaskSize = 16

// BitValue returns 1 if bit is set in n, otherwise 0.
func BitValue(n uint32, bit int) int {
	if n&(1<<bit) == 0 {
		return 0
	}
	return 1
}

// BitIsSet reports whether bit n is set in value.
func BitIsSet(value uint32, n int) bool {
	return ((value >> n) & 1) != 0
}

// IsCentered reports whether a pixel coordinate is aligned to an 8x8 tile boundary.
func IsCentered(pos config.Vector) bool {
	return (pos.X|pos.Y)&(config.Tile-1) == 0
}

// WrapHorizontal wraps an actor's X coordinate across the left/right tunnel edges.
func WrapHorizontal(pos *config.Vector) {
	const edge = config.GameWidth * config.Tile
	if pos.X < 0 {
		pos.X += edge
	} else if pos.X >= edge {
		pos.X -= edge
	}
}

// MoveActor advances an actor's pixel position according to the arcade 16-frame
// speed bitmask, optionally applying Pac-Man corner-snapping, and returns true
// if the actor moved at least one pixel.
func MoveActor(transform *component.TransformData, motion *component.MotionData, speed uint32) bool {
	if motion.DotPause > 0 {
		motion.DotPause--
		return false
	}

	move := int((speed>>uint(15-motion.SpeedIndex))&1) + int(speed>>31)
	motion.SpeedIndex = (motion.SpeedIndex + 1) & (speedMaskSize - 1)

	if move == 0 {
		return false
	}

	if move == 1 && !motion.CornerSnap {
		switch transform.Dir {
		case config.Up:
			transform.Pos.Y--
		case config.Right:
			transform.Pos.X++
		case config.Down:
			transform.Pos.Y++
		case config.Left:
			transform.Pos.X--
		}
		return true
	}

	for range move {
		switch transform.Dir {
		case config.Up:
			transform.Pos.Y--
		case config.Right:
			transform.Pos.X++
		case config.Down:
			transform.Pos.Y++
		case config.Left:
			transform.Pos.X--
		}

		if motion.CornerSnap {
			switch transform.Dir {
			case config.Up, config.Down:
				targetX := (transform.Pos.X + config.Half) / config.Tile * config.Tile
				if transform.Pos.X < targetX {
					transform.Pos.X++
				} else if transform.Pos.X > targetX {
					transform.Pos.X--
				}
			case config.Left, config.Right:
				targetY := (transform.Pos.Y + config.Half) / config.Tile * config.Tile
				if transform.Pos.Y < targetY {
					transform.Pos.Y++
				} else if transform.Pos.Y > targetY {
					transform.Pos.Y--
				}
			}
		}

		if move > 1 && IsCentered(transform.Pos) {
			break
		}
	}

	return true
}
