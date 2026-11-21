package game

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"pacfan/assets"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/sheet"
	"pacfan/internal/sim"
)

const (
	MinReplaySpeed = 0.25
	MaxReplaySpeed = 16.0
)

// ReplayArt holds the resolved spritesheet bytes, parsed Sheet, and MazeSet for a replay.
type ReplayArt struct {
	File    string
	Bytes   []byte
	Sheet   *sheet.Sheet
	Mazes   *component.MazeSet
	Warning string
}

// DiscoverReplayFiles resolves a single .pfr file path or recursively collects
// sorted .pfr files inside a directory.
func DiscoverReplayFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("replay: stat %q: %w", path, err)
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	var files []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(d.Name()), ".pfr") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("replay: scan %q: %w", path, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("replay: no .pfr files found in %q", path)
	}
	return files, nil
}

// ResolveReplayArt selects the PNG matching artOverride (if set) or the
// embedded asset whose base filename matches rep.Header.Art.File (else default
// embedded asset), parses its MazeSet, and rejects a maze_hash mismatch.
func ResolveReplayArt(assetsFS fs.FS, artOverride string, rep *sim.Replay) (*ReplayArt, error) {
	var (
		name string
		data []byte
		err  error
	)

	if artOverride != "" {
		data, err = os.ReadFile(artOverride)
		if err != nil {
			return nil, fmt.Errorf("replay: read --art %q: %w", artOverride, err)
		}
		name = filepath.Base(artOverride)
	} else {
		chosen := ""
		if rep.Header.Art.File != "" {
			candidate := "assets/" + filepath.Base(rep.Header.Art.File)
			if assets.IsAssetFile(assetsFS, candidate) {
				chosen = candidate
			}
		}
		if chosen == "" {
			chosen = assets.GameAssetPath(assetsFS)
		}
		data, err = fs.ReadFile(assetsFS, chosen)
		if err != nil {
			return nil, fmt.Errorf("replay: read embedded asset %q: %w", chosen, err)
		}
		name = filepath.Base(chosen)
	}

	s, err := sheet.Decode(name, data)
	if err != nil {
		return nil, err
	}
	mazes := component.ParseMazeSet(s.IsPixelLit)

	if rep.Header.MazeHash != "" && mazes.Hash() != rep.Header.MazeHash {
		needFile := rep.Header.Art.File
		if needFile == "" {
			needFile = "matching artwork PNG"
		}
		return nil, fmt.Errorf(
			"replay maze_hash mismatch: replay requires %q (maze_hash=%s), but loaded %q (maze_hash=%s); pass --art FILE with matching mazes",
			needFile, rep.Header.MazeHash, name, mazes.Hash(),
		)
	}

	var warning string
	if rep.Header.Art.SHA256 != "" && s.SHA256() != rep.Header.Art.SHA256 {
		warning = fmt.Sprintf("artwork SHA-256 differs from replay (%s), but maze_hash matches", rep.Header.Art.File)
	}

	return &ReplayArt{
		File:    name,
		Bytes:   data,
		Sheet:   s,
		Mazes:   mazes,
		Warning: warning,
	}, nil
}

// ReplayMeta captures optional well-known fields from ReplayHeader.Meta for HUD display.
type ReplayMeta struct {
	RunID      string `json:"run_id"`
	Checkpoint string `json:"checkpoint"`
	Episode    *int   `json:"episode"`
}

// ReplayPlayer drives one or more .pfr replays through sim.Env without Ebitengine dependencies.
type ReplayPlayer struct {
	assetsFS    fs.FS
	artOverride string
	zoom        float32

	files     []string
	fileIndex int

	replay        *sim.Replay
	art           *ReplayArt
	env           *sim.Env
	meta          ReplayMeta
	checkpoints   map[int]uint64
	runIndex      int
	runOffset     int
	speed         float64
	paused        bool
	finished      bool
	checkpointsOK int
	hashErr       error
	clockAcc      float64
}

// NewReplayPlayer validates and loads the first replay from path (file or directory).
func NewReplayPlayer(assetsFS fs.FS, path, artOverride string, zoom float32, speed float64) (*ReplayPlayer, error) {
	files, err := DiscoverReplayFiles(path)
	if err != nil {
		return nil, err
	}
	p := &ReplayPlayer{
		assetsFS:    assetsFS,
		artOverride: artOverride,
		zoom:        zoom,
		files:       files,
		speed:       clampReplaySpeed(speed),
	}
	if err := p.LoadIndex(0); err != nil {
		return nil, err
	}
	return p, nil
}

func clampReplaySpeed(s float64) float64 {
	if s < MinReplaySpeed {
		return MinReplaySpeed
	}
	if s > MaxReplaySpeed {
		return MaxReplaySpeed
	}
	return s
}

// LoadIndex loads playlist item idx, validates its artwork/maze_hash, and resets sim.Env.
func (p *ReplayPlayer) LoadIndex(idx int) error {
	if len(p.files) == 0 {
		return fmt.Errorf("replay: empty playlist")
	}
	idx = ((idx % len(p.files)) + len(p.files)) % len(p.files)
	rep, err := sim.ReadReplayFile(p.files[idx])
	if err != nil {
		return err
	}
	art, err := ResolveReplayArt(p.assetsFS, p.artOverride, rep)
	if err != nil {
		return err
	}

	env := sim.NewEnv(sim.Config{
		Mazes:           art.Mazes,
		PacmanMode:      rep.Header.Pacman,
		StartLevel:      rep.Header.Level,
		Seed:            rep.Header.Seed,
		EpisodicLife:    rep.Header.Options.EpisodicLife,
		EndOnLevelClear: rep.Header.Options.EndOnLevelClear,
		MaxTicks:        rep.Header.Options.MaxTicks,
	})
	// Zoom is cosmetic and excluded from HashWorld.
	component.MustFindGameState(env.World()).Zoom = p.zoom

	cpMap := make(map[int]uint64, len(rep.Checkpoints))
	for _, cp := range rep.Checkpoints {
		cpMap[cp.Tick] = cp.Hash
	}

	var meta ReplayMeta
	if len(rep.Header.Meta) > 0 {
		_ = json.Unmarshal(rep.Header.Meta, &meta)
	}

	p.fileIndex = idx
	p.replay = rep
	p.art = art
	p.env = env
	p.meta = meta
	p.checkpoints = cpMap
	p.runIndex = 0
	p.runOffset = 0
	p.finished = rep.TotalTicks == 0
	p.checkpointsOK = 0
	p.hashErr = nil
	p.clockAcc = 0

	if want, ok := p.checkpoints[0]; ok {
		got := sim.HashWorld(p.env.World())
		if got != want {
			p.hashErr = fmt.Errorf("hash mismatch at tick 0: got %016x want %016x", got, want)
		} else {
			p.checkpointsOK++
		}
	}
	return nil
}

// Advance ticks playback according to elapsed wall seconds dt and current speed.
func (p *ReplayPlayer) Advance(dt float64) {
	if p.paused || p.finished || p.hashErr != nil {
		return
	}
	if dt < 0 {
		dt = 0
	}
	if dt > 0.25 {
		dt = 0.25
	}
	p.clockAcc += dt * config.ArcadeFPS * p.speed
	const maxTicksPerFrame = 64
	ticks := 0
	for p.clockAcc >= 1.0 && ticks < maxTicksPerFrame && !p.finished && p.hashErr == nil {
		p.clockAcc -= 1.0
		p.StepTick()
		ticks++
	}
}

// StepTick advances playback by exactly one recorded tick and verifies any checkpoint at that tick.
func (p *ReplayPlayer) StepTick() bool {
	if p.finished || p.hashErr != nil {
		return false
	}
	for p.runIndex < len(p.replay.Runs) && p.runOffset >= p.replay.Runs[p.runIndex].Count {
		p.runIndex++
		p.runOffset = 0
	}
	if p.runIndex >= len(p.replay.Runs) {
		p.finished = true
		return false
	}

	action := p.replay.Runs[p.runIndex].Action
	p.runOffset++
	p.env.TickOne(action)

	tick := p.env.Tick()
	if want, ok := p.checkpoints[tick]; ok {
		got := sim.HashWorld(p.env.World())
		if got != want {
			p.hashErr = fmt.Errorf("hash mismatch at tick %d: got %016x want %016x", tick, got, want)
		} else {
			p.checkpointsOK++
		}
	}

	if tick >= p.replay.TotalTicks {
		p.finished = true
	}
	return true
}

func (p *ReplayPlayer) TogglePause()     { p.paused = !p.paused }
func (p *ReplayPlayer) Paused() bool     { return p.paused }
func (p *ReplayPlayer) Finished() bool   { return p.finished }
func (p *ReplayPlayer) Speed() float64   { return p.speed }
func (p *ReplayPlayer) HashError() error { return p.hashErr }
func (p *ReplayPlayer) CheckpointsOK() int {
	return p.checkpointsOK
}
func (p *ReplayPlayer) Env() *sim.Env       { return p.env }
func (p *ReplayPlayer) Art() *ReplayArt     { return p.art }
func (p *ReplayPlayer) Replay() *sim.Replay { return p.replay }
func (p *ReplayPlayer) FilePath() string    { return p.files[p.fileIndex] }
func (p *ReplayPlayer) FileCount() int      { return len(p.files) }
func (p *ReplayPlayer) FileIndex() int      { return p.fileIndex }

func (p *ReplayPlayer) Faster() {
	p.speed = clampReplaySpeed(p.speed * 2.0)
}

func (p *ReplayPlayer) Slower() {
	p.speed = clampReplaySpeed(p.speed / 2.0)
}

func (p *ReplayPlayer) NextFile() error {
	return p.LoadIndex(p.fileIndex + 1)
}

func (p *ReplayPlayer) PrevFile() error {
	return p.LoadIndex(p.fileIndex - 1)
}

// HUDLine returns a concise status line describing run metadata, tick, speed, and playlist index.
func (p *ReplayPlayer) HUDLine() string {
	var parts []string
	if p.meta.RunID != "" {
		parts = append(parts, p.meta.RunID)
	}
	if p.meta.Checkpoint != "" {
		parts = append(parts, p.meta.Checkpoint)
	}
	if p.meta.Episode != nil {
		parts = append(parts, fmt.Sprintf("ep=%d", *p.meta.Episode))
	}
	if len(parts) == 0 {
		parts = append(parts, filepath.Base(p.FilePath()))
	}
	if len(p.files) > 1 {
		parts = append(parts, fmt.Sprintf("[%d/%d]", p.fileIndex+1, len(p.files)))
	}
	parts = append(parts, fmt.Sprintf("t=%d/%d", p.env.Tick(), p.replay.TotalTicks))
	parts = append(parts, fmt.Sprintf("%.2gx", p.speed))
	switch {
	case p.hashErr != nil:
		parts = append(parts, "HASH ERR")
	case p.finished:
		parts = append(parts, "DONE")
	case p.paused:
		parts = append(parts, "PAUSED")
	}
	return strings.Join(parts, " ")
}
