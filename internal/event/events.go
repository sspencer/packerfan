package event

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
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

type FruitBouncedEvent struct{}

type ExtraLifeAwardedEvent struct{}

const (
	bitDotEaten uint16 = 1 << iota
	bitPowerUpEaten
	bitGhostEaten
	bitPlayerEaten
	bitGhostStateChanged
	bitFrightModeEnded
	bitLevelCompleted
	bitFruitEaten
	bitFruitBounced
	bitExtraLifeAwarded
)

type queue[T any] struct {
	subs    []func(donburi.World, T)
	pending []T
	scratch []T
}

func (q *queue[T]) subscribe(fn func(donburi.World, T)) {
	q.subs = append(q.subs, fn)
}

func (q *queue[T]) publish(ev T) {
	q.pending = append(q.pending, ev)
}

func (q *queue[T]) drain(w donburi.World) {
	if len(q.pending) == 0 {
		return
	}
	q.scratch, q.pending = q.pending, q.scratch[:0]
	for _, ev := range q.scratch {
		for _, sub := range q.subs {
			sub(w, ev)
		}
	}
	q.scratch = q.scratch[:0]
}

// Bus is a per-world typed event bus stored as a singleton Donburi component.
// Checking HasPending() is a single integer comparison with zero map lookups,
// queries, locks, or heap allocations.
type Bus struct {
	pendingMask       uint16
	dotEaten          queue[DotEatenEvent]
	powerUpEaten      queue[PowerUpEatenEvent]
	ghostEaten        queue[GhostEatenEvent]
	playerEaten       queue[PlayerEatenEvent]
	ghostStateChanged queue[GhostStateChangedEvent]
	frightModeEnded   queue[FrightModeEndedEvent]
	levelCompleted    queue[LevelCompletedEvent]
	fruitEaten        queue[FruitEatenEvent]
	fruitBounced      queue[FruitBouncedEvent]
	extraLifeAwarded  queue[ExtraLifeAwardedEvent]
}

var busComponent = donburi.NewComponentType[Bus]()

// FindBus returns the world's singleton event Bus, or nil if no events have
// been subscribed or published on w yet.
func FindBus(w donburi.World) *Bus {
	entry, ok := busComponent.First(w)
	if !ok {
		return nil
	}
	return busComponent.Get(entry)
}

// EnsureBus returns the world's singleton event Bus, creating it if absent.
func EnsureBus(w donburi.World) *Bus {
	if entry, ok := busComponent.First(w); ok {
		return busComponent.Get(entry)
	}
	entry := w.Entry(w.Create(busComponent))
	return busComponent.Get(entry)
}

// HasPending reports whether any event is queued for processing.
func (b *Bus) HasPending() bool {
	return b != nil && b.pendingMask != 0
}

// ProcessAll dispatches one generation of queued events to their subscribers.
// Events published by subscribers during dispatch remain queued for the next call.
func (b *Bus) ProcessAll(w donburi.World) {
	if b == nil || b.pendingMask == 0 {
		return
	}
	mask := b.pendingMask
	b.pendingMask = 0

	if mask&bitDotEaten != 0 {
		b.dotEaten.drain(w)
	}
	if mask&bitPowerUpEaten != 0 {
		b.powerUpEaten.drain(w)
	}
	if mask&bitGhostEaten != 0 {
		b.ghostEaten.drain(w)
	}
	if mask&bitPlayerEaten != 0 {
		b.playerEaten.drain(w)
	}
	if mask&bitGhostStateChanged != 0 {
		b.ghostStateChanged.drain(w)
	}
	if mask&bitFrightModeEnded != 0 {
		b.frightModeEnded.drain(w)
	}
	if mask&bitLevelCompleted != 0 {
		b.levelCompleted.drain(w)
	}
	if mask&bitFruitEaten != 0 {
		b.fruitEaten.drain(w)
	}
	if mask&bitFruitBounced != 0 {
		b.fruitBounced.drain(w)
	}
	if mask&bitExtraLifeAwarded != 0 {
		b.extraLifeAwarded.drain(w)
	}
}

// ProcessAllEvents dispatches one generation of queued events on w.
func ProcessAllEvents(w donburi.World) {
	if bus := FindBus(w); bus != nil {
		bus.ProcessAll(w)
	}
}

// EventType provides typed Subscribe/Publish methods backed by the world's Bus.
type EventType[T any] struct {
	bit   uint16
	queue func(*Bus) *queue[T]
}

// Subscribe registers a handler for this event type on w.
func (e EventType[T]) Subscribe(w donburi.World, fn func(donburi.World, T)) {
	bus := EnsureBus(w)
	e.queue(bus).subscribe(fn)
}

// Publish enqueues ev on w for dispatch during the next ProcessAll call.
func (e EventType[T]) Publish(w donburi.World, ev T) {
	bus := EnsureBus(w)
	e.queue(bus).publish(ev)
	bus.pendingMask |= e.bit
}

// PublishTo enqueues ev directly onto a cached Bus pointer without querying w.
func (e EventType[T]) PublishTo(bus *Bus, ev T) {
	e.queue(bus).publish(ev)
	bus.pendingMask |= e.bit
}

var (
	DotEaten          = EventType[DotEatenEvent]{bit: bitDotEaten, queue: func(b *Bus) *queue[DotEatenEvent] { return &b.dotEaten }}
	PowerUpEaten      = EventType[PowerUpEatenEvent]{bit: bitPowerUpEaten, queue: func(b *Bus) *queue[PowerUpEatenEvent] { return &b.powerUpEaten }}
	GhostEaten        = EventType[GhostEatenEvent]{bit: bitGhostEaten, queue: func(b *Bus) *queue[GhostEatenEvent] { return &b.ghostEaten }}
	PlayerEaten       = EventType[PlayerEatenEvent]{bit: bitPlayerEaten, queue: func(b *Bus) *queue[PlayerEatenEvent] { return &b.playerEaten }}
	GhostStateChanged = EventType[GhostStateChangedEvent]{bit: bitGhostStateChanged, queue: func(b *Bus) *queue[GhostStateChangedEvent] { return &b.ghostStateChanged }}
	FrightModeEnded   = EventType[FrightModeEndedEvent]{bit: bitFrightModeEnded, queue: func(b *Bus) *queue[FrightModeEndedEvent] { return &b.frightModeEnded }}
	LevelCompleted    = EventType[LevelCompletedEvent]{bit: bitLevelCompleted, queue: func(b *Bus) *queue[LevelCompletedEvent] { return &b.levelCompleted }}
	FruitEaten        = EventType[FruitEatenEvent]{bit: bitFruitEaten, queue: func(b *Bus) *queue[FruitEatenEvent] { return &b.fruitEaten }}
	FruitBounced      = EventType[FruitBouncedEvent]{bit: bitFruitBounced, queue: func(b *Bus) *queue[FruitBouncedEvent] { return &b.fruitBounced }}
	ExtraLifeAwarded  = EventType[ExtraLifeAwardedEvent]{bit: bitExtraLifeAwarded, queue: func(b *Bus) *queue[ExtraLifeAwardedEvent] { return &b.extraLifeAwarded }}
)
