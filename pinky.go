package main

type pinkyAI struct {
}

func NewPinky() *Ghost {
	return newGhost(Actor{
		ID:       Pinky,
		StartDir: Down,
		StartPos: Vector{14, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		startState:    InHouse,
		eatenTarget:   Vector{14, 14}.ToPixel().Sub(Half, 0),
		scatterTarget: Vector{2, -3}.ToPixel(),
		brain:         pinkyAI{},
	})
}

func (ai pinkyAI) chaseAI(_ *Ghost) Vector {
	return player.pos
}

func (ai pinkyAI) leaveAI(_ *Ghost) bool {
	return false /* TODO change to true */
}
