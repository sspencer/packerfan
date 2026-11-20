package system

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/event"
)

type cachedPlayer struct {
	entry     *donburi.Entry
	entity    donburi.Entity
	actor     *component.ActorData
	transform *component.TransformData
	motion    *component.MotionData
	anim      *component.AnimationData
	input     *component.PlayerInputData
}

type cachedGhost struct {
	entry     *donburi.Entry
	entity    donburi.Entity
	actor     *component.ActorData
	transform *component.TransformData
	motion    *component.MotionData
	anim      *component.AnimationData
	ghost     *component.GhostData
}

// worldCache stores direct component pointers for the fixed entities of a
// single donburi.World, invalidated automatically whenever the world ID or
// entity count changes.
type worldCache struct {
	worldID   donburi.WorldId
	entityLen int
	valid     bool

	gs   *component.GameStateData
	maze *component.MazeData
	prng *component.PRNGData
	bus  *event.Bus

	players   []cachedPlayer
	ghosts    []cachedGhost
	blinkyIdx int
}

var (
	cachePlayerQuery = donburi.NewQuery(
		filter.Contains(
			component.PlayerTag,
			component.Actor,
			component.Transform,
			component.Motion,
			component.Animation,
			component.PlayerInput,
		),
	)
	cacheGhostQuery = donburi.NewQuery(
		filter.Contains(
			component.GhostTag,
			component.Actor,
			component.Transform,
			component.Motion,
			component.Animation,
			component.Ghost,
		),
	)
)

func (c *worldCache) ensure(w donburi.World) {
	if c.valid && c.worldID == w.Id() && c.entityLen == w.Len() {
		return
	}
	c.refresh(w)
}

func (c *worldCache) refresh(w donburi.World) {
	c.worldID = w.Id()
	c.entityLen = w.Len()
	c.valid = true

	if entry, ok := component.GameState.First(w); ok {
		c.gs = component.GameState.Get(entry)
	} else {
		c.gs = nil
	}
	if entry, ok := component.Maze.First(w); ok {
		c.maze = component.Maze.Get(entry)
	} else {
		c.maze = nil
	}
	if entry, ok := component.PRNG.First(w); ok {
		c.prng = component.PRNG.Get(entry)
	} else {
		c.prng = nil
	}
	c.bus = event.FindBus(w)

	c.players = c.players[:0]
	for entry := range cachePlayerQuery.Iter(w) {
		c.players = append(c.players, cachedPlayer{
			entry:     entry,
			entity:    entry.Entity(),
			actor:     component.Actor.Get(entry),
			transform: component.Transform.Get(entry),
			motion:    component.Motion.Get(entry),
			anim:      component.Animation.Get(entry),
			input:     component.PlayerInput.Get(entry),
		})
	}

	c.ghosts = c.ghosts[:0]
	c.blinkyIdx = -1
	for entry := range cacheGhostQuery.Iter(w) {
		actor := component.Actor.Get(entry)
		if actor.ID == component.Blinky {
			c.blinkyIdx = len(c.ghosts)
		}
		c.ghosts = append(c.ghosts, cachedGhost{
			entry:     entry,
			entity:    entry.Entity(),
			actor:     actor,
			transform: component.Transform.Get(entry),
			motion:    component.Motion.Get(entry),
			anim:      component.Animation.Get(entry),
			ghost:     component.Ghost.Get(entry),
		})
	}
}
