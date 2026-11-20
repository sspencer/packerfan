package component

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/config"
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
