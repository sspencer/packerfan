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
				if dir != transform.Dir && ghost.PixelsMoved > config.Tile {
					transform.Dir = dir
					ghost.PixelsMoved = 0
				}
			}
		case config.Eaten:
			updateGhostTarget(maze, prng, transform, ghost, ghost.EatenTarget)
		default:
			fmt.Printf("unhandled %q state for ghost %s\n", ghost.State.String(), actor.ID.String())
		}

		MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
		WrapHorizontal(&transform.Pos)
		ghost.PixelsMoved++
	}
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
	if dir != transform.Dir && ghost.PixelsMoved > config.Tile {
		transform.Dir = dir
		ghost.PixelsMoved = 0
	}
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
	const (
		houseX = 108
		houseY = 88
	)

	if transform.Pos.X == houseX && transform.Pos.Y <= houseY {
		transform.Pos.Y = houseY
		ghost.State = gameState.GhostState
		transform.Dir = chooseRandomDirection(prng, transform.Dir, []config.Direction{config.Right, config.Left})
		return
	}

	if transform.Pos.X < houseX {
		transform.Dir = config.Right
	} else if transform.Pos.X > houseX {
		transform.Dir = config.Left
	} else {
		transform.Dir = config.Up
	}

	MoveActor(transform, motion, ghostSpeed(maze, gameState, actor, transform, ghost))
}
