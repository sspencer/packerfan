package main

type blinkyAI struct {
}

func NewBlinky(gm *Game) *Ghost {
	return newGhost(gm, Actor{
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

func (ai blinkyAI) ChaseTarget(gm *Game, _ *Ghost) Vector {
	return gm.player.pos
}

func (ai blinkyAI) ShouldLeaveHouse(_ *Game, _ *Ghost) bool {
	return true
}
