package main

type clydeAI struct {
}

func NewClyde(gm *Game) *Ghost {
	return newGhost(gm, GhostConfig{
		ID:            Clyde,
		StartState:    InHouse,
		StartDir:      Up,
		StartPos:      Vector{16, 14}.ToPixel().Sub(Half, 0),
		HouseTarget:   Vector{16, 14}.ToPixel().Sub(Half, 0),
		ScatterTarget: Vector{0, GameHeight}.ToPixel(),
		Brain:         clydeAI{},
	})
}

func (ai clydeAI) ChaseTarget(gm *Game, _ *Ghost) Vector {
	return gm.player.pos
}

func (ai clydeAI) ShouldLeaveHouse(_ *Game, _ *Ghost) bool {
	return false
}
