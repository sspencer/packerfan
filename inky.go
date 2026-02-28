package main

type inkyBehavior struct{}

func NewInky(g *Game) *Ghost {
	return &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    InHouse,
		behavior: inkyBehavior{},
	}
}

func (inkyBehavior) StartPos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyBehavior) HousePos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyBehavior) ScatterTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyBehavior) ChaseTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyBehavior) Leave(*Game) bool {
	return false
}
