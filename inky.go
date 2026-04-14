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
	vx, vy := 0, 0

	// Reproduce original Inky chase bug for UP.
	// To remove the bug, use vx = 0 when facing Up.
	switch player.dir {
	case Up:
		vx, vy = -2, -2
	case Down:
		vy = 2
	case Right:
		vx = 2
	default: // Left
		vx = -2
	}

	playerTile := player.pos.ToTile()
	pivot := Vector{playerTile.x + vx, playerTile.y + vy}
	blinkyTile := blinky.pos.ToTile() // yes, Blinky
	dx := pivot.x - blinkyTile.x
	dy := pivot.y - blinkyTile.y
	targetTile := Vector{blinkyTile.x + 2*dx, blinkyTile.y + 2*dy}
	return targetTile.ToPixel()
}

func (ai inkyAI) leaveAI(_ *Ghost) bool {
	if game.level == 1 {
		return game.dotsEaten > 30 || game.frameCount > secondsToFrames(7)
	}

	return true
}
