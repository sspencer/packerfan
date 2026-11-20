package system

import (
	"fmt"
	"math"
	"slices"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

func computeChaseTarget(
	ghost *component.GhostData,
	ghostTransform *component.TransformData,
	playerTransform *component.TransformData,
	blinkyTransform *component.TransformData,
) config.Vector {
	switch ghost.AIKind {
	case component.BlinkyAI:
		return playerTransform.Pos

	case component.PinkyAI:
		vx, vy := 0, 0
		// Reproduce original Pinky chase bug when Pac-Man faces Up.
		switch playerTransform.Dir {
		case config.Up:
			vx, vy = -4, -4
		case config.Down:
			vy = 4
		case config.Right:
			vx = 4
		default: // Left
			vx = -4
		}
		playerTile := playerTransform.Pos.ToTile()
		return config.Vector{X: playerTile.X + vx, Y: playerTile.Y + vy}.ToPixel()

	case component.InkyAI:
		vx, vy := 0, 0
		// Reproduce original Inky chase bug when Pac-Man faces Up.
		switch playerTransform.Dir {
		case config.Up:
			vx, vy = -2, -2
		case config.Down:
			vy = 2
		case config.Right:
			vx = 2
		default: // Left
			vx = -2
		}
		playerTile := playerTransform.Pos.ToTile()
		pivot := config.Vector{X: playerTile.X + vx, Y: playerTile.Y + vy}
		if blinkyTransform == nil {
			return pivot.ToPixel()
		}
		blinkyTile := blinkyTransform.Pos.ToTile()
		dx := pivot.X - blinkyTile.X
		dy := pivot.Y - blinkyTile.Y
		return config.Vector{X: blinkyTile.X + 2*dx, Y: blinkyTile.Y + 2*dy}.ToPixel()

	case component.ClydeAI:
		pv := playerTransform.Pos.ToTile()
		sv := ghostTransform.Pos.ToTile()
		if pv.DistanceSq(sv) >= 64 {
			return pv.ToPixel()
		}
		return ghost.ScatterTarget

	default:
		return config.Vector{}
	}
}

func shouldLeaveHouse(ghost *component.GhostData, gameState *component.GameStateData) bool {
	switch ghost.AIKind {
	case component.BlinkyAI, component.PinkyAI:
		return true
	case component.InkyAI:
		if gameState.Level == 1 {
			return gameState.DotsEaten > 30 || gameState.FrameCount > config.SecondsToFrames(7)
		}
		return true
	case component.ClydeAI:
		if gameState.Level == 1 {
			return gameState.DotsEaten > 90 || gameState.FrameCount > config.SecondsToFrames(15)
		}
		if gameState.Level == 2 {
			return gameState.DotsEaten > 50 || gameState.FrameCount > config.SecondsToFrames(10)
		}
		return true
	default:
		return false
	}
}

func isValidGhostTile(maze *component.MazeData, state config.GhostState, t config.Vector) bool {
	if t.X < 0 {
		t.X += config.GameWidth
	}
	if t.X >= config.GameWidth {
		t.X -= config.GameWidth
	}
	if t.Y < 0 || t.Y >= config.GameHeight {
		return false
	}

	cell := maze.GetCell(t.X, t.Y)

	if state == config.Chase || state == config.Scatter {
		if maze.IsGhostWall(t.X, t.Y) {
			return false
		}
	}

	if cell == component.Wall {
		return false
	}

	if cell == component.Door {
		return state == config.LeavingHouse || state == config.Eaten || state == config.EnteringHouse
	}

	return true
}

func chooseGhostDirection(
	maze *component.MazeData,
	prng *component.PRNGData,
	transform *component.TransformData,
	ghost *component.GhostData,
) config.Direction {
	if ghost.Reverse {
		ghost.Reverse = false
		return transform.Dir.Opposite()
	}

	if ghost.State == config.Frightened {
		return chooseFrightenedDirection(maze, prng, transform, ghost)
	}

	bestDir := transform.Dir
	minDist := math.MaxInt
	targetTile := ghost.Target.ToTile()
	curTile := transform.Pos.ToTile()

	for _, dir := range config.GhostPriorityDirs {
		if dir == transform.Dir.Opposite() {
			continue
		}
		nextTile := curTile.AddDir(dir)
		if !isValidGhostTile(maze, ghost.State, nextTile) {
			continue
		}
		if d := nextTile.DistanceSq(targetTile); d < minDist {
			minDist = d
			bestDir = dir
		}
	}

	return bestDir
}

func chooseFrightenedDirection(
	maze *component.MazeData,
	prng *component.PRNGData,
	transform *component.TransformData,
	ghost *component.GhostData,
) config.Direction {
	tile := transform.Pos.ToTile()
	var allowedDirs []config.Direction

	if isValidGhostTile(maze, ghost.State, tile.AddDir(config.Right)) {
		allowedDirs = append(allowedDirs, config.Right)
	}
	if isValidGhostTile(maze, ghost.State, tile.AddDir(config.Left)) {
		allowedDirs = append(allowedDirs, config.Left)
	}
	if isValidGhostTile(maze, ghost.State, tile.AddDir(config.Up)) {
		allowedDirs = append(allowedDirs, config.Up)
	}
	if isValidGhostTile(maze, ghost.State, tile.AddDir(config.Down)) {
		allowedDirs = append(allowedDirs, config.Down)
	}

	preferred := prng.PreferredDirection()
	reverse := transform.Dir.Opposite()

	for range config.DirCount {
		if slices.Contains(allowedDirs, preferred) && preferred != reverse {
			return preferred
		}
		preferred = preferred.ClockwiseNext()
	}

	return transform.Dir
}

func chooseRandomDirection(prng *component.PRNGData, fallback config.Direction, allowedDirs []config.Direction) config.Direction {
	preferred := prng.PreferredDirection()
	for range config.DirCount {
		if slices.Contains(allowedDirs, preferred) {
			return preferred
		}
		preferred = preferred.ClockwiseNext()
	}
	return fallback
}

func ghostSpeed(
	maze *component.MazeData,
	gameState *component.GameStateData,
	actor *component.ActorData,
	transform *component.TransformData,
	ghost *component.GhostData,
) uint32 {
	s := gameState.Spec
	tile := transform.Pos.ToTile()
	inTunnel := false
	if !maze.InBounds(tile.X, tile.Y) {
		fmt.Printf("ghost %s is out of bounds, pos: %v\n", actor.ID.String(), transform.Pos)
	} else {
		inTunnel = maze.GetCell(tile.X, tile.Y) == component.Tunnel
	}

	if inTunnel || ghost.State == config.InHouse || ghost.State == config.LeavingHouse || ghost.State == config.EnteringHouse {
		return s.GhostTunnelSpeed
	}

	if ghost.State == config.Eaten {
		return config.Speed80
	}
	if ghost.State == config.Frightened {
		return s.GhostFrightSpeed
	}

	if actor.ID == component.Blinky {
		if gameState.DotsRemaining <= s.Elroy2DotsLeft {
			return s.Elroy2Speed
		}
		if gameState.DotsRemaining <= s.Elroy1DotsLeft {
			return s.Elroy1Speed
		}
	}

	return s.GhostSpeed
}
