package main

import (
	"fmt"
	"math"
	"slices"
)

//go:generate stringer -type=GhostState
type GhostState int

const (
	Scatter GhostState = iota
	Chase
	Frightened
	Eaten
	EnteringHouse
	InHouse
	LeavingHouse
)

type GhostBrain interface {
	chaseAI(*Ghost) Vector
	leaveAI(*Ghost) bool
}

type GhostConfig struct {
	startState    GhostState
	eatenTarget   Vector
	scatterTarget Vector
	brain         GhostBrain
}

type Ghost struct {
	Actor
	target      Vector
	state       GhostState
	pixelsMoved int
	config      GhostConfig
}

func (g *Ghost) Reset() {
	g.state = g.config.startState
	g.pixelsMoved = 0
	g.ResetToStart()
}

func (g *Ghost) ChaseTarget() Vector {
	if g.config.brain == nil {
		return Vector{}
	}
	return g.config.brain.chaseAI(g)
}

func (g *Ghost) ScatterTarget() Vector {
	return g.config.scatterTarget
}

func (g *Ghost) ShouldLeave() bool {
	if g.config.brain == nil {
		return false
	}
	return g.config.brain.leaveAI(g)
}

func (g *Ghost) HouseTarget() Vector {
	return g.config.eatenTarget
}

func newGhost(actor Actor, config GhostConfig) *Ghost {
	if config.brain == nil {
		panic("ghost brain is required")
	}
	actor.isGhost = true

	g := &Ghost{
		Actor:  actor,
		config: config,
	}

	g.Reset()

	return g
}

func (g *Ghost) draw() {
	// r l u d -- graphic
	// u d l r -- dir
	var sprite Vector
	if g.state == Eaten {
		sprite.x = []int{616, 584, 632, 600}[g.dir]
		sprite.y = 80
	} else if g.state == Frightened {
		sprite.x = 584 // white is 616
		sprite.y = 64
		/*
		   if (game.frightened_frames >= game.frightened_flashes * FRIGHTENED_FLASH_FRAMES) {
		       return (Vector2){blues_x[dir], 64};
		   }

		   const int n = (game.frightened_frames / FRIGHTENED_FLASH_FRAMES) + 1;
		   if (n % 2 == 0) {
		       return (Vector2){blues_x[dir], 64};
		   }

		   return (Vector2){whites_x[dir], 64};
		*/
	} else {
		sprite.y = []int{64, 80, 96, 112}[int(g.Actor.ID)] // blinky, inky, pinky, clyde
		sprite.x = []int{456, 488, 520, 552}[g.dir]
	}

	drawSprite(g.pos, sprite, g.frameIndex)
}

func normalizeTunnelTile(tile Vector) Vector {
	if tile.x < 0 {
		tile.x += GameWidth
	}
	if tile.x >= GameWidth {
		tile.x -= GameWidth
	}
	return tile
}

func (g *Ghost) isValidTile(t Vector) bool {
	wrappedTile := normalizeTunnelTile(t)

	if wrappedTile.y < 0 || wrappedTile.y >= GameHeight {
		return false
	}

	cell := maze.getCell(wrappedTile.x, wrappedTile.y)

	if g.state == Chase || g.state == Scatter {
		if maze.isGhostWall(wrappedTile.x, wrappedTile.y) {
			return false
		}
	}

	if cell == Wall {
		return false
	}

	if cell == Door {
		return g.state == LeavingHouse || g.state == Eaten || g.state == EnteringHouse
	}

	return true
}

func (g *Ghost) chooseDirection() Direction {
	if g.state == Frightened {
		return g.chooseFrightenedDirection()
	}

	bestDir := g.dir
	minDist := math.MaxInt
	targetTile := g.target.ToTile()

	for _, dir := range ghostPriorityDir {
		if dir == g.dir.Opposite() {
			continue
		}
		nextTile := g.nextTile(dir)
		if !g.isValidTile(nextTile) {
			continue
		}
		if d := nextTile.DistanceSq(targetTile); d < minDist {
			minDist = d
			bestDir = dir
		}
	}

	return bestDir
}

func (g *Ghost) nextTile(dir Direction) Vector {
	return g.pos.ToTile().AddDir(dir)
}

// Simulate choosing the next direction at an intersection (reverse won't be chosen)
func (g *Ghost) chooseFrightenedDirection() Direction {
	tile := g.pos.ToTile()
	var allowedDirs []Direction

	if g.isValidTile(tile.AddDir(Right)) {
		allowedDirs = append(allowedDirs, Right)
	}
	if g.isValidTile(tile.AddDir(Left)) {
		allowedDirs = append(allowedDirs, Left)
	}
	if g.isValidTile(tile.AddDir(Up)) {
		allowedDirs = append(allowedDirs, Up)
	}
	if g.isValidTile(tile.AddDir(Down)) {
		allowedDirs = append(allowedDirs, Down)
	}

	preferred := getPreferredDirection()
	reverse := g.dir.Opposite()

	for range DirCount {
		if slices.Contains(allowedDirs, preferred) && preferred != reverse {
			return preferred
		}
		preferred = clockwiseNext(preferred)
	}

	// Should never reach here if at least one valid direction exists
	return g.dir // fallback
}

// Simulate choosing any of the allowed directions (including reverse)
func (g *Ghost) chooseRandomDirection(allowedDirs []Direction) Direction {
	preferred := getPreferredDirection()

	for range DirCount {
		if slices.Contains(allowedDirs, preferred) {
			return preferred
		}
		preferred = clockwiseNext(preferred)
	}

	// Should never reach here if at least one valid direction exists
	return g.dir
}

func (g *Ghost) updateFrame() {
	const framesPerState = 15
	const cycleLength = 2 * framesPerState // 30 frames for full cycle
	frameInCycle := g.frameCount % cycleLength
	if frameInCycle < framesPerState {
		g.frameIndex = 0 // First pose
	} else {
		g.frameIndex = 1
	}
	g.frameCount++
	if g.frameCount > cycleLength {
		g.frameCount = 0
	}
}

func (g *Ghost) update() {
	g.updateFrame()
	switch g.state {
	case InHouse:
		g.updateInHouse()
		return // DONE
	case LeavingHouse:
		g.updateLeavingHouse()
		return // DONE
	case Scatter:
		g.updateScatter()
	default:
		fmt.Printf("unhandled %q state for ghost %s\n", g.state.String(), g.ID.String())
	}

	g.moveActor(g.getSpeed())
	g.teleport() // teleport to other side of screen when out of bounds
	g.pixelsMoved++
}

func (g *Ghost) updateTarget(target Vector) {
	g.target = target
	dir := g.chooseDirection()
	if dir != g.dir && g.pixelsMoved > Tile {
		g.dir = dir
		g.pixelsMoved = 0
	}
}

func (g *Ghost) updateScatter() {
	if !g.isCentered() {
		return
	}

	g.updateTarget(g.ScatterTarget())
}

func (g *Ghost) updateInHouse() {
	const (
		bounceTop    = 14*Tile - Half
		bounceBottom = 14*Tile + Half
	)

	if g.ShouldLeave() {
		g.state = LeavingHouse
		return
	}

	moved := g.moveActor(g.getSpeed())
	if moved && (g.pos.y == bounceBottom || g.pos.y == bounceTop) {
		g.dir = g.dir.Opposite()
	}
}

func (g *Ghost) updateLeavingHouse() {
	const (
		houseX = 108
		houseY = 88
	)

	if g.pos.x == houseX && g.pos.y <= houseY {
		g.pos.y = houseY // just in case ghost moves 2 pixels and misses target
		g.state = game.ghostState
		return
	}

	// still in house - move to horizontal center then start ascending
	if g.pos.x < houseX {
		g.dir = Right
	} else if g.pos.x > houseX {
		g.dir = Left
	} else {
		g.dir = Up
	}

	g.moveActor(g.getSpeed())
}

func (g *Ghost) getSpeed() uint32 {
	s := game.spec
	tile := g.pos.ToTile()

	if maze.getCell(tile.x, tile.y) == Tunnel || g.state == InHouse || g.state == LeavingHouse || g.state == EnteringHouse {
		return s.GhostTunnelSpeed
	}

	//if g.state == Eaten {
	//	return Speed80
	//}

	if g.state == Frightened {
		return s.GhostFrightSpeed
	}

	if g.ID == Blinky {
		if game.dotsRemaining <= s.Elroy2DotsLeft {
			return s.Elroy2Speed
		}

		if game.dotsRemaining <= s.Elroy1DotsLeft {
			return s.Elroy1Speed
		}
	}

	return s.GhostSpeed
}

func (g *Ghost) teleport() {
	const edge = GameWidth * Tile

	if g.pos.x < 0 {
		g.pos.x += edge
	} else if g.pos.x >= edge {
		g.pos.x -= edge
	}
}
