package main

func (g *Ghost) setTilePos(x, y int) {
	g.x = x * Tile
	g.y = y * Tile
}

func (g *Ghost) setTileTarget(x, y int) {
	g.targetX = x * Tile
	g.targetY = y * Tile
}

type Ghost struct {
	Actor
	targetX  int
	targetY  int
	state    GhostState
	reverse  bool
	behavior Behavior
}

type Behavior interface {
	ChaseTarget(*Game)
	ScatterTarget(*Game)
	Leave(*Game) bool
	HousePos(*Game)
	StartPos(*Game)
}

func (g *Ghost) ChaseTarget(game *Game) {
	if g.behavior == nil {
		return
	}

	g.behavior.ChaseTarget(game)
}

func (g *Ghost) ScatterTarget(game *Game) {
	if g.behavior == nil {
		return
	}

	g.behavior.ScatterTarget(game)
}

func (g *Ghost) StartPos(game *Game) {
	if g.behavior == nil {
		return
	}

	g.behavior.StartPos(game)
}

func (g *Ghost) HousePos(game *Game) {
	if g.behavior == nil {
		return
	}

	g.behavior.HousePos(game)
}

func (g *Ghost) Leave(game *Game) bool {
	if g.behavior == nil {
		return false
	}

	return g.behavior.Leave(game)
}
