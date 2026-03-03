package main

type clydeAI struct {
}

func NewClyde(gm *Game) *Ghost {
	return newGhost(gm, clydeAI{})
}

func (ai clydeAI) ID() GhostID {
	return Clyde
}

func (ai clydeAI) Start(gm *Game, g *Ghost) {
	g.state = InHouse
	g.dir = Up
	g.pos = Vector{16, 14}.ToPixel().Sub(Half, 0)
}

func (ai clydeAI) House(_ *Game, g *Ghost) {
	g.target = Vector{16, 14}.ToPixel().Sub(Half, 0)
}

func (ai clydeAI) Scatter(_ *Game, g *Ghost) {
	g.target = Vector{0, GameHeight}.ToPixel()
}

func (ai clydeAI) Chase(gm *Game, g *Ghost) {
	g.target = gm.player.pos
}

func (ai clydeAI) Leave(_ *Game, _ *Ghost) bool {
	return false
}
