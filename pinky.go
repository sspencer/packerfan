package main

type pinkyBehavior struct{}

func NewPinky(g *Game) *Ghost {
	return &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    InHouse,
		behavior: pinkyBehavior{},
	}
}

func (p pinkyBehavior) StartPos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyBehavior) HousePos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyBehavior) ScatterTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyBehavior) ChaseTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyBehavior) Leave(*Game) bool {
	return false
}
