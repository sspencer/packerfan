package main

type blinkyAI struct {
}

func NewBlinky(gm *Game) *Ghost {
	return newGhost(gm, blinkyAI{})
}

func (ai blinkyAI) ID() GhostID {
	return Blinky
}

func (ai blinkyAI) Start(gm *Game, g *Ghost) {
	g.state = Scatter
	g.dir = Left
	g.pos = Vector{14, 11}.ToPixel()
}

func (ai blinkyAI) House(_ *Game, g *Ghost) {
	g.target = Vector{14, 14}.ToPixel()
}

func (ai blinkyAI) Scatter(_ *Game, g *Ghost) {
	// TODO check dots remaining for Elroy 1 / 2
	g.target = Vector{GameWidth - 3, -3}.ToPixel()
}

func (ai blinkyAI) Chase(gm *Game, g *Ghost) {
	g.target = gm.player.pos
}

func (ai blinkyAI) Leave(_ *Game, _ *Ghost) bool {
	return true
}
