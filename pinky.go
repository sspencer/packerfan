package main

type pinkyAI struct {
}

func NewPinky(gm *Game) *Ghost {
	return newGhost(gm, Actor{
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

func (ai pinkyAI) ChaseTarget(gm *Game, _ *Ghost) Vector {
	return gm.player.pos
}

func (ai pinkyAI) ShouldLeaveHouse(_ *Game, _ *Ghost) bool {
	return true
}
