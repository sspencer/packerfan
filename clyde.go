package main

type clydeAI struct {
}

func NewClyde() *Ghost {
	return newGhost(Actor{
		ID:       Clyde,
		StartDir: Up,
		StartPos: Vector{16, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		StartState:    InHouse,
		HouseTarget:   Vector{16, 14}.ToPixel().Sub(Half, 0),
		ScatterTarget: Vector{0, GameHeight}.ToPixel(),
		Brain:         clydeAI{},
	})
}

func (ai clydeAI) ChaseTarget(_ *Ghost) Vector {
	return player.pos
}

func (ai clydeAI) ShouldLeaveHouse(_ *Ghost) bool {
	return false
}
