package archetype

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

type ghostSpec struct {
	id            component.ActorID
	aiKind        component.GhostAIKind
	startDir      config.Direction
	startPos      config.Vector
	startState    config.GhostState
	eatenTarget   config.Vector
	scatterTarget config.Vector
	spriteRow     int
}

func spawnGhost(w donburi.World, spec ghostSpec) *donburi.Entry {
	entity := w.Create(
		component.GhostTag,
		component.Actor,
		component.Transform,
		component.Spawn,
		component.Motion,
		component.Animation,
		component.Ghost,
	)
	entry := w.Entry(entity)

	component.Actor.SetValue(entry, component.ActorData{
		ID: spec.id,
	})
	component.Spawn.SetValue(entry, component.SpawnData{
		StartPos: spec.startPos,
		StartDir: spec.startDir,
	})
	component.Transform.SetValue(entry, component.TransformData{
		Pos: spec.startPos,
		Dir: spec.startDir,
	})
	component.Motion.SetValue(entry, component.MotionData{
		CornerSnap: false,
	})
	component.Animation.SetValue(entry, component.AnimationData{})
	component.Ghost.SetValue(entry, component.GhostData{
		AIKind:        spec.aiKind,
		State:         spec.startState,
		StartState:    spec.startState,
		ScatterTarget: spec.scatterTarget,
		EatenTarget:   spec.eatenTarget,
		SpriteRow:     spec.spriteRow,
	})

	return entry
}

// SpawnBlinky creates Blinky (red ghost) outside the ghost house in Scatter mode.
func SpawnBlinky(w donburi.World) *donburi.Entry {
	return spawnGhost(w, ghostSpec{
		id:            component.Blinky,
		aiKind:        component.BlinkyAI,
		startDir:      config.Left,
		startPos:      config.Vector{X: 14, Y: 11}.ToPixel().Sub(config.Half, 0),
		startState:    config.Scatter,
		eatenTarget:   config.Vector{X: 14, Y: 14}.ToPixel().Sub(config.Half, 0),
		scatterTarget: config.Vector{X: config.GameWidth, Y: -3}.ToPixel(),
		spriteRow:     64,
	})
}

// SpawnInky creates Inky (cyan ghost) inside the ghost house.
func SpawnInky(w donburi.World) *donburi.Entry {
	return spawnGhost(w, ghostSpec{
		id:            component.Inky,
		aiKind:        component.InkyAI,
		startDir:      config.Up,
		startPos:      config.Vector{X: 12, Y: 14}.ToPixel().Sub(config.Half, 0),
		startState:    config.InHouse,
		eatenTarget:   config.Vector{X: 12, Y: 14}.ToPixel().Sub(config.Half, 0),
		scatterTarget: config.Vector{X: config.GameWidth - 1, Y: config.GameHeight}.ToPixel(),
		spriteRow:     96,
	})
}

// SpawnPinky creates Pinky (pink ghost) inside the ghost house.
func SpawnPinky(w donburi.World) *donburi.Entry {
	return spawnGhost(w, ghostSpec{
		id:            component.Pinky,
		aiKind:        component.PinkyAI,
		startDir:      config.Down,
		startPos:      config.Vector{X: 14, Y: 14}.ToPixel().Sub(config.Half, 0),
		startState:    config.InHouse,
		eatenTarget:   config.Vector{X: 14, Y: 14}.ToPixel().Sub(config.Half, 0),
		scatterTarget: config.Vector{X: 2, Y: -3}.ToPixel(),
		spriteRow:     80,
	})
}

// SpawnClyde creates Clyde (orange ghost) inside the ghost house.
func SpawnClyde(w donburi.World) *donburi.Entry {
	return spawnGhost(w, ghostSpec{
		id:            component.Clyde,
		aiKind:        component.ClydeAI,
		startDir:      config.Up,
		startPos:      config.Vector{X: 16, Y: 14}.ToPixel().Sub(config.Half, 0),
		startState:    config.InHouse,
		eatenTarget:   config.Vector{X: 16, Y: 14}.ToPixel().Sub(config.Half, 0),
		scatterTarget: config.Vector{X: 0, Y: config.GameHeight}.ToPixel(),
		spriteRow:     112,
	})
}

// SpawnAllGhosts spawns Blinky, Inky, Pinky, and Clyde in canonical update/draw order.
func SpawnAllGhosts(w donburi.World) []*donburi.Entry {
	return []*donburi.Entry{
		SpawnBlinky(w),
		SpawnInky(w),
		SpawnPinky(w),
		SpawnClyde(w),
	}
}
