package audio

import (
	"io/fs"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// RaylibPlayer loads embedded WAV clips into Raylib and drives one-shot SFX
// (with alternating waka-waka chomp pitch) and continuous background loops.
// Missing audio files or unavailable audio devices degrade silently to no-ops.
type RaylibPlayer struct {
	ready        bool
	paused       bool
	sounds       map[SoundID]rl.Sound
	loops        map[LoopID]rl.Sound
	pausedSounds []rl.Sound
	activeLoop   LoopID
	chompAlt     bool
}

// NewRaylibPlayer verifies that at least one audio file exists in assetsFS,
// initializes Raylib's audio device, and preloads available WAV assets for
// Pac-Man or Ms. Pac-Man mode.
func NewRaylibPlayer(assetsFS fs.FS, pacmanMode bool) *RaylibPlayer {
	p := &RaylibPlayer{
		sounds: make(map[SoundID]rl.Sound),
		loops:  make(map[LoopID]rl.Sound),
	}

	if !HasAnyAssets(assetsFS, pacmanMode) {
		return p
	}

	rl.InitAudioDevice()
	if !rl.IsAudioDeviceReady() {
		return p
	}
	p.ready = true

	for _, id := range AllSounds {
		if snd, ok := loadFirstWAV(assetsFS, SoundPaths(pacmanMode, id)); ok {
			p.sounds[id] = snd
		}
	}

	for _, loop := range AllLoops {
		if snd, ok := loadFirstWAV(assetsFS, LoopPaths(loop)); ok {
			p.loops[loop] = snd
		}
	}

	return p
}

func loadFirstWAV(assetsFS fs.FS, paths []string) (rl.Sound, bool) {
	path, ok := firstExistingPath(assetsFS, paths)
	if !ok {
		return rl.Sound{}, false
	}
	data, err := fs.ReadFile(assetsFS, path)
	if err != nil || len(data) == 0 {
		return rl.Sound{}, false
	}
	wave := rl.LoadWaveFromMemory(".wav", data, int32(len(data)))
	if wave.FrameCount == 0 {
		return rl.Sound{}, false
	}
	snd := rl.LoadSoundFromWave(wave)
	rl.UnloadWave(wave)
	return snd, true
}

// Play triggers a one-shot sound effect. Dot chomps alternate pitch slightly
// to reproduce the arcade waka-waka cadence.
func (p *RaylibPlayer) Play(id SoundID) {
	if !p.ready || p.paused {
		return
	}
	snd, ok := p.sounds[id]
	if !ok {
		return
	}
	if id == SoundChomp {
		p.chompAlt = !p.chompAlt
		if p.chompAlt {
			rl.SetSoundPitch(snd, 0.94)
		} else {
			rl.SetSoundPitch(snd, 1.06)
		}
		if rl.IsSoundPlaying(snd) {
			rl.StopSound(snd)
		}
	}
	rl.PlaySound(snd)
}

// SetLoop ensures loop is playing continuously and stops any previous loop.
// Call every frame during gameplay to seamlessly re-trigger completed loops.
func (p *RaylibPlayer) SetLoop(loop LoopID) {
	if !p.ready || p.paused {
		return
	}
	if loop != p.activeLoop {
		if prev, ok := p.loops[p.activeLoop]; ok && rl.IsSoundPlaying(prev) {
			rl.StopSound(prev)
		}
		p.activeLoop = loop
	}
	if loop == LoopNone {
		return
	}
	if snd, ok := p.loops[loop]; ok && !rl.IsSoundPlaying(snd) {
		rl.PlaySound(snd)
	}
}

// SetPaused pauses all currently playing sounds/loops when paused becomes true
// and resumes them when paused becomes false.
func (p *RaylibPlayer) SetPaused(paused bool) {
	if !p.ready || paused == p.paused {
		return
	}
	p.paused = paused
	if paused {
		p.pausedSounds = p.pausedSounds[:0]
		for _, snd := range p.sounds {
			if rl.IsSoundPlaying(snd) {
				rl.PauseSound(snd)
				p.pausedSounds = append(p.pausedSounds, snd)
			}
		}
		for _, snd := range p.loops {
			if rl.IsSoundPlaying(snd) {
				rl.PauseSound(snd)
				p.pausedSounds = append(p.pausedSounds, snd)
			}
		}
		return
	}
	for _, snd := range p.pausedSounds {
		rl.ResumeSound(snd)
	}
	p.pausedSounds = p.pausedSounds[:0]
}

// StopAll halts every active sound effect and background loop.
func (p *RaylibPlayer) StopAll() {
	if !p.ready {
		return
	}
	p.paused = false
	p.pausedSounds = p.pausedSounds[:0]
	for _, snd := range p.sounds {
		if rl.IsSoundPlaying(snd) {
			rl.StopSound(snd)
		}
	}
	for _, snd := range p.loops {
		if rl.IsSoundPlaying(snd) {
			rl.StopSound(snd)
		}
	}
	p.activeLoop = LoopNone
}

// Unload frees all loaded Raylib sounds and closes the audio device.
func (p *RaylibPlayer) Unload() {
	if !p.ready {
		return
	}
	p.StopAll()
	for _, snd := range p.sounds {
		rl.UnloadSound(snd)
	}
	for _, snd := range p.loops {
		rl.UnloadSound(snd)
	}
	clear(p.sounds)
	clear(p.loops)
	rl.CloseAudioDevice()
	p.ready = false
}
