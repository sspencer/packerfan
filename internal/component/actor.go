package component

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/config"
)

//go:generate stringer -type=ActorID
type ActorID int

const (
	Blinky ActorID = iota
	Pinky
	Inky
	Clyde
	Pacman
)

// ActorData identifies a playable or AI-controlled character in the maze.
type ActorData struct {
	ID      ActorID
	IsEaten bool
}

// TransformData stores an entity's pixel position and facing direction.
type TransformData struct {
	Pos config.Vector
	Dir config.Direction
}

// SpawnData records the initial position and direction used when resetting an actor.
type SpawnData struct {
	StartPos config.Vector
	StartDir config.Direction
}

// MotionData stores bit-mask speed pacing and dot-eating pause state.
type MotionData struct {
	SpeedIndex int
	DotPause   int
	CornerSnap bool
}

// AnimationData tracks sprite frame cycling for an animated entity.
type AnimationData struct {
	FrameCount int
	FrameIndex int
}

var (
	Actor     = donburi.NewComponentType[ActorData]()
	Transform = donburi.NewComponentType[TransformData]()
	Spawn     = donburi.NewComponentType[SpawnData]()
	Motion    = donburi.NewComponentType[MotionData]()
	Animation = donburi.NewComponentType[AnimationData]()
)

// ResetToStart restores an entity's Transform from its Spawn component.
func ResetToStart(entry *donburi.Entry) {
	spawn := Spawn.Get(entry)
	transform := Transform.Get(entry)
	transform.Pos = spawn.StartPos
	transform.Dir = spawn.StartDir
}
