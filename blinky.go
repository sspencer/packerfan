package main

var (
	blinkyStartTile = Vector{14, 11}
	blinkyHouseTile = Vector{14, 14}
	blinkyStartDir  = Left
)

type blinkyAI struct{}

func NewBlinky() *Ghost {
	blinky := &Ghost{
		Actor: Actor{
			pos:     blinkyStartTile.ToPixel(),
			dir:     blinkyStartDir,
			isGhost: true,
		},
		state:    Scatter,
		behavior: blinkyAI{},
	}

	return blinky
}

func (blinkyAI) StartPos(g *Game) {
	g.blinky.pos = blinkyStartTile.ToPixel()
	g.blinky.dir = Left
}

func (blinkyAI) HousePos(g *Game) {
	g.blinky.pos = blinkyHouseTile.ToPixel()
}

func (blinkyAI) ScatterTarget(g *Game) {
	// TODO check dots remaining for Elroy 1 / 2
	g.blinky.target = Vector{GameWidth - 3, -3}.ToPixel()
}

func (blinkyAI) ChaseTarget(g *Game) {
	g.blinky.target = g.player.pos
}

func (blinkyAI) Leave(*Game) bool {
	return true
}
