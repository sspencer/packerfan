package component

import (
	"github.com/yohamta/donburi"

	"cheddarhead/internal/config"
)

// PlayerInputData buffers the player's desired movement direction.
type PlayerInputData struct {
	Requested config.Direction
	Active    bool
}

var (
	PlayerTag   = donburi.NewTag().SetName("Player")
	PlayerInput = donburi.NewComponentType[PlayerInputData]()
)

// ResetPlayer restores the player's transform, motion, animation, and buffered input to spawn defaults.
func ResetPlayer(entry *donburi.Entry) {
	if entry.HasComponent(Actor) {
		Actor.Get(entry).IsEaten = false
	}
	if entry.HasComponent(Motion) {
		motion := Motion.Get(entry)
		motion.SpeedIndex = 0
		motion.DotPause = 0
	}
	if entry.HasComponent(Animation) {
		anim := Animation.Get(entry)
		anim.FrameCount = 0
		anim.FrameIndex = 0
	}
	if entry.HasComponent(PlayerInput) {
		input := PlayerInput.Get(entry)
		input.Requested = Spawn.Get(entry).StartDir
		input.Active = false
	}
	ResetToStart(entry)
}
