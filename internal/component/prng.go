package component

import (
	"math/rand"

	"github.com/yohamta/donburi"

	"cheddarhead/internal/config"
)

const (
	ROMSize         = 8192
	DefaultPRNGSeed = 42
)

// PRNGData models the arcade game's ROM-indexed pseudo-random byte generator
// used by frightened ghosts and house exits, scoped to a single ECS World.
type PRNGData struct {
	ROM         [ROMSize]byte
	Index       uint
	seed        int64
	initialized bool
}

// NewPRNGData creates and seeds a deterministic simulated ROM table.
func NewPRNGData() PRNGData {
	p := PRNGData{}
	p.Init()
	return p
}

// Init fills the simulated ROM with reproducible pseudo-random bytes using DefaultPRNGSeed.
func (p *PRNGData) Init() {
	p.InitWithSeed(DefaultPRNGSeed)
}

// InitWithSeed fills the simulated ROM with reproducible pseudo-random bytes
// derived from seed and rewinds the cursor to 0. If the ROM is already filled
// for seed, it simply rewinds Index to 0.
func (p *PRNGData) InitWithSeed(seed int64) {
	if p.initialized && p.seed == seed {
		p.Index = 0
		return
	}
	r := rand.New(rand.NewSource(seed))
	for i := range p.ROM {
		p.ROM[i] = byte(r.Intn(256))
	}
	p.seed = seed
	p.initialized = true
	p.Index = 0
}

// Reset rewinds the PRNG cursor to the beginning of the ROM (e.g., on new level).
func (p *PRNGData) Reset() {
	p.Index = 0
}

// NextByte advances the arcade PRNG recurrence and returns the corresponding ROM byte.
func (p *PRNGData) NextByte() byte {
	p.Index = (p.Index*5 + 1) % ROMSize
	return p.ROM[p.Index]
}

// PreferredDirection maps the lowest 2 bits of the next PRNG byte to a direction:
// 0=Up, 1=Right, 2=Down, 3=Left.
func (p *PRNGData) PreferredDirection() config.Direction {
	switch p.NextByte() & 0x03 {
	case 0:
		return config.Up
	case 1:
		return config.Right
	case 2:
		return config.Down
	default:
		return config.Left
	}
}

var PRNG = donburi.NewComponentType[PRNGData]()

// MustFindPRNG returns the singleton PRNG component in the world.
func MustFindPRNG(w donburi.World) *PRNGData {
	return PRNG.Get(PRNG.MustFirst(w))
}
