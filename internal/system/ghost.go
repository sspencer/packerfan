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

const (
	houseEntranceX = 108
	houseEntranceY = 88
)

func (s *GhostSystem) Update(w donburi.World) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)
	prng := component.MustFindPRNG(w)

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
			updateGhostInHouse(maze, gameState, actor, transform, motion, ghost)
			continue
		case config.LeavingHouse:
			updateGhostLeavingHouse(maze, gameState, prng, actor, transform, motion, ghost)
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
			if transform.Pos.X == houseEntranceX && transform.Pos.Y == houseEntranceY {
				ghost.State = config.EnteringHouse
				transform.Dir = config.Down
				continue
			}
			if IsCentered(transform.Pos) {
				entrance := config.Vector{X: houseEntranceX, Y: houseEntranceY}
				updateGhostTarget(maze, prng, transform, ghost, entrance)
			}
		default:
			fmt.Printf("unhandled %q state for ghost %s\n", ghost.State.String(), actor.ID.String())
		}

		oldPos := transform.Pos
		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		ghost.PixelsMoved += absInt(transform.Pos.X-oldPos.X) + absInt(transform.Pos.Y-oldPos.Y)
		WrapHorizontal(&transform.Pos)

		if ghost.State == config.Eaten && transform.Pos.X == houseEntranceX && transform.Pos.Y == houseEntranceY {
			ghost.State = config.EnteringHouse
			transform.Dir = config.Down
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
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

	if transform.Pos.Y < target.Y {
		transform.Dir = config.Down
		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		if transform.Pos.Y > target.Y {
			transform.Pos.Y = target.Y
		}
		return
	}

	if transform.Pos.X < target.X {
		transform.Dir = config.Right
		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		if transform.Pos.X > target.X {
			transform.Pos.X = target.X
		}
		return
	}

	if transform.Pos.X > target.X {
		transform.Dir = config.Left
		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		if transform.Pos.X < target.X {
			transform.Pos.X = target.X
		}
		return
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
	gameState *component.GameStateData,
	actor *component.ActorData,
	transform *component.TransformData,
	motion *component.MotionData,
	ghost *component.GhostData,
) {
	const (
		bounceTop    = 14*config.Tile - config.Half
		bounceBottom = 14*config.Tile + config.Half
	)

	if shouldLeaveHouse(ghost, gameState) {
		ghost.State = config.LeavingHouse
		return
	}

	moved := MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
	if moved && (transform.Pos.Y == bounceBottom || transform.Pos.Y == bounceTop) {
		transform.Dir = transform.Dir.Opposite()
	}
}

func updateGhostLeavingHouse(
	maze *component.MazeData,
	gameState *component.GameStateData,
	prng *component.PRNGData,
	actor *component.ActorData,
	transform *component.TransformData,
	motion *component.MotionData,
	ghost *component.GhostData,
) {
	if transform.Pos.X == houseEntranceX && transform.Pos.Y <= houseEntranceY {
		transform.Pos.Y = houseEntranceY
		ghost.State = gameState.GhostState
		ghost.PixelsMoved = config.Tile
		transform.Dir = chooseRandomDirection(prng, transform.Dir, []config.Direction{config.Right, config.Left})
		return
	}

	if transform.Pos.X < houseEntranceX {
		transform.Dir = config.Right
	} else if transform.Pos.X > houseEntranceX {
		transform.Dir = config.Left
	} else {
		transform.Dir = config.Up
	}

	MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
}
