package main

type inkyAI struct {
}

func NewInky() *Ghost {
	return newGhost(Actor{
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

func (ai inkyAI) ChaseTarget(_ *Ghost) Vector {
	return player.pos
}

func (ai inkyAI) ShouldLeaveHouse(_ *Ghost) bool {
	return false
}
