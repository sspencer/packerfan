package config

//go:generate stringer -type=GhostState
type GhostState int

const (
	Scatter GhostState = iota
	Chase
	Frightened
	Eaten
	EnteringHouse
	InHouse
	LeavingHouse
)
