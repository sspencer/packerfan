package main

type blinkyAI struct {
}

func NewBlinky() *Ghost {
	return newGhost(Actor{
		ID:       Blinky,
		StartDir: Left,
		StartPos: Vector{14, 11}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		StartState:    Scatter,
		HouseTarget:   Vector{14, 14}.ToPixel().Sub(Half, 0),
		ScatterTarget: Vector{GameWidth - 3, -3}.ToPixel(),
		Brain:         blinkyAI{},
	})
}

func (ai blinkyAI) ChaseTarget(_ *Ghost) Vector {
	return player.pos
}

func (ai blinkyAI) ShouldLeaveHouse(_ *Ghost) bool {
	return true
}
