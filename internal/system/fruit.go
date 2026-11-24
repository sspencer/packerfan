package system

import (
	"math"

	"github.com/yohamta/donburi"

	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
	"cheddarhead/internal/event"
)

const fruitBounceCycle = 16

// FruitSystem navigates Ms. Pac-Man's bouncing bonus fruit from its entry
// tunnel past the center of the maze and out through its chosen exit tunnel.
type FruitSystem struct{}

func NewFruitSystem() *FruitSystem {
	return &FruitSystem{}
}

func (s *FruitSystem) Update(w donburi.World) {
	s.updateFruit(w, component.MustFindGameState(w), nil, nil)
}

func (s *FruitSystem) updateCached(w donburi.World, c *worldCache) {
	if !c.gs.FruitActive || !c.gs.FruitMoving {
		return
	}
	s.updateFruit(w, c.gs, c.maze, c.bus)
}

func (s *FruitSystem) updateFruit(w donburi.World, gs *component.GameStateData, maze *component.MazeData, bus *event.Bus) {
	if !gs.FruitActive || !gs.FruitMoving {
		return
	}

	if maze == nil {
		maze = component.MustFindMaze(w)
	}

	if gs.FruitBounceFrame == 0 {
		if bus != nil {
			event.FruitBounced.PublishTo(bus, event.FruitBouncedEvent{})
		} else {
			event.FruitBounced.Publish(w, event.FruitBouncedEvent{})
		}
	}
	gs.FruitBounceFrame = (gs.FruitBounceFrame + 1) % fruitBounceCycle

	if IsCentered(gs.FruitPos) {
		curTile := gs.FruitPos.ToTile()
		targetTile := gs.FruitTarget.ToTile()

		if curTile.DistanceSq(targetTile) <= 1 {
			if !gs.FruitExiting {
				gs.FruitExiting = true
				gs.FruitTarget = gs.FruitExitPos
				targetTile = gs.FruitTarget.ToTile()
			} else {
				gs.FruitActive = false
				gs.FruitMoving = false
				gs.FruitExiting = false
				gs.FruitTimer = 0
				return
			}
		}

		gs.FruitDir = chooseFruitDirection(maze, curTile, gs.FruitDir, targetTile)
	}

	tr := component.TransformData{Pos: gs.FruitPos, Dir: gs.FruitDir}
	MoveActor(&tr, &gs.FruitMotion, config.Speed40)
	gs.FruitPos = tr.Pos

	// Despawn cleanly when the exiting fruit crosses out of the side tunnel edge.
	if gs.FruitExiting && (gs.FruitPos.X <= 0 || gs.FruitPos.X >= (config.GameWidth-1)*config.Tile) {
		if gs.FruitPos.ToTile().Y == gs.FruitExitPos.ToTile().Y {
			gs.FruitActive = false
			gs.FruitMoving = false
			gs.FruitExiting = false
			gs.FruitTimer = 0
			return
		}
	}

	WrapHorizontal(&gs.FruitPos)
}

func chooseFruitDirection(maze *component.MazeData, curTile config.Vector, curDir config.Direction, targetTile config.Vector) config.Direction {
	bestDir := curDir
	minDist := math.MaxInt
	found := false

	for _, dir := range config.GhostPriorityDirs {
		if dir == curDir.Opposite() {
			continue
		}
		if !CanPlayerMove(maze, curTile.X, curTile.Y, dir) {
			continue
		}
		nextTile := curTile.AddDir(dir)
		if d := nextTile.DistanceSq(targetTile); d < minDist {
			minDist = d
			bestDir = dir
			found = true
		}
	}

	if found {
		return bestDir
	}
	if CanPlayerMove(maze, curTile.X, curTile.Y, curDir.Opposite()) {
		return curDir.Opposite()
	}
	return curDir
}
