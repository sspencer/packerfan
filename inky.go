package main

type inkyAI struct {
}

func NewInky(gm *Game) *Ghost {
	return newGhost(gm, inkyAI{})
}

func (ai inkyAI) ID() GhostID {
	return Inky
}

func (ai inkyAI) Start(gm *Game, g *Ghost) {
	g.state = InHouse
	g.dir = Up
	g.pos = Vector{12, 14}.ToPixel().Sub(Half, 0)
}

func (ai inkyAI) House(_ *Game, g *Ghost) {
	g.target = Vector{12, 14}.ToPixel().Sub(Half, 0)
}

func (ai inkyAI) Scatter(_ *Game, g *Ghost) {
	g.target = Vector{GameWidth - 3, -3}.ToPixel()
}

func (ai inkyAI) Chase(gm *Game, g *Ghost) {
	g.target = gm.player.pos
}

func (ai inkyAI) Leave(_ *Game, _ *Ghost) bool {
	return false
}
