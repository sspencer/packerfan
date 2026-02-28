package main

type pinkyAI struct{}

func NewPinky() *Ghost {
	return &Ghost{
		Actor: Actor{
			isGhost: true,
		},
		state:    InHouse,
		behavior: pinkyAI{},
	}
}

func (p pinkyAI) StartPos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyAI) HousePos(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyAI) ScatterTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyAI) ChaseTarget(*Game) {
	//TODO implement me
	panic("implement me")
}

func (p pinkyAI) Leave(*Game) bool {
	return false
}
