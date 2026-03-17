package main

type blinkyAI struct {
}

func NewBlinky() *Ghost {
	return newGhost(Actor{
		ID:       Blinky,
		StartDir: Left,
		StartPos: Vector{14, 11}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		startState:    Scatter,
		eatenTarget:   Vector{14, 14}.ToPixel().Sub(Half, 0),
		scatterTarget: Vector{GameWidth - 3, -3}.ToPixel(),
		brain:         blinkyAI{},
	})
}

func (ai blinkyAI) chaseAI(_ *Ghost) Vector {
	return player.pos
}

func (ai blinkyAI) leaveAI(_ *Ghost) bool {
	return true
}
