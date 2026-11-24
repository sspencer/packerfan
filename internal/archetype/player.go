package archetype

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
)

// SpawnPlayer creates the Pac-Man / Ms. Pac-Man player entity with all required components.
func SpawnPlayer(w donburi.World) *donburi.Entry {
	entity := w.Create(
		component.PlayerTag,
		component.Actor,
		component.Transform,
		component.Spawn,
		component.Motion,
		component.Animation,
		component.PlayerInput,
	)
	entry := w.Entry(entity)

	startPos := config.Vector{X: 108, Y: 184}
	startDir := config.Left

	component.Actor.SetValue(entry, component.ActorData{
		ID: component.Pacman,
	})
	component.Spawn.SetValue(entry, component.SpawnData{
		StartPos: startPos,
		StartDir: startDir,
	})
	component.Transform.SetValue(entry, component.TransformData{
		Pos: startPos,
		Dir: startDir,
	})
	component.Motion.SetValue(entry, component.MotionData{
		CornerSnap: true,
	})
	component.Animation.SetValue(entry, component.AnimationData{})
	component.PlayerInput.SetValue(entry, component.PlayerInputData{
		Requested: startDir,
	})

	return entry
}
