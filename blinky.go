package main

type blinkyBehavior struct{}

func NewBlinky(g *Game) *Ghost {
	blinky := &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    Scatter,
		behavior: blinkyBehavior{},
	}

	return blinky
}

func (blinkyBehavior) StartPos(g *Game) {
	g.blinky.setTilePos(14, 11) // always starts outside house
	g.blinky.dir = Left
}

func (blinkyBehavior) HousePos(g *Game) {
	g.blinky.setTilePos(14, 14) // in house
}

func (blinkyBehavior) ScatterTarget(g *Game) {
	// TODO check dots remaining for Elroy 1 / 2
	g.blinky.setTileTarget(GameWidth-3, -3)
}

func (blinkyBehavior) ChaseTarget(g *Game) {
	g.blinky.targetX = g.player.x
	g.blinky.targetY = g.player.y
}

func (blinkyBehavior) Leave(*Game) bool {
	return true
}
