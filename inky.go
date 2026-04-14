package main

type inkyAI struct {
}

func NewInky() *Ghost {
	return newGhost(Actor{
		ID:       Inky,
		StartDir: Up,
		StartPos: Vector{12, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		startState:    InHouse,
		eatenTarget:   Vector{12, 14}.ToPixel().Sub(Half, 0),
		scatterTarget: Vector{GameWidth - 1, GameHeight}.ToPixel(),
		brain:         inkyAI{},
	})
}

func (ai inkyAI) chaseAI(_ *Ghost) Vector {
	return player.pos
}

func (ai inkyAI) leaveAI(_ *Ghost) bool {
	if game.level == 1 {
		return game.dotsEaten > 30 || game.frameCount > secondsToFrames(7)
	}

	return true
}
