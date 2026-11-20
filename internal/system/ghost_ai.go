package system

import (
	"fmt"
	"math"
	"slices"

	"github.com/yohamta/donburi"

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
		return playerTransform.Pos.ToCenterTile().ToPixel()

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
		playerTile := playerTransform.Pos.ToCenterTile()
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
		playerTile := playerTransform.Pos.ToCenterTile()
		pivot := config.Vector{X: playerTile.X + vx, Y: playerTile.Y + vy}
		if blinkyTransform == nil {
			return pivot.ToPixel()
		}
		blinkyTile := blinkyTransform.Pos.ToCenterTile()
		dx := pivot.X - blinkyTile.X
		dy := pivot.Y - blinkyTile.Y
		return config.Vector{X: blinkyTile.X + 2*dx, Y: blinkyTile.Y + 2*dy}.ToPixel()

	case component.ClydeAI:
		pv := playerTransform.Pos.ToCenterTile()
		sv := ghostTransform.Pos.ToCenterTile()
		if pv.DistanceSq(sv) >= 64 {
			return pv.ToPixel()
		}
		return ghost.ScatterTarget

	default:
		return config.Vector{}
	}
}

// ghostHousePriorityOrder defines the strict order in which waiting ghosts
// receive dot-counter increments and timer-forced releases (Pinky -> Inky -> Clyde).
var ghostHousePriorityOrder = [...]component.GhostAIKind{
	component.PinkyAI,
	component.InkyAI,
	component.ClydeAI,
}

func preferredInHouseGhost(w donburi.World, ghostQuery *donburi.Query) *donburi.Entry {
	for _, kind := range ghostHousePriorityOrder {
		for entry := range ghostQuery.Iter(w) {
			g := component.Ghost.Get(entry)
			if g.State == config.InHouse && g.AIKind == kind {
				return entry
			}
		}
	}
	return nil
}

func personalDotLimit(kind component.GhostAIKind, level int) int {
	switch kind {
	case component.BlinkyAI, component.PinkyAI:
		return 0
	case component.InkyAI:
		if level == 1 {
			return 30
		}
		return 0
	case component.ClydeAI:
		if level == 1 {
			return 60
		}
		if level == 2 {
			return 50
		}
		return 0
	default:
		return 0
	}
}

func ghostHouseTimeoutFrames(level int) int {
	if level >= 5 {
		return config.SecondsToFrames(3)
	}
	return config.SecondsToFrames(4)
}

func shouldLeaveHouse(ghost *component.GhostData, gameState *component.GameStateData) bool {
	if ghost.AIKind == component.BlinkyAI {
		return true
	}
	if gameState.UseGlobalDotCounter {
		switch ghost.AIKind {
		case component.PinkyAI:
			return gameState.GlobalDotCounter >= 7
		case component.InkyAI:
			return gameState.GlobalDotCounter >= 17
		case component.ClydeAI:
			if gameState.GlobalDotCounter >= 32 {
				gameState.UseGlobalDotCounter = false
				gameState.GlobalDotCounter = 0
				return true
			}
			return false
		default:
			return false
		}
	}
	return ghost.DotCounter >= personalDotLimit(ghost.AIKind, gameState.Level)
}

func isValidGhostTile(maze *component.MazeData, state config.GhostState, t config.Vector, dir config.Direction) bool {
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

	if dir == config.Up && (state == config.Chase || state == config.Scatter) {
		if maze.IsGhostWall(t.X, t.Y) {
			return false
		}
	}

	if cell == component.Wall {
		return false
	}

	if cell == component.Door {
		return state == config.LeavingHouse || state == config.EnteringHouse
	}

	return true
}

func chooseDirectionAt(
	maze *component.MazeData,
	prng *component.PRNGData,
	curTile config.Vector,
	curDir config.Direction,
	state config.GhostState,
	target config.Vector,
) config.Direction {
	if state == config.Frightened {
		return chooseFrightenedDirectionAt(maze, prng, curTile, curDir, state)
	}

	bestDir := curDir
	minDist := math.MaxInt
	targetTile := target.ToTile()

	for _, dir := range config.GhostPriorityDirs {
		if dir == curDir.Opposite() {
			continue
		}
		nextTile := curTile.AddDir(dir)
		if !isValidGhostTile(maze, state, nextTile, dir) {
			continue
		}
		if d := nextTile.DistanceSq(targetTile); d < minDist {
			minDist = d
			bestDir = dir
		}
	}

	return bestDir
}

func chooseFrightenedDirectionAt(
	maze *component.MazeData,
	prng *component.PRNGData,
	tile config.Vector,
	curDir config.Direction,
	state config.GhostState,
) config.Direction {
	var allowedDirs []config.Direction

	if isValidGhostTile(maze, state, tile.AddDir(config.Right), config.Right) {
		allowedDirs = append(allowedDirs, config.Right)
	}
	if isValidGhostTile(maze, state, tile.AddDir(config.Left), config.Left) {
		allowedDirs = append(allowedDirs, config.Left)
	}
	if isValidGhostTile(maze, state, tile.AddDir(config.Up), config.Up) {
		allowedDirs = append(allowedDirs, config.Up)
	}
	if isValidGhostTile(maze, state, tile.AddDir(config.Down), config.Down) {
		allowedDirs = append(allowedDirs, config.Down)
	}

	preferred := prng.PreferredDirection()
	reverse := curDir.Opposite()

	for range config.DirCount {
		if slices.Contains(allowedDirs, preferred) && preferred != reverse {
			return preferred
		}
		preferred = preferred.ClockwiseNext()
	}

	return curDir
}

func isCruiseElroy(gameState *component.GameStateData, ghost *component.GhostData) bool {
	return ghost.AIKind == component.BlinkyAI &&
		!gameState.ElroySuspended &&
		gameState.DotsRemaining <= gameState.Spec.Elroy1DotsLeft
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

	if ghost.State == config.Eaten || ghost.State == config.EnteringHouse {
		return config.SpeedEaten
	}

	if inTunnel || ghost.State == config.InHouse || ghost.State == config.LeavingHouse {
		return s.GhostTunnelSpeed
	}

	if ghost.State == config.Frightened {
		return s.GhostFrightSpeed
	}

	if actor.ID == component.Blinky && !gameState.ElroySuspended {
		if gameState.DotsRemaining <= s.Elroy2DotsLeft {
			return s.Elroy2Speed
		}
		if gameState.DotsRemaining <= s.Elroy1DotsLeft {
			return s.Elroy1Speed
		}
	}

	return s.GhostSpeed
}
