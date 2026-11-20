package component

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/config"
)

// GhostAIKind selects which classic arcade targeting & house-exit rule a ghost uses.
type GhostAIKind int

const (
	BlinkyAI GhostAIKind = iota
	PinkyAI
	InkyAI
	ClydeAI
)

// GhostData stores ghost-specific state, navigation targets, and sprite metadata.
type GhostData struct {
	AIKind            GhostAIKind
	State             config.GhostState
	StartState        config.GhostState
	Target            config.Vector
	ScatterTarget     config.Vector
	EatenTarget       config.Vector
	PixelsMoved       int
	Reverse           bool
	FrightenedInHouse bool
	NextDir           config.Direction
	PlannedTile       config.Vector
	HasNextDir        bool
	DotCounter        int
	SpriteRow         int
}

var (
	GhostTag = donburi.NewTag().SetName("Ghost")
	Ghost    = donburi.NewComponentType[GhostData]()
)

// ResetGhost restores a ghost's transform, state, and movement counters to spawn defaults.
func ResetGhost(entry *donburi.Entry) {
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
	ghost := Ghost.Get(entry)
	ghost.State = ghost.StartState
	ghost.Target = config.Vector{}
	ghost.PixelsMoved = config.Tile
	ghost.Reverse = false
	ghost.FrightenedInHouse = false
	ghost.HasNextDir = false
	ghost.PlannedTile = config.Vector{}
	ResetToStart(entry)
}
