package main

type Ghost struct {
	Actor
	target   Vector
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
