package main

type clydeAI struct {
}

func NewClyde() *Ghost {
	return newGhost(Actor{
		ID:       Clyde,
		StartDir: Up,
		StartPos: Vector{16, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		startState:    InHouse,
		eatenTarget:   Vector{16, 14}.ToPixel().Sub(Half, 0),
		scatterTarget: Vector{0, GameHeight}.ToPixel(),
		brain:         clydeAI{},
	})
}

func (ai clydeAI) chaseAI(_ *Ghost) Vector {
	return player.pos
}

func (ai clydeAI) leaveAI(_ *Ghost) bool {
	if game.level == 1 {
		return game.dotsEaten > 90 || game.frameCount > secondsToFrames(15)
	} else if game.level == 2 {
		return game.dotsEaten > 50 || game.frameCount > secondsToFrames(10)
	}

	return true
}
