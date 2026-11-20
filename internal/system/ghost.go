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

	for entry := range s.ghostQuery.Iter(w) {
		actor := component.Actor.Get(entry)
		transform := component.Transform.Get(entry)
		motion := component.Motion.Get(entry)
		anim := component.Animation.Get(entry)
		ghost := component.Ghost.Get(entry)

		updateGhostAnimation(anim)

		switch ghost.State {
		case config.InHouse:
			updateGhostInHouse(maze, house, gameState, actor, transform, motion, ghost)
			continue
		case config.LeavingHouse:
			updateGhostLeavingHouse(maze, house, gameState, prng, actor, transform, motion, ghost)
			continue
		case config.EnteringHouse:
			updateGhostEnteringHouse(maze, gameState, actor, transform, motion, ghost)
			continue
		case config.Scatter:
			if IsCentered(transform.Pos) {
				updateGhostTarget(maze, prng, transform, ghost, ghost.ScatterTarget)
			}
		case config.Chase:
			if IsCentered(transform.Pos) && playerTransform != nil {
				target := computeChaseTarget(ghost, transform, playerTransform, blinkyTransform)
				updateGhostTarget(maze, prng, transform, ghost, target)
			}
		case config.Frightened:
			if IsCentered(transform.Pos) {
				dir := chooseGhostDirection(maze, prng, transform, ghost)
				if dir != transform.Dir && ghost.PixelsMoved >= config.Tile {
					transform.Dir = dir
					ghost.PixelsMoved = 0
				}
			}
		case config.Eaten:
			if transform.Pos == house.Entrance {
				ghost.State = config.EnteringHouse
				transform.Dir = config.Down
				continue
			}
			if IsCentered(transform.Pos) {
				updateGhostTarget(maze, prng, transform, ghost, house.Entrance)
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

func updateGhostTarget(
	maze *component.MazeData,
	prng *component.PRNGData,
	transform *component.TransformData,
	ghost *component.GhostData,
	target config.Vector,
) {
	ghost.Target = target
	dir := chooseGhostDirection(maze, prng, transform, ghost)
	if dir != transform.Dir && ghost.PixelsMoved >= config.Tile {
		transform.Dir = dir
		ghost.PixelsMoved = 0
	}
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
) {
	if shouldLeaveHouse(ghost, gameState) {
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
	prng *component.PRNGData,
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

	// Reached the corridor above the door: enter active wave state.
	ghost.State = gameState.GhostState
	ghost.PixelsMoved = config.Tile
	transform.Dir = chooseRandomDirection(prng, transform.Dir, []config.Direction{config.Right, config.Left})
}
