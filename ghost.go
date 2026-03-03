package main

type GhostID int

const (
	Blinky GhostID = iota
	Pinky
	Inky
	Clyde
)

type Behavior interface {
	ID() GhostID
	Chase(*Game, *Ghost)
	Scatter(*Game, *Ghost)
	Leave(*Game, *Ghost) bool
	House(*Game, *Ghost)
	Start(*Game, *Ghost)
}

type Ghost struct {
	Actor
	target   Vector
	state    GhostState
	reverse  bool
	behavior Behavior
}

func (g *Ghost) ID() GhostID {
	if g.behavior != nil {
		return g.behavior.ID()
	}
	return Blinky
}

func (g *Ghost) Chase(gm *Game) {
	if g.behavior == nil {
		return
	}
	g.behavior.Chase(gm, g)
}

func (g *Ghost) Scatter(gm *Game) {
	if g.behavior == nil {
		return
	}
	g.behavior.Scatter(gm, g)
}

func (g *Ghost) Leave(gm *Game) bool {
	if g.behavior == nil {
		return false
	}
	return g.behavior.Leave(gm, g)
}

func (g *Ghost) House(gm *Game) {
	if g.behavior == nil {
		return
	}
	g.behavior.House(gm, g)
}

func (g *Ghost) Start(gm *Game) {
	if g.behavior == nil {
		return
	}
	g.behavior.Start(gm, g)
}

func newGhost(gm *Game, ai Behavior) *Ghost {
	g := &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		behavior: ai,
	}

	ai.Start(gm, g)

	return g
}

func (gm *Game) drawGhost(g *Ghost) {
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
		sprite.y = []int{64, 80, 96, 112}[g.ID()] // blinky, inky, pinky, clyde
		sprite.x = []int{456, 488, 520, 552}[g.dir]
	}

	gm.drawActor2(g.pos, sprite, g.frameIndex)
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

func (gm *Game) updateGhost(g *Ghost) {
	g.updateFrame()
}
