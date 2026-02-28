package main

type inkyAI struct{}

func NewInky() *Ghost {
	return &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    InHouse,
		behavior: inkyAI{},
	}
}

func (inkyAI) StartPos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyAI) HousePos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyAI) ScatterTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyAI) ChaseTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (inkyAI) Leave(*Game) bool {
	return false
}
