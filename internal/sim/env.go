package sim

import (
	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/system"
)

// StepMode controls how many simulation ticks a single Env.Step advances.
type StepMode uint8

const (
	// StepFrames advances a fixed number of non-freeze ticks (default 1).
	StepFrames StepMode = iota
	// StepTile advances until Pac-Man is centered on a new tile or blocked by a wall.
	StepTile
	// StepIntersection advances until Pac-Man reaches a centered decision tile
	// (a tile with a perpendicular exit or a wall straight ahead) or is blocked.
	StepIntersection
)

const (
	maxTileStepTicks         = 64
	maxIntersectionStepTicks = 320
)

// ParseStepMode parses a wire step mode name ("frames", "tile", "intersection").
func ParseStepMode(s string) (StepMode, bool) {
	switch s {
	case "", "frames":
		return StepFrames, true
	case "tile":
		return StepTile, true
	case "intersection":
		return StepIntersection, true
	default:
		return StepFrames, false
	}
}

// Config configures a play-only headless simulation Env.
type Config struct {
	Mazes           *component.MazeSet
	PacmanMode      bool
	StartLevel      int
	Seed            int64
	EpisodicLife    bool
	EndOnLevelClear bool
	MaxTicks        int
}

// ResetOptions optionally overrides fields of Env's Config on Reset.
// Nil pointer fields keep the Env's existing configuration.
type ResetOptions struct {
	PacmanMode      *bool
	StartLevel      *int
	Seed            *int64
	EpisodicLife    *bool
	EndOnLevelClear *bool
	MaxTicks        *int
	ForceFullReset  bool
}

// StepResult summarizes the outcome of a single Env.Step call.
type StepResult struct {
	Tick         int
	Ticks        int
	Reward       int
	Terminated   bool
	Truncated    bool
	Events       EventMask
	Eaten        []TileCoord
	LevelChanged bool
	MazeIndex    int
}

// Env owns a Donburi ECS World and drives play-only arcade simulation
// deterministically without scenes, audio, or Raylib.
type Env struct {
	cfg       Config
	world     donburi.World
	input     *ScriptedInput
	pipeline  *system.Pipeline
	collector eventCollector

	player *donburi.Entry
	ghosts [4]*donburi.Entry // Indexed by component.ActorID (Blinky, Pinky, Inky, Clyde)

	tickCount       int
	done            bool
	continueOnReset bool
	onTick          func(action Action, tick int)
}

// NewEnv constructs an Env and resets it to the initial playable state.
func NewEnv(cfg Config) *Env {
	if cfg.StartLevel <= 0 {
		cfg.StartLevel = 1
	}
	cfg.StartLevel = max(1, min(cfg.StartLevel, 255))

	w := donburi.NewWorld()
	archetype.CreateGameState(w, cfg.PacmanMode, 1)
	mazeEntry := archetype.CreateMaze(w, cfg.PacmanMode)
	if cfg.Mazes != nil {
		component.Maze.Get(mazeEntry).UseTemplates(cfg.Mazes)
	}
	archetype.CreatePRNG(w)

	player := archetype.SpawnPlayer(w)
	spawnedGhosts := archetype.SpawnAllGhosts(w)

	env := &Env{
		cfg:    cfg,
		world:  w,
		input:  NewScriptedInput(),
		player: player,
	}
	for _, entry := range spawnedGhosts {
		id := int(component.Actor.Get(entry).ID)
		if id >= 0 && id < len(env.ghosts) {
			env.ghosts[id] = entry
		}
	}

	system.RegisterEventHandlers(w)
	env.collector.register(w)
	env.pipeline = system.NewPipeline(env.input)

	env.Reset(nil)
	return env
}

// Config returns the current active configuration of the environment.
func (e *Env) Config() Config {
	return e.cfg
}

// World returns the underlying Donburi ECS world.
func (e *Env) World() donburi.World {
	return e.world
}

// Tick returns the total number of simulation ticks executed since the last full game reset.
func (e *Env) Tick() int {
	return e.tickCount
}

// PlayerEntry returns the player entity entry.
func (e *Env) PlayerEntry() *donburi.Entry {
	return e.player
}

// GhostEntries returns the four ghost entries in canonical ActorID order
// (Blinky, Pinky, Inky, Clyde).
func (e *Env) GhostEntries() [4]*donburi.Entry {
	return e.ghosts
}

// SetTickHook installs an optional callback invoked after each individual sim tick
// (used by replay recording).
func (e *Env) SetTickHook(fn func(action Action, tick int)) {
	e.onTick = fn
}

// Reset starts a new episode. When EpisodicLife is enabled and the previous
// episode ended on a non-final death (lives remain), Reset continues the same
// game unless ForceFullReset is set or PacmanMode/StartLevel/EpisodicLife changed.
func (e *Env) Reset(opts *ResetOptions) {
	forceFull := false
	if opts != nil {
		if opts.ForceFullReset {
			forceFull = true
		}
		if opts.PacmanMode != nil && *opts.PacmanMode != e.cfg.PacmanMode {
			e.cfg.PacmanMode = *opts.PacmanMode
			forceFull = true
		}
		if opts.StartLevel != nil {
			lvl := max(1, min(*opts.StartLevel, 255))
			if lvl != e.cfg.StartLevel {
				e.cfg.StartLevel = lvl
				forceFull = true
			}
		}
		if opts.Seed != nil {
			e.cfg.Seed = *opts.Seed
		}
		if opts.EpisodicLife != nil {
			if !*opts.EpisodicLife && e.cfg.EpisodicLife {
				forceFull = true
			}
			e.cfg.EpisodicLife = *opts.EpisodicLife
		}
		if opts.EndOnLevelClear != nil {
			e.cfg.EndOnLevelClear = *opts.EndOnLevelClear
		}
		if opts.MaxTicks != nil {
			e.cfg.MaxTicks = *opts.MaxTicks
		}
	}

	gs := component.MustFindGameState(e.world)
	if e.continueOnReset && !forceFull && e.cfg.EpisodicLife && !gs.GameOver {
		e.done = false
		e.continueOnReset = false
		e.collector.reset()
		return
	}

	maze := component.MustFindMaze(e.world)
	gs.PacmanMode = e.cfg.PacmanMode
	maze.PacmanMode = e.cfg.PacmanMode
	gs.PlayerCount = 1
	gs.HighScore = 0

	prng := component.MustFindPRNG(e.world)
	prng.InitWithSeed(e.cfg.Seed)

	system.StartGameAtLevel(e.world, e.cfg.StartLevel, nil)
	e.tickCount = 0
	e.done = false
	e.continueOnReset = false
	e.collector.reset()
}

// LegalActions returns which cardinal directions [Right, Left, Up, Down] are
// currently passable from the player's position.
func (e *Env) LegalActions() [config.DirCount]bool {
	maze := component.MustFindMaze(e.world)
	pos := component.Transform.Get(e.player).Pos
	ct := pos.ToCenterTile()
	ox := pos.X % config.Tile
	oy := pos.Y % config.Tile

	var legal [config.DirCount]bool
	for d := range config.DirCount {
		dir := config.Direction(d)
		if system.CanPlayerMove(maze, ct.X, ct.Y, dir) {
			legal[d] = true
			continue
		}
		if (dir == config.Left || dir == config.Right) && ox != 0 {
			legal[d] = true
		} else if (dir == config.Up || dir == config.Down) && oy != 0 {
			legal[d] = true
		}
	}
	return legal
}

// Step advances the simulation according to mode and frames, applying action.
// All modes always run through any GhostEatenPause freeze and stop early on
// death, level clear, termination, or truncation.
func (e *Env) Step(action Action, mode StepMode, frames int) StepResult {
	e.collector.reset()
	gs := component.MustFindGameState(e.world)
	maze := component.MustFindMaze(e.world)

	if e.done {
		return StepResult{
			Tick:       e.tickCount,
			Terminated: gs.GameOver || (e.cfg.EpisodicLife && e.continueOnReset) || (e.cfg.EndOnLevelClear && gs.LevelComplete),
			Truncated:  e.cfg.MaxTicks > 0 && e.tickCount >= e.cfg.MaxTicks,
			MazeIndex:  maze.GetIndex(),
		}
	}

	startScore := gs.Score
	startLevel := gs.Level
	startTick := e.tickCount
	startPos := component.Transform.Get(e.player).Pos
	startTile := startPos.ToCenterTile()

	var targetNonFreeze int
	switch mode {
	case StepFrames:
		if frames <= 0 {
			frames = 1
		}
		targetNonFreeze = frames
	case StepTile:
		targetNonFreeze = maxTileStepTicks
	case StepIntersection:
		targetNonFreeze = maxIntersectionStepTicks
	default:
		targetNonFreeze = 1
	}

	terminated := false
	truncated := false
	nonFreezeTicks := 0

	for nonFreezeTicks < targetNonFreeze {
		prevPos := component.Transform.Get(e.player).Pos

		term, trunc := e.tick(action)
		nonFreezeTicks++

		// Decision 1: always run through the ghost-eaten freeze inside the same Step
		// so the agent is never asked for a decision while eaten ghost eyes travel home.
		for !term && !trunc && gs.GhostEatenPause > 0 {
			term, trunc = e.tick(action)
		}

		if term {
			terminated = true
			break
		}
		if trunc {
			truncated = true
			break
		}

		// Stop early on death or level clear even when the game continues.
		if e.collector.mask.Has(EventDeath) || e.collector.mask.Has(EventLevelClear) {
			break
		}

		if mode == StepTile || mode == StepIntersection {
			tr := component.Transform.Get(e.player)
			motion := component.Motion.Get(e.player)
			pos := tr.Pos

			// Stopped by a wall: player did not move and is not in a dot-eating pause
			// and cannot continue in its current direction.
			if pos == prevPos && motion.DotPause == 0 && !canPlayerContinue(maze, pos, tr.Dir) {
				break
			}

			if system.IsCentered(pos) {
				tile := pos.ToTile()
				if tile != startTile || pos != startPos {
					if mode == StepTile {
						break
					}
					if isDecisionTile(maze, tile.X, tile.Y, tr.Dir) {
						break
					}
				}
			}
		}
	}

	eatenCopy := make([]TileCoord, len(e.collector.eaten))
	copy(eatenCopy, e.collector.eaten)

	return StepResult{
		Tick:         e.tickCount,
		Ticks:        e.tickCount - startTick,
		Reward:       gs.Score - startScore,
		Terminated:   terminated,
		Truncated:    truncated,
		Events:       e.collector.mask,
		Eaten:        eatenCopy,
		LevelChanged: gs.Level != startLevel,
		MazeIndex:    maze.GetIndex(),
	}
}

// TickOne executes a single raw simulation tick with the given action.
// Exported for replay playback and viewer loops that drive the sim tick-by-tick.
func (e *Env) TickOne(action Action) (terminated, truncated bool) {
	if e.done {
		gs := component.MustFindGameState(e.world)
		return gs.GameOver || (e.cfg.EndOnLevelClear && gs.LevelComplete) || e.continueOnReset,
			e.cfg.MaxTicks > 0 && e.tickCount >= e.cfg.MaxTicks
	}
	return e.tick(action)
}

// tick executes exactly one Pipeline.Update and applies the three play-only
// transition rules from Decision 1.
func (e *Env) tick(action Action) (terminated, truncated bool) {
	e.input.Set(action)
	e.pipeline.Update(e.world)
	e.tickCount++

	gs := component.MustFindGameState(e.world)

	// Rule 1: PlayerDying -> ConsumeLifeOrSwitchPlayer (1-player only).
	if gs.PlayerDying {
		gs.PlayerCount = 1
		alive, _ := system.ConsumeLifeOrSwitchPlayer(e.world, nil)
		if !alive {
			e.done = true
			e.continueOnReset = false
			terminated = true
		} else if e.cfg.EpisodicLife {
			e.done = true
			e.continueOnReset = true
			terminated = true
		}
	}

	// Rule 2: LevelComplete -> AdvanceToNextLevel (or terminate if EndOnLevelClear).
	if !terminated && gs.LevelComplete {
		if e.cfg.EndOnLevelClear {
			e.done = true
			e.continueOnReset = false
			terminated = true
		} else {
			system.AdvanceToNextLevel(e.world, nil)
		}
	}

	// Rule 3 (GhostEatenPause > 0) is handled by Step's inner loop so TickOne
	// remains a faithful 1-tick primitive for replays and the windowed viewer.

	if !terminated && e.cfg.MaxTicks > 0 && e.tickCount >= e.cfg.MaxTicks {
		e.done = true
		e.continueOnReset = false
		truncated = true
	}

	if e.onTick != nil {
		e.onTick(action, e.tickCount)
	}

	return terminated, truncated
}

func canPlayerContinue(maze *component.MazeData, pos config.Vector, dir config.Direction) bool {
	tx := pos.X / config.Tile
	ty := pos.Y / config.Tile
	ox := pos.X % config.Tile
	oy := pos.Y % config.Tile

	if dir == config.Up || dir == config.Down {
		if oy != 0 {
			return true
		}
		targetTx := tx
		if ox > config.Half {
			targetTx++
		}
		return system.CanPlayerMove(maze, targetTx, ty, dir)
	}
	if ox != 0 {
		return true
	}
	targetTy := ty
	if oy > config.Half {
		targetTy++
	}
	return system.CanPlayerMove(maze, tx, targetTy, dir)
}

func isDecisionTile(maze *component.MazeData, tx, ty int, dir config.Direction) bool {
	if !system.CanPlayerMove(maze, tx, ty, dir) {
		return true
	}
	opp := dir.Opposite()
	for d := range config.DirCount {
		cand := config.Direction(d)
		if cand == dir || cand == opp {
			continue
		}
		if system.CanPlayerMove(maze, tx, ty, cand) {
			return true
		}
	}
	return false
}
