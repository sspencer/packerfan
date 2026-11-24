package sim

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/event"
)

// EventMask is a bitmask of notable gameplay events that occurred during a Step.
type EventMask uint8

const (
	EventDot EventMask = 1 << iota
	EventPower
	EventGhost
	EventFruit
	EventExtraLife
	EventDeath
	EventLevelClear
)

// Has reports whether all bits in flag are set in m.
func (m EventMask) Has(flag EventMask) bool {
	return m&flag == flag
}

// Strings returns the canonical wire names of the events set in m.
func (m EventMask) Strings() []string {
	if m == 0 {
		return nil
	}
	out := make([]string, 0, 4)
	if m.Has(EventDot) {
		out = append(out, "dot")
	}
	if m.Has(EventPower) {
		out = append(out, "power")
	}
	if m.Has(EventGhost) {
		out = append(out, "ghost")
	}
	if m.Has(EventFruit) {
		out = append(out, "fruit")
	}
	if m.Has(EventExtraLife) {
		out = append(out, "extra_life")
	}
	if m.Has(EventDeath) {
		out = append(out, "death")
	}
	if m.Has(EventLevelClear) {
		out = append(out, "level_clear")
	}
	return out
}

// TileCoord identifies a tile position (X column, Y row) on the 28x31 maze grid.
type TileCoord struct {
	X int
	Y int
}

// eventCollector subscribes to gameplay events on a donburi.World and accumulates
// them into reusable per-step buffers.
type eventCollector struct {
	mask  EventMask
	eaten []TileCoord
}

func (c *eventCollector) reset() {
	c.mask = 0
	c.eaten = c.eaten[:0]
}

func (c *eventCollector) register(w donburi.World) {
	event.DotEaten.Subscribe(w, func(_ donburi.World, ev event.DotEatenEvent) {
		c.mask |= EventDot
		c.eaten = append(c.eaten, TileCoord{X: ev.X, Y: ev.Y})
	})
	event.PowerUpEaten.Subscribe(w, func(_ donburi.World, ev event.PowerUpEatenEvent) {
		c.mask |= EventPower
		c.eaten = append(c.eaten, TileCoord{X: ev.X, Y: ev.Y})
	})
	event.GhostEaten.Subscribe(w, func(_ donburi.World, _ event.GhostEatenEvent) {
		c.mask |= EventGhost
	})
	event.FruitEaten.Subscribe(w, func(_ donburi.World, _ event.FruitEatenEvent) {
		c.mask |= EventFruit
	})
	event.ExtraLifeAwarded.Subscribe(w, func(_ donburi.World, _ event.ExtraLifeAwardedEvent) {
		c.mask |= EventExtraLife
	})
	event.PlayerEaten.Subscribe(w, func(_ donburi.World, _ event.PlayerEatenEvent) {
		c.mask |= EventDeath
	})
	event.LevelCompleted.Subscribe(w, func(_ donburi.World, _ event.LevelCompletedEvent) {
		c.mask |= EventLevelClear
	})
}
