package main

type inkyAI struct {
}

func NewInky(gm *Game) *Ghost {
	return newGhost(gm, Actor{
		ID:       Inky,
		StartDir: Up,
		StartPos: Vector{12, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		StartState:    InHouse,
		HouseTarget:   Vector{12, 14}.ToPixel().Sub(Half, 0),
		ScatterTarget: Vector{GameWidth - 3, -3}.ToPixel(),
		Brain:         inkyAI{},
	})
}

func (ai inkyAI) ChaseTarget(gm *Game, _ *Ghost) Vector {
	return gm.player.pos
}

func (ai inkyAI) ShouldLeaveHouse(_ *Game, _ *Ghost) bool {
	return false
}
