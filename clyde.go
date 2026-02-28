package main

type clydeAI struct{}

func NewClyde() *Ghost {
	return &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    InHouse,
		behavior: clydeAI{},
	}
}

func (clydeAI) StartPos(g *Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeAI) HousePos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeAI) ScatterTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeAI) ChaseTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (clydeAI) Leave(*Game) bool {
	return false
}
