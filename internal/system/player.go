package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/event"
)

// PlayerSystem updates player sprite animation, intersection turning, movement,
// dot/power-pellet eating events, and tunnel wrapping.
type PlayerSystem struct {
	playerQuery *donburi.Query
}

func NewPlayerSystem() *PlayerSystem {
	return &PlayerSystem{
		playerQuery: donburi.NewQuery(
			filter.Contains(
				component.PlayerTag,
				component.Transform,
				component.Motion,
				component.Animation,
				component.PlayerInput,
			),
		),
	}
}

func (s *PlayerSystem) Update(w donburi.World) {
	gameState := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	for entry := range s.playerQuery.Iter(w) {
		transform := component.Transform.Get(entry)
		motion := component.Motion.Get(entry)
		anim := component.Animation.Get(entry)
		input := component.PlayerInput.Get(entry)

		updatePlayerAnimation(anim)

		const threshold = 3

		tx := transform.Pos.X / config.Tile
		ty := transform.Pos.Y / config.Tile
		ox := transform.Pos.X % config.Tile
		oy := transform.Pos.Y % config.Tile

		// Direction change: opposite reversal is immediate; perpendicular turns
		// trigger when near an intersection.
		if input.Active || input.Requested != transform.Dir {
			if transform.Dir.IsOpposite(input.Requested) {
				transform.Dir = input.Requested
			} else {
				nearX := ox <= threshold || ox >= config.Tile-threshold
				nearY := oy <= threshold || oy >= config.Tile-threshold

				if transform.Dir == config.Up || transform.Dir == config.Down {
					if nearY {
						targetTy := ty
						if oy > config.Half {
							targetTy++
						}
						if CanPlayerMove(maze, tx, targetTy, input.Requested) {
							transform.Dir = input.Requested
						}
					}
				} else {
					if nearX {
						targetTx := tx
						if ox > config.Half {
							targetTx++
						}
						if CanPlayerMove(maze, targetTx, ty, input.Requested) {
							transform.Dir = input.Requested
						}
					}
				}
			}
		}

		// Move if not blocked by a wall or ghost house door.
		canMove := false
		if transform.Dir == config.Up || transform.Dir == config.Down {
			if oy != 0 {
				canMove = true
			} else {
				targetTx := tx
				if ox > config.Half {
					targetTx++
				}
				canMove = CanPlayerMove(maze, targetTx, ty, transform.Dir)
			}
		} else {
			if ox != 0 {
				canMove = true
			} else {
				targetTy := ty
				if oy > config.Half {
					targetTy++
				}
				canMove = CanPlayerMove(maze, tx, targetTy, transform.Dir)
			}
		}

		if canMove {
			MoveActor(transform, motion, playerSpeed(gameState))
		}

		// Eat dots/powerups using center-aligned tile coordinates.
		centerTile := transform.Pos.ToCenterTile()
		if maze.InBounds(centerTile.X, centerTile.Y) {
			tile := maze.GetCell(centerTile.X, centerTile.Y)
			switch tile {
			case component.Dot:
				event.DotEaten.Publish(w, event.DotEatenEvent{X: centerTile.X, Y: centerTile.Y})
			case component.Power:
				event.PowerUpEaten.Publish(w, event.PowerUpEatenEvent{X: centerTile.X, Y: centerTile.Y})
			}
		}

		WrapHorizontal(&transform.Pos)
	}
}

// CanPlayerMove reports whether the tile adjacent to (tx, ty) in dir is passable for the player.
func CanPlayerMove(maze *component.MazeData, tx, ty int, dir config.Direction) bool {
	nx, ny := tx, ty

	switch dir {
	case config.Up:
		ny--
	case config.Down:
		ny++
	case config.Left:
		nx--
	case config.Right:
		nx++
	}

	if nx < 0 {
		nx = config.GameWidth - 1
	}
	if nx >= config.GameWidth {
		nx = 0
	}

	if ny < 0 || ny >= config.GameHeight {
		return false
	}

	tile := maze.GetCell(nx, ny)
	return tile != component.Wall && tile != component.Door
}

// updatePlayerAnimation cycles Pac-Man's 4-step mouth animation:
// closed (2) -> half-open (1) -> fully open (0) -> half-open (1), 5 frames each.
func updatePlayerAnimation(anim *component.AnimationData) {
	const framesPerState = 5
	const cycleLength = 4 * framesPerState
	frameInCycle := anim.FrameCount % cycleLength
	switch {
	case frameInCycle < framesPerState:
		anim.FrameIndex = 2 // Closed
	case frameInCycle < 2*framesPerState:
		anim.FrameIndex = 1 // Half-open
	case frameInCycle < 3*framesPerState:
		anim.FrameIndex = 0 // Fully open
	default:
		anim.FrameIndex = 1 // Half-open (returning)
	}

	anim.FrameCount++
	if anim.FrameCount > cycleLength {
		anim.FrameCount = 0
	}
}

func playerSpeed(gameState *component.GameStateData) uint32 {
	if gameState.IsFrightMode {
		return gameState.Spec.PacmanFrightSpeed
	}
	return gameState.Spec.PacmanSpeed
}
