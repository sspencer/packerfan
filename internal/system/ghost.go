package system

import (
	"fmt"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

// GhostSystem updates ghost animation, house exit logic, AI targeting,
// intersection direction selection, movement, and tunnel wrapping.
type GhostSystem struct {
	ghostQuery  *donburi.Query
	playerQuery *donburi.Query
}

func NewGhostSystem() *GhostSystem {
	return &GhostSystem{
		ghostQuery: donburi.NewQuery(
			filter.Contains(
				component.GhostTag,
				component.Actor,
				component.Transform,
				component.Motion,
				component.Animation,
				component.Ghost,
			),
		),
		playerQuery: donburi.NewQuery(
			filter.Contains(component.PlayerTag, component.Transform),
		),
	}
}

func (s *GhostSystem) Update(w donburi.World) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)
	prng := component.MustFindPRNG(w)
	house := maze.GhostHouse()

	playerEntry, hasPlayer := s.playerQuery.First(w)
	var playerTransform *component.TransformData
	if hasPlayer {
		playerTransform = component.Transform.Get(playerEntry)
	}

	var blinkyTransform *component.TransformData
	for entry := range s.ghostQuery.Iter(w) {
		if component.Actor.Get(entry).ID == component.Blinky {
			blinkyTransform = component.Transform.Get(entry)
			break
		}
	}

	preferredHouseEntry := preferredInHouseGhost(w, s.ghostQuery)
	timeoutRelease := false
	if preferredHouseEntry != nil {
		gameState.FramesSinceLastDot++
		if gameState.FramesSinceLastDot >= ghostHouseTimeoutFrames(gameState.Level) {
			gameState.FramesSinceLastDot = 0
			timeoutRelease = true
		}
	}

	for entry := range s.ghostQuery.Iter(w) {
		actor := component.Actor.Get(entry)
		transform := component.Transform.Get(entry)
		motion := component.Motion.Get(entry)
		anim := component.Animation.Get(entry)
		ghost := component.Ghost.Get(entry)

		updateGhostAnimation(anim)

		switch ghost.State {
		case config.InHouse:
			isPreferred := preferredHouseEntry != nil && entry.Entity() == preferredHouseEntry.Entity()
			forceRelease := isPreferred && timeoutRelease
			updateGhostInHouse(maze, house, gameState, actor, transform, motion, ghost, isPreferred, forceRelease)
			continue
		case config.LeavingHouse:
			updateGhostLeavingHouse(maze, house, gameState, actor, transform, motion, ghost)
			continue
		case config.EnteringHouse:
			updateGhostEnteringHouse(maze, gameState, actor, transform, motion, ghost)
			continue
		case config.Scatter:
			if IsCentered(transform.Pos) {
				target := ghost.ScatterTarget
				if isCruiseElroy(gameState, ghost) && playerTransform != nil {
					target = computeChaseTarget(ghost, transform, playerTransform, blinkyTransform)
				}
				updateGhostTurnWithLookahead(maze, prng, transform, ghost, target)
			}
		case config.Chase:
			if IsCentered(transform.Pos) && playerTransform != nil {
				target := computeChaseTarget(ghost, transform, playerTransform, blinkyTransform)
				updateGhostTurnWithLookahead(maze, prng, transform, ghost, target)
			}
		case config.Frightened:
			if IsCentered(transform.Pos) {
				updateGhostTurnWithLookahead(maze, prng, transform, ghost, config.Vector{})
			}
		case config.Eaten:
			if transform.Pos == house.Entrance {
				ghost.State = config.EnteringHouse
				ghost.HasNextDir = false
				transform.Dir = config.Down
				continue
			}
			if IsCentered(transform.Pos) {
				updateGhostTurnWithLookahead(maze, prng, transform, ghost, house.Entrance)
			}
		default:
			fmt.Printf("unhandled %q state for ghost %s\n", ghost.State.String(), actor.ID.String())
		}

		oldPos := transform.Pos
		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		ghost.PixelsMoved += absInt(transform.Pos.X-oldPos.X) + absInt(transform.Pos.Y-oldPos.Y)

		if ghost.State == config.Eaten && crossedHorizontalPoint(oldPos, transform.Pos, house.Entrance) {
			transform.Pos = house.Entrance
			ghost.State = config.EnteringHouse
			ghost.HasNextDir = false
			transform.Dir = config.Down
			continue
		}

		WrapHorizontal(&transform.Pos)
	}
}

// UpdateEatenGhosts advances only previously eaten ghosts (Eaten or EnteringHouse,
// excluding the just-eaten GhostEatenEntity) so retreating eyes continue returning
// to the house during the ghost-eaten score freeze.
func (s *GhostSystem) UpdateEatenGhosts(w donburi.World) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)
	prng := component.MustFindPRNG(w)
	house := maze.GhostHouse()

	for entry := range s.ghostQuery.Iter(w) {
		if entry.Entity() == gameState.GhostEatenEntity {
			continue
		}
		ghost := component.Ghost.Get(entry)
		if ghost.State != config.Eaten && ghost.State != config.EnteringHouse {
			continue
		}

		actor := component.Actor.Get(entry)
		transform := component.Transform.Get(entry)
		motion := component.Motion.Get(entry)

		if ghost.State == config.EnteringHouse {
			updateGhostEnteringHouse(maze, gameState, actor, transform, motion, ghost)
			continue
		}

		if transform.Pos == house.Entrance {
			ghost.State = config.EnteringHouse
			ghost.HasNextDir = false
			transform.Dir = config.Down
			continue
		}
		if IsCentered(transform.Pos) {
			updateGhostTurnWithLookahead(maze, prng, transform, ghost, house.Entrance)
		}

		oldPos := transform.Pos
		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		ghost.PixelsMoved += absInt(transform.Pos.X-oldPos.X) + absInt(transform.Pos.Y-oldPos.Y)

		if crossedHorizontalPoint(oldPos, transform.Pos, house.Entrance) {
			transform.Pos = house.Entrance
			ghost.State = config.EnteringHouse
			ghost.HasNextDir = false
			transform.Dir = config.Down
			continue
		}

		WrapHorizontal(&transform.Pos)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// crossedHorizontalPoint reports whether moving from oldPos to newPos along
// target.Y reached or crossed target.X, making entrance detection immune to
// multi-pixel steps or odd pixel offsets.
func crossedHorizontalPoint(oldPos, newPos, target config.Vector) bool {
	if oldPos.Y != target.Y || newPos.Y != target.Y {
		return false
	}
	return (oldPos.X <= target.X && newPos.X >= target.X) ||
		(oldPos.X >= target.X && newPos.X <= target.X)
}

// stepAxisToward advances transform along a single axis toward targetValue via
// MoveActor and clamps against overshoot so multi-pixel steps never oscillate.
// It returns true once the axis coordinate equals targetValue.
func stepAxisToward(
	transform *component.TransformData,
	motion *component.MotionData,
	speed uint32,
	horizontal bool,
	targetValue int,
) bool {
	if horizontal {
		if transform.Pos.X == targetValue {
			return true
		}
		if transform.Pos.X < targetValue {
			transform.Dir = config.Right
			MoveActor(transform, motion, speed)
			if transform.Pos.X > targetValue {
				transform.Pos.X = targetValue
			}
		} else {
			transform.Dir = config.Left
			MoveActor(transform, motion, speed)
			if transform.Pos.X < targetValue {
				transform.Pos.X = targetValue
			}
		}
		return transform.Pos.X == targetValue
	}

	if transform.Pos.Y == targetValue {
		return true
	}
	if transform.Pos.Y < targetValue {
		transform.Dir = config.Down
		MoveActor(transform, motion, speed)
		if transform.Pos.Y > targetValue {
			transform.Pos.Y = targetValue
		}
	} else {
		transform.Dir = config.Up
		MoveActor(transform, motion, speed)
		if transform.Pos.Y < targetValue {
			transform.Pos.Y = targetValue
		}
	}
	return transform.Pos.Y == targetValue
}

func updateGhostAnimation(anim *component.AnimationData) {
	const framesPerState = 15
	const cycleLength = 2 * framesPerState // 30 frames for full cycle
	frameInCycle := anim.FrameCount % cycleLength
	if frameInCycle < framesPerState {
		anim.FrameIndex = 0
	} else {
		anim.FrameIndex = 1
	}
	anim.FrameCount++
	if anim.FrameCount > cycleLength {
		anim.FrameCount = 0
	}
}

// updateGhostTurnWithLookahead applies any reversal or previously planned
// one-tile-ahead direction at the current centered tile, then precomputes the
// turn to execute when the ghost reaches the next tile.
func updateGhostTurnWithLookahead(
	maze *component.MazeData,
	prng *component.PRNGData,
	transform *component.TransformData,
	ghost *component.GhostData,
	target config.Vector,
) {
	curTile := transform.Pos.ToTile()
	ghost.Target = target

	switch {
	case ghost.Reverse:
		ghost.Reverse = false
		ghost.HasNextDir = false
		if ghost.PixelsMoved >= config.Tile {
			transform.Dir = transform.Dir.Opposite()
			ghost.PixelsMoved = 0
		}
	case ghost.HasNextDir && curTile == ghost.PlannedTile:
		dir := ghost.NextDir
		ghost.HasNextDir = false
		if dir != transform.Dir && ghost.PixelsMoved >= config.Tile {
			transform.Dir = dir
			ghost.PixelsMoved = 0
		}
	default:
		ghost.HasNextDir = false
		dir := chooseDirectionAt(maze, prng, curTile, transform.Dir, ghost.State, target)
		if dir != transform.Dir && ghost.PixelsMoved >= config.Tile {
			transform.Dir = dir
			ghost.PixelsMoved = 0
		}
	}

	nextTile := curTile.AddDir(transform.Dir)
	if nextTile.X < 0 {
		nextTile.X += config.GameWidth
	} else if nextTile.X >= config.GameWidth {
		nextTile.X -= config.GameWidth
	}
	if !isValidGhostTile(maze, ghost.State, nextTile, transform.Dir) {
		return
	}

	ghost.NextDir = chooseDirectionAt(maze, prng, nextTile, transform.Dir, ghost.State, target)
	ghost.PlannedTile = nextTile
	ghost.HasNextDir = true
}

func updateGhostEnteringHouse(
	maze *component.MazeData,
	gameState *component.GameStateData,
	actor *component.ActorData,
	transform *component.TransformData,
	motion *component.MotionData,
	ghost *component.GhostData,
) {
	target := ghost.EatenTarget
	speed := ghostSpeed(maze, gameState, actor, transform, ghost)

	// Phase 1: descend vertically through the door to the home row.
	if transform.Pos.Y != target.Y {
		if !stepAxisToward(transform, motion, speed, false, target.Y) || transform.Pos.X != target.X {
			return
		}
	} else if transform.Pos.X != target.X {
		// Phase 2: slide horizontally into the ghost's home slot.
		if !stepAxisToward(transform, motion, speed, true, target.X) {
			return
		}
	}

	// Reached home slot inside the ghost house: regenerate into a normal ghost
	// and immediately begin exiting the house.
	actor.IsEaten = false
	ghost.Reverse = false
	ghost.FrightenedInHouse = false
	ghost.PixelsMoved = 0
	ghost.State = config.LeavingHouse
}

func updateGhostInHouse(
	maze *component.MazeData,
	house component.GhostHouse,
	gameState *component.GameStateData,
	actor *component.ActorData,
	transform *component.TransformData,
	motion *component.MotionData,
	ghost *component.GhostData,
	isPreferred bool,
	forceRelease bool,
) {
	if forceRelease || (isPreferred && shouldLeaveHouse(ghost, gameState)) || ghost.AIKind == component.BlinkyAI {
		ghost.State = config.LeavingHouse
		return
	}

	if transform.Dir != config.Up && transform.Dir != config.Down {
		transform.Dir = config.Up
	}

	MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
	if transform.Pos.Y <= house.BounceTop {
		transform.Pos.Y = house.BounceTop
		transform.Dir = config.Down
	} else if transform.Pos.Y >= house.BounceBottom {
		transform.Pos.Y = house.BounceBottom
		transform.Dir = config.Up
	}
}

func updateGhostLeavingHouse(
	maze *component.MazeData,
	house component.GhostHouse,
	gameState *component.GameStateData,
	actor *component.ActorData,
	transform *component.TransformData,
	motion *component.MotionData,
	ghost *component.GhostData,
) {
	speed := ghostSpeed(maze, gameState, actor, transform, ghost)

	if transform.Pos.X != house.Entrance.X {
		// Side-bay ghosts align to the house center row before sliding to the center shaft.
		if transform.Pos.Y != house.Center.Y {
			stepAxisToward(transform, motion, speed, false, house.Center.Y)
			return
		}
		stepAxisToward(transform, motion, speed, true, house.Entrance.X)
		return
	}

	if transform.Pos.Y != house.Entrance.Y {
		if !stepAxisToward(transform, motion, speed, false, house.Entrance.Y) {
			return
		}
	}

	// Reached the corridor above the door: enter Frightened if marked while in
	// house and fright mode is still active; otherwise enter the active wave state.
	if ghost.FrightenedInHouse && gameState.IsFrightMode {
		ghost.State = config.Frightened
	} else {
		ghost.State = gameState.GhostState
	}
	ghost.FrightenedInHouse = false
	ghost.HasNextDir = false
	ghost.PixelsMoved = config.Tile
	if ghost.Reverse {
		transform.Dir = config.Right
		ghost.Reverse = false
	} else {
		transform.Dir = config.Left
	}
	if ghost.AIKind == component.ClydeAI {
		gameState.ElroySuspended = false
	}
}
