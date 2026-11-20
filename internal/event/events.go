package event

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/features/events"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

type DotEatenEvent struct {
	X int
	Y int
}

type PowerUpEatenEvent struct {
	X int
	Y int
}

type GhostEatenEvent struct {
	Entity donburi.Entity
	ID     component.ActorID
	X      int
	Y      int
}

type PlayerEatenEvent struct {
	Entity donburi.Entity
	ID     component.ActorID
	X      int
	Y      int
}

type GhostStateChangedEvent struct {
	State config.GhostState
}

type FrightModeEndedEvent struct{}

type LevelCompletedEvent struct{}

type FruitEatenEvent struct {
	Points int
}

type ExtraLifeAwardedEvent struct{}

var (
	DotEaten          = events.NewEventType[DotEatenEvent]()
	PowerUpEaten      = events.NewEventType[PowerUpEatenEvent]()
	GhostEaten        = events.NewEventType[GhostEatenEvent]()
	PlayerEaten       = events.NewEventType[PlayerEatenEvent]()
	GhostStateChanged = events.NewEventType[GhostStateChangedEvent]()
	FrightModeEnded   = events.NewEventType[FrightModeEndedEvent]()
	LevelCompleted    = events.NewEventType[LevelCompletedEvent]()
	FruitEaten        = events.NewEventType[FruitEatenEvent]()
	ExtraLifeAwarded  = events.NewEventType[ExtraLifeAwardedEvent]()
)
