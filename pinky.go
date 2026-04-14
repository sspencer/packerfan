package main

type pinkyAI struct {
}

func NewPinky() *Ghost {
	return newGhost(Actor{
		ID:       Pinky,
		StartDir: Down,
		StartPos: Vector{14, 14}.ToPixel().Sub(Half, 0),
	}, GhostConfig{
		startState:    InHouse,
		eatenTarget:   Vector{14, 14}.ToPixel().Sub(Half, 0),
		scatterTarget: Vector{2, -3}.ToPixel(),
		brain:         pinkyAI{},
	})
}

func (ai pinkyAI) chaseAI(_ *Ghost) Vector {
	vx, vy := 0, 0

	// Reproduce original Pinky chase bug for UP.
	// To remove the bug, use vx = 0 when facing Up.
	switch player.dir {
	case Up:
		vx, vy = -4, -4
	case Down:
		vy = 4
	case Right:
		vx = 4
	default: // Left
		vx = -4
	}

	playerTile := player.pos.ToTile()
	targetTile := Vector{playerTile.x + vx, playerTile.y + vy}
	return targetTile.ToPixel()
}

func (ai pinkyAI) leaveAI(_ *Ghost) bool {
	return true
}
