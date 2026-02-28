package main

type clydeBehavior struct{}

func NewClyde(g *Game) *Ghost {
	return &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    InHouse,
		behavior: clydeBehavior{},
	}
}

func (clydeBehavior) StartPos(g *Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeBehavior) HousePos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeBehavior) ScatterTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeBehavior) ChaseTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeBehavior) Leave(*Game) bool {
	return false
}
