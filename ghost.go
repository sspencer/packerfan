package main

import "fmt"

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
	ChaseTarget(*Ghost) Vector
	ShouldLeaveHouse(*Ghost) bool
}

type GhostConfig struct {
	StartState    GhostState
	HouseTarget   Vector
	ScatterTarget Vector
	Brain         GhostBrain
}

type Ghost struct {
	Actor
	target  Vector
	state   GhostState
	reverse bool
	config  GhostConfig
}

func (g *Ghost) Start() {
	g.state = g.config.StartState
	g.ResetToStart()
}

func (g *Ghost) Chase() {
	if g.config.Brain == nil {
		return
	}
	g.target = g.config.Brain.ChaseTarget(g)
}

func (g *Ghost) Scatter() {
	g.target = g.config.ScatterTarget
}

func (g *Ghost) Leave() bool {
	if g.config.Brain == nil {
		return false
	}
	return g.config.Brain.ShouldLeaveHouse(g)
}

func (g *Ghost) House() {
	g.target = g.config.HouseTarget
}

func newGhost(actor Actor, config GhostConfig) *Ghost {
	if config.Brain == nil {
		panic("ghost brain is required")
	}
	actor.isGhost = true

	g := &Ghost{
		Actor:  actor,
		config: config,
	}

	g.Start()

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

func (g *Ghost) chooseDirection() {

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
	case LeavingHouse:
		g.updateLeavingHouse()
	case Scatter:
		g.updateScatter()
	default:
		fmt.Printf("unhandled %q state for ghost %s\n", g.state.String(), g.ID.String())
	}
}

func (g *Ghost) updateScatter() {
	g.moveActor(g.getSpeed())
}

func (g *Ghost) updateInHouse() {
	if g.Leave() {
		g.state = LeavingHouse
	} else {
		const bounce1 = 14*Tile + Half
		const bounce2 = 14*Tile - Half
		moved := g.moveActor(g.getSpeed())

		if moved && (g.pos.y == bounce1 || g.pos.y == bounce2) {
			g.dir = g.dir.Opposite()
		}
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
		if g.state == Scatter {
			g.Scatter()
		} else {
			g.Chase()
		}

		g.chooseDirection()
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
