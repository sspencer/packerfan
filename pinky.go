package main

type pinkyAI struct {
}

func NewPinky() *Ghost {
	return newGhost(Actor{
		ID:       Pinky,
		StartDir: Down,
		StartPos: Vector{14, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		StartState:    InHouse,
		HouseTarget:   Vector{14, 14}.ToPixel().Sub(Half, 0),
		ScatterTarget: Vector{2, -3}.ToPixel(),
		Brain:         pinkyAI{},
	})
}

func (ai pinkyAI) ChaseTarget(_ *Ghost) Vector {
	return player.pos
}

func (ai pinkyAI) ShouldLeaveHouse(_ *Ghost) bool {
	return true
}
