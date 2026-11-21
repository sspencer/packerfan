package audio

import (
	"bytes"
	"io"
	"io/fs"
	"math"

	ebitenAudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const defaultSampleRate = 48000

// EbitenPlayer loads embedded WAV clips into Ebitengine's audio mixer and
// drives one-shot SFX (with alternating waka-waka chomp pitch) and continuous
// background loops (with optional *_firstloop.wav transition intros). Missing
// audio files degrade silently to no-ops.
type EbitenPlayer struct {
	ready         bool
	paused        bool
	muted         bool
	sounds        map[SoundID]*ebitenAudio.Player
	chompLow      *ebitenAudio.Player
	chompHigh     *ebitenAudio.Player
	loops         map[LoopID]*ebitenAudio.Player
	pausedPlayers []*ebitenAudio.Player
	activeLoop    LoopID
	chompAlt      bool
}

// NewEbitenPlayer verifies that at least one audio file exists in assetsFS,
// initializes (or reuses) Ebitengine's audio context, and preloads available
// WAV assets for Pac-Man or Ms. Pac-Man mode.
func NewEbitenPlayer(assetsFS fs.FS, pacmanMode bool) *EbitenPlayer {
	p := &EbitenPlayer{
		sounds: make(map[SoundID]*ebitenAudio.Player),
		loops:  make(map[LoopID]*ebitenAudio.Player),
	}

	if !HasAnyAssets(assetsFS, pacmanMode) {
		return p
	}

	ctx := ebitenAudio.CurrentContext()
	if ctx == nil {
		ctx = ebitenAudio.NewContext(defaultSampleRate)
	}
	sr := ctx.SampleRate()
	p.ready = true

	for _, id := range AllSounds {
		data, ok := readFirstWAV(assetsFS, SoundPaths(pacmanMode, id))
		if !ok {
			continue
		}
		if id == SoundChomp {
			if pcm, ok := decodeWAVPCM(data, int(math.Round(float64(sr)/0.94))); ok {
				p.chompLow = ctx.NewPlayerFromBytes(pcm)
			}
			if pcm, ok := decodeWAVPCM(data, int(math.Round(float64(sr)/1.06))); ok {
				p.chompHigh = ctx.NewPlayerFromBytes(pcm)
			}
			continue
		}
		if pcm, ok := decodeWAVPCM(data, sr); ok {
			p.sounds[id] = ctx.NewPlayerFromBytes(pcm)
		}
	}

	for _, loop := range AllLoops {
		loopData, ok := readFirstWAV(assetsFS, LoopPaths(loop))
		if !ok {
			continue
		}
		loopPCM, ok := decodeWAVPCM(loopData, sr)
		if !ok {
			continue
		}

		var stream io.Reader
		if firstData, ok := readFirstWAV(assetsFS, LoopFirstPaths(loop)); ok {
			if firstPCM, ok := decodeWAVPCM(firstData, sr); ok && len(firstPCM) > 0 {
				combined := make([]byte, 0, len(firstPCM)+len(loopPCM))
				combined = append(combined, firstPCM...)
				combined = append(combined, loopPCM...)
				stream = ebitenAudio.NewInfiniteLoopWithIntro(
					bytes.NewReader(combined),
					int64(len(firstPCM)),
					int64(len(loopPCM)),
				)
			}
		}
		if stream == nil {
			stream = ebitenAudio.NewInfiniteLoop(bytes.NewReader(loopPCM), int64(len(loopPCM)))
		}
		if pl, err := ctx.NewPlayer(stream); err == nil {
			p.loops[loop] = pl
		}
	}

	return p
}

func readFirstWAV(assetsFS fs.FS, paths []string) ([]byte, bool) {
	path, ok := firstExistingPath(assetsFS, paths)
	if !ok {
		return nil, false
	}
	data, err := fs.ReadFile(assetsFS, path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func decodeWAVPCM(data []byte, targetSampleRate int) ([]byte, bool) {
	stream, err := wav.DecodeWithSampleRate(targetSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	pcm, err := io.ReadAll(stream)
	if err != nil || len(pcm) == 0 {
		return nil, false
	}
	return pcm, true
}

// Play triggers a one-shot sound effect. Dot chomps alternate pitch slightly
// to reproduce the arcade waka-waka cadence.
func (p *EbitenPlayer) Play(id SoundID) {
	if !p.ready || p.paused || p.muted {
		return
	}
	if id == SoundChomp {
		if p.chompLow != nil && p.chompLow.IsPlaying() {
			p.chompLow.Pause()
		}
		if p.chompHigh != nil && p.chompHigh.IsPlaying() {
			p.chompHigh.Pause()
		}
		p.chompAlt = !p.chompAlt
		pl := p.chompHigh
		if p.chompAlt {
			pl = p.chompLow
		}
		if pl != nil {
			_ = pl.Rewind()
			pl.Play()
		}
		return
	}
	pl, ok := p.sounds[id]
	if !ok {
		return
	}
	_ = pl.Rewind()
	pl.Play()
}

// SetLoop ensures loop is playing continuously and stops any previous loop.
// When an optional *_firstloop.wav clip exists for loop, InfiniteLoopWithIntro
// plays it once on transition before seamlessly repeating the main loop clip.
func (p *EbitenPlayer) SetLoop(loop LoopID) {
	if !p.ready || p.paused || p.muted {
		return
	}
	if loop != p.activeLoop {
		if prev, ok := p.loops[p.activeLoop]; ok {
			prev.Pause()
			_ = prev.Rewind()
		}
		p.activeLoop = loop
		if loop == LoopNone {
			return
		}
		if next, ok := p.loops[loop]; ok {
			_ = next.Rewind()
			next.Play()
		}
		return
	}
	if loop == LoopNone {
		return
	}
	if pl, ok := p.loops[loop]; ok && !pl.IsPlaying() {
		_ = pl.Rewind()
		pl.Play()
	}
}

// SetPaused pauses all currently playing sounds/loops when paused becomes true
// and resumes them when paused becomes false.
func (p *EbitenPlayer) SetPaused(paused bool) {
	if !p.ready || paused == p.paused {
		return
	}
	p.paused = paused
	if paused {
		p.pausedPlayers = p.pausedPlayers[:0]
		p.forEachPlayer(func(pl *ebitenAudio.Player) {
			if pl.IsPlaying() {
				pl.Pause()
				p.pausedPlayers = append(p.pausedPlayers, pl)
			}
		})
		return
	}
	for _, pl := range p.pausedPlayers {
		pl.Play()
	}
	p.pausedPlayers = p.pausedPlayers[:0]
}

// SetMuted silences or restores audio playback.
func (p *EbitenPlayer) SetMuted(muted bool) {
	if p.muted == muted {
		return
	}
	p.muted = muted
	if muted {
		p.StopAll()
	}
}

// Muted reports whether audio is currently muted.
func (p *EbitenPlayer) Muted() bool {
	return p.muted
}

// StopAll halts every active sound effect and background loop.
func (p *EbitenPlayer) StopAll() {
	if !p.ready {
		return
	}
	p.paused = false
	p.pausedPlayers = p.pausedPlayers[:0]
	p.forEachPlayer(func(pl *ebitenAudio.Player) {
		if pl.IsPlaying() {
			pl.Pause()
		}
		_ = pl.Rewind()
	})
	p.activeLoop = LoopNone
}

// Unload closes all Ebitengine audio players owned by this instance.
func (p *EbitenPlayer) Unload() {
	if !p.ready {
		return
	}
	p.StopAll()
	p.forEachPlayer(func(pl *ebitenAudio.Player) {
		_ = pl.Close()
	})
	clear(p.sounds)
	clear(p.loops)
	p.chompLow = nil
	p.chompHigh = nil
	p.ready = false
}

func (p *EbitenPlayer) forEachPlayer(fn func(*ebitenAudio.Player)) {
	for _, pl := range p.sounds {
		if pl != nil {
			fn(pl)
		}
	}
	if p.chompLow != nil {
		fn(p.chompLow)
	}
	if p.chompHigh != nil {
		fn(p.chompHigh)
	}
	for _, pl := range p.loops {
		if pl != nil {
			fn(pl)
		}
	}
}
