package audio

import (
	"io/fs"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"

	"pacfan/internal/component"
	"pacfan/internal/config"
)

// SoundID identifies a one-shot arcade sound effect or cutscene jingle.
type SoundID int

const (
	SoundStart SoundID = iota
	SoundCredit
	SoundChomp
	SoundEatGhost
	SoundEatFruit
	SoundExtraLife
	SoundDeath
	SoundIntermission1
	SoundIntermission2
	SoundIntermission3
)

// LoopID identifies a continuous background arcade audio loop.
type LoopID int

const (
	LoopNone LoopID = iota
	LoopSiren0
	LoopSiren1
	LoopSiren2
	LoopSiren3
	LoopSiren4
	LoopFright
	LoopEyes
)

// Player abstracts audio playback so the ECS simulation and scenes remain
// completely decoupled from Raylib and run headlessly in tests.
type Player interface {
	Play(id SoundID)
	SetLoop(loop LoopID)
	SetPaused(paused bool)
	SetMuted(muted bool)
	Muted() bool
	StopAll()
	Unload()
}

// NoopPlayer silently ignores all playback calls (used in tests and -mute mode).
type NoopPlayer struct{}

func (NoopPlayer) Play(SoundID)   {}
func (NoopPlayer) SetLoop(LoopID) {}
func (NoopPlayer) SetPaused(bool) {}
func (NoopPlayer) SetMuted(bool)  {}
func (NoopPlayer) Muted() bool    { return true }
func (NoopPlayer) StopAll()       {}
func (NoopPlayer) Unload()        {}

// SoundPaths returns the ordered candidate asset paths for a one-shot sound.
// The first path found in the embedded filesystem is loaded.
func SoundPaths(pacmanMode bool, id SoundID) []string {
	if pacmanMode {
		switch id {
		case SoundStart:
			return []string{"assets/pacman/beginning.wav", "assets/ms-pacman/start.wav"}
		case SoundCredit:
			return []string{"assets/ms-pacman/credit.wav"}
		case SoundChomp:
			return []string{"assets/pacman/chomp.wav", "assets/ms-pacman/eat_dot.wav"}
		case SoundEatGhost:
			return []string{"assets/pacman/eatghost.wav", "assets/ms-pacman/eat_ghost.wav"}
		case SoundEatFruit:
			return []string{"assets/pacman/eatfruit.wav", "assets/ms-pacman/eat_fruit.wav"}
		case SoundExtraLife:
			return []string{"assets/pacman/extrapac.wav", "assets/ms-pacman/extend.wav"}
		case SoundDeath:
			return []string{"assets/pacman/death.wav", "assets/ms-pacman/death.wav"}
		case SoundIntermission1, SoundIntermission2, SoundIntermission3:
			return []string{"assets/pacman/intermission.wav", "assets/ms-pacman/intermission1.wav"}
		}
		return nil
	}

	switch id {
	case SoundStart:
		return []string{"assets/ms-pacman/start.wav", "assets/pacman/beginning.wav"}
	case SoundCredit:
		return []string{"assets/ms-pacman/credit.wav"}
	case SoundChomp:
		return []string{"assets/ms-pacman/eat_dot.wav", "assets/pacman/chomp.wav"}
	case SoundEatGhost:
		return []string{"assets/ms-pacman/eat_ghost.wav", "assets/pacman/eatghost.wav"}
	case SoundEatFruit:
		return []string{"assets/ms-pacman/eat_fruit.wav", "assets/pacman/eatfruit.wav"}
	case SoundExtraLife:
		return []string{"assets/ms-pacman/extend.wav", "assets/pacman/extrapac.wav"}
	case SoundDeath:
		return []string{"assets/ms-pacman/death.wav", "assets/pacman/death.wav"}
	case SoundIntermission1:
		return []string{"assets/ms-pacman/intermission1.wav", "assets/pacman/intermission.wav"}
	case SoundIntermission2:
		return []string{"assets/ms-pacman/intermission2.wav", "assets/pacman/intermission.wav"}
	case SoundIntermission3:
		return []string{"assets/ms-pacman/intermission3.wav", "assets/pacman/intermission.wav"}
	}
	return nil
}

// LoopPaths returns the ordered candidate asset paths for a background loop.
func LoopPaths(loop LoopID) []string {
	switch loop {
	case LoopSiren0:
		return []string{"assets/ms-pacman/siren0.wav"}
	case LoopSiren1:
		return []string{"assets/ms-pacman/siren1.wav"}
	case LoopSiren2:
		return []string{"assets/ms-pacman/siren2.wav"}
	case LoopSiren3:
		return []string{"assets/ms-pacman/siren3.wav"}
	case LoopSiren4:
		return []string{"assets/ms-pacman/siren4.wav"}
	case LoopFright:
		return []string{"assets/ms-pacman/fright.wav"}
	case LoopEyes:
		return []string{"assets/ms-pacman/eyes.wav"}
	}
	return nil
}

// LoopFirstPaths returns the optional intro clip paths played once when
// transitioning into a background loop before seamlessly repeating LoopPaths.
func LoopFirstPaths(loop LoopID) []string {
	switch loop {
	case LoopSiren0:
		return []string{"assets/ms-pacman/siren0_firstloop.wav"}
	case LoopSiren1:
		return []string{"assets/ms-pacman/siren1_firstloop.wav"}
	case LoopSiren2:
		return []string{"assets/ms-pacman/siren2_firstloop.wav"}
	case LoopSiren3:
		return []string{"assets/ms-pacman/siren3_firstloop.wav"}
	case LoopEyes:
		return []string{"assets/ms-pacman/eyes_firstloop.wav"}
	}
	return nil
}

// AllSounds lists every SoundID loaded by NewRaylibPlayer.
var AllSounds = []SoundID{
	SoundStart,
	SoundCredit,
	SoundChomp,
	SoundEatGhost,
	SoundEatFruit,
	SoundExtraLife,
	SoundDeath,
	SoundIntermission1,
	SoundIntermission2,
	SoundIntermission3,
}

// AllLoops lists every continuous LoopID loaded by NewRaylibPlayer.
var AllLoops = []LoopID{
	LoopSiren0,
	LoopSiren1,
	LoopSiren2,
	LoopSiren3,
	LoopSiren4,
	LoopFright,
	LoopEyes,
}

func fileExists(assetsFS fs.FS, path string) bool {
	if assetsFS == nil || path == "" {
		return false
	}
	info, err := fs.Stat(assetsFS, path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func firstExistingPath(assetsFS fs.FS, paths []string) (string, bool) {
	for _, path := range paths {
		if fileExists(assetsFS, path) {
			return path, true
		}
	}
	return "", false
}

// MissingAssets verifies every sound and loop candidate in assetsFS via fs.Stat
// before attempting to load audio. It returns the deduplicated list of primary
// expected paths for which no candidate file exists.
func MissingAssets(assetsFS fs.FS, pacmanMode bool) []string {
	var missing []string
	seen := make(map[string]bool)
	addMissing := func(paths []string) {
		if len(paths) == 0 {
			return
		}
		if _, ok := firstExistingPath(assetsFS, paths); ok {
			return
		}
		primary := paths[0]
		if !seen[primary] {
			seen[primary] = true
			missing = append(missing, primary)
		}
	}

	for _, id := range AllSounds {
		addMissing(SoundPaths(pacmanMode, id))
	}
	for _, loop := range AllLoops {
		addMissing(LoopPaths(loop))
	}
	return missing
}

// HasAnyAssets reports whether at least one usable WAV asset exists in assetsFS.
func HasAnyAssets(assetsFS fs.FS, pacmanMode bool) bool {
	for _, id := range AllSounds {
		if _, ok := firstExistingPath(assetsFS, SoundPaths(pacmanMode, id)); ok {
			return true
		}
	}
	for _, loop := range AllLoops {
		if _, ok := firstExistingPath(assetsFS, LoopPaths(loop)); ok {
			return true
		}
	}
	return false
}

// IntermissionSound maps a 1-based intermission act number to its SoundID.
func IntermissionSound(act int) SoundID {
	switch act {
	case 2:
		return SoundIntermission2
	case 3:
		return SoundIntermission3
	default:
		return SoundIntermission1
	}
}

// SirenLoopForDots selects one of the five escalating arcade siren pitches
// based on the fraction of dots remaining on the current board.
func SirenLoopForDots(dotsRemaining, totalDots int) LoopID {
	if totalDots <= 0 {
		return LoopSiren0
	}
	pct := (dotsRemaining * 100) / totalDots
	switch {
	case pct > 75:
		return LoopSiren0
	case pct > 50:
		return LoopSiren1
	case pct > 25:
		return LoopSiren2
	case pct > 10:
		return LoopSiren3
	default:
		return LoopSiren4
	}
}

var ghostLoopQuery = donburi.NewQuery(
	filter.Contains(component.GhostTag, component.Ghost),
)

// SelectBackgroundLoop chooses the highest-priority background audio loop for
// the current ECS world state: retreating eyes > frightened hum > dynamic siren.
func SelectBackgroundLoop(w donburi.World) LoopID {
	gs := component.MustFindGameState(w)
	if gs.Paused || gs.GameOver || gs.PlayerDying || gs.LevelComplete {
		return LoopNone
	}

	for entry := range ghostLoopQuery.Iter(w) {
		state := component.Ghost.Get(entry).State
		if state == config.Eaten || state == config.EnteringHouse {
			return LoopEyes
		}
	}

	if gs.IsFrightMode {
		return LoopFright
	}

	return SirenLoopForDots(gs.DotsRemaining, gs.DotsEaten+gs.DotsRemaining)
}

// HasOtherRetreatingEyes reports whether any ghost other than GhostEatenEntity
// is currently returning to the ghost house as eyes.
func HasOtherRetreatingEyes(w donburi.World) bool {
	gs := component.MustFindGameState(w)
	for entry := range ghostLoopQuery.Iter(w) {
		if entry.Entity() == gs.GhostEatenEntity {
			continue
		}
		state := component.Ghost.Get(entry).State
		if state == config.Eaten || state == config.EnteringHouse {
			return true
		}
	}
	return false
}
