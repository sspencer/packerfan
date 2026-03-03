package main

type pinkyAI struct {
}

func NewPinky(gm *Game) *Ghost {
	return newGhost(gm, pinkyAI{})
}

func (ai pinkyAI) ID() GhostID {
	return Pinky
}

func (ai pinkyAI) Start(gm *Game, g *Ghost) {
	g.state = InHouse
	g.dir = Down
	g.pos = Vector{14, 14}.ToPixel().Sub(Half, 0)
}

func (ai pinkyAI) House(_ *Game, g *Ghost) {
	g.target = Vector{14, 14}.ToPixel().Sub(Half, 0)
}

func (ai pinkyAI) Scatter(_ *Game, g *Ghost) {
	g.target = Vector{2, -3}.ToPixel()
}

func (ai pinkyAI) Chase(gm *Game, g *Ghost) {
	g.target = gm.player.pos
}

func (ai pinkyAI) Leave(_ *Game, _ *Ghost) bool {
	return true
}
