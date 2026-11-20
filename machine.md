# Headless Play for Machine Learning (`machine.md`)

This plan describes how I would let a machine-learning agent play `pacfan` without graphics, at many times the arcade's 60.6 ticks per second, while capturing every game so it can be replayed later in the normal Raylib window to watch the agent improve.

---

## Goals

1. **No graphics or audio needed to play.** A training process must run on a machine with no display, GPU, OpenGL, or sound device, and must not link Raylib at all.
2. **Run as fast as the CPU allows.** The simulation advances only when the agent asks for it, never on a 60.6 Hz timer, and many games run in parallel.
3. **Same rules as the real game.** The agent must learn the game people play, not a rewritten copy of it. Headless mode reuses the existing ECS systems and scenes unchanged.
4. **Every game can be replayed.** Any game the agent plays can be watched later, frame for frame, in the real renderer with sound.
5. **Progress is visible over time.** Replays are organized by training run and checkpoint so it is easy to compare an early agent with a late one.

---

## Key Idea: Determinism Makes Replays Nearly Free

The simulation is already deterministic:

- All state lives in one `donburi.World`, with no package-level globals.
- The only randomness is `component.PRNG`, whose simulated ROM is generated from a fixed seed (`rand.NewSource(42)`), and whose cursor is part of the world.
- Systems run in a fixed order in `system.Pipeline.Update`, and the scene manager makes transitions on fixed frame counts.

So a game is completely described by **its starting configuration plus the input given on each tick**. Instead of recording video or full state snapshots, a replay file stores only:

- a header (game mode, starting level, PRNG seed, timing profile, version), and
- the agent's joystick direction, run-length encoded (store a direction only when it changes, with the number of ticks it was held).

A full game compresses to a few kilobytes, so **every** training game can be kept. To watch one, the normal game feeds the recorded inputs back through `system.InputProvider` and the real renderer draws it exactly as it happened.

The rest of this plan is about protecting that determinism, making the headless loop fast, and building tools around the replay files.

---

## Architecture Overview

```text
                    ┌──────────────────────── shared, unchanged ─────────────────────────┐
                    │ internal/component  internal/system  internal/scene  internal/config│
                    └──────────────▲──────────────────────────────▲──────────────────────┘
                                   │                              │
       ┌───────────────────────────┴───────┐        ┌─────────────┴──────────────────────┐
       │ internal/sim  (NEW, no Raylib)    │        │ main.go + internal/render (Raylib) │
       │  Env: Reset / Step / Observe      │        │  normal play                       │
       │  ScriptedInput, NoopDrawer        │        │  -record  (save human games)       │
       │  replay.Writer / replay.Reader    │        │  -replay  (watch any saved game)   │
       └───────────────▲───────────────────┘        └─────────────▲──────────────────────┘
                       │                                          │
       ┌───────────────┴───────────────────┐                      │
       │ cmd/pacfan-train (NEW)            │   runs/<run>/…/*.pfr │
       │  Go agents, or c-shared library   │ ─────────────────────┘
       │  for Python (vectorized envs)     │
       └───────────────────────────────────┘
```

---

## Part 1: Make the Simulation Run Without Raylib

The ECS and scene packages already have no Raylib imports, and scenes already talk to the outside world through interfaces (`scene.Drawer`, `audio.Player`, `system.InputProvider`). Three things still tie a game session to Raylib.

### 1.1 Maze loading without a GPU texture

`MazeData.LoadLevel` builds the tile grid by sampling spritesheet pixels through an `isPixelLit` callback, and today that callback comes from `render.Art`, which loads the PNG through Raylib.

- Add a small pure-Go package (for example `internal/sheet`) that decodes `assets/game.png` (or `game-example.png`) with `image/png` from an `fs.FS` and returns `isPixelLit(x, y int) bool`. The tests already contain this logic; it just needs a shared home.
- Decode the sheet **once per process** and share the read-only pixel data across all environments.
- Better still, parse each of the five maze grids once and cache the result, so a level reset copies a 28×31 grid instead of sampling about 55,000 pixels.

### 1.2 A composition root that is not `game.Engine`

`game.NewEngine` builds the world but also creates the Raylib renderer, Raylib input, the audio player, and high-score persistence. Headless mode needs its own small constructor in `internal/sim` that wires the same pieces with headless stand-ins:

| Dependency | Normal game | Headless |
| --- | --- | --- |
| `scene.Drawer` | `render.Renderer` | `NoopDrawer` (or `nil`; scenes already check for it) |
| `audio.Player` | `audio.RaylibPlayer` | `audio.NoopPlayer` |
| `system.InputProvider` | `render.RaylibInput` | `sim.ScriptedInput` (the agent's action for the next tick) |
| `isPixelLit` | `render.Art.IsPixelLit` | `sheet` package (Part 1.1) |
| High score file | read and written under `os.UserConfigDir()` | not used: no file I/O per episode |

The training binary (`cmd/pacfan-train`) imports `internal/sim` but never `internal/render` or `internal/audio/raylib.go`, so it builds as a plain Go program that can run on headless Linux servers and in containers. A CI check (`go list -deps ./cmd/pacfan-train | grep raylib` must find nothing) keeps it that way.

### 1.3 Remove waiting from the loop

Raylib's `SetTargetFPS` and `config.ArcadeClock` only live in `main.go`. The headless loop simply calls `scene.Manager.Update()` as fast as the agent supplies actions; the 60.6 Hz pacing disappears without any change to game code.

---

## Part 2: Keep the Real Rules, Skip the Dead Time

I would drive the **real scene manager**, not just `Pipeline.Update`. Deaths, life loss, level advances, ghost-eaten freezes, and Game Over are handled by the scenes (`PacmanDyingScene`, `RoundClearedScene`, `GhostEatenFreezeScene`, …), and copying that logic into a separate headless loop would let the two drift apart.

The scenes do contain many frames where nothing can be controlled: the READY! countdown, death animation, maze flashing, and intermissions. `scene.Context` already has override fields for tests, and the headless constructor uses them as a **fast timing profile**:

- `ReadyDuration`, `PlayerOneDuration`, `RoundClearedHoldFrames`, `RoundClearedFlashes`, `PlayerDyingHoldFrames`, `PlayerDyingFrameTicks`, `PlayerDyingPostFrames` set to 1.
- `IntermissionFactory` returns `nil` to skip intermissions.
- The episode starts with `system.StartNewGame(w, 1, isPixelLit)` and the ready scene, bypassing the attract screen; the Game Over scene ends the episode.
- The 61-frame ghost-eaten freeze is kept (eaten eyes keep moving during it, so it affects play), but the agent is not asked for decisions during it.

The timing profile is written into every replay header, and replays are always played back with the profile they were recorded with (Part 4.3), so this shortcut never causes a desync.

---

## Part 3: The Agent Interface

### 3.1 Core Go API (`internal/sim`)

```go
type Config struct {
    PacmanMode  bool        // classic Pac-Man instead of Ms. Pac-Man
    StartLevel  int         // curriculum: start at a later level
    Seed        int64       // PRNG ROM seed (42 = arcade default)
    Timing      TimingProfile
    FrameSkip   int         // ticks per decision (action repeat), default 4
    MaxTicks    int         // truncate long episodes (0 = no limit)
    Record      bool        // keep the input log for a replay file
}

type Action uint8 // NoOp, Up, Left, Down, Right

type StepResult struct {
    Reward     float32
    Terminated bool         // Game Over (or life lost, if configured)
    Truncated  bool         // MaxTicks reached
    Events     EventFlags   // dot, power pellet, ghost eaten, fruit, death, level clear, extra life
}

func NewEnv(sheet *sheet.Sheet, cfg Config) *Env
func (e *Env) Reset(seed int64)
func (e *Env) Step(a Action) StepResult
func (e *Env) Observe(dst []float32)      // writes into a caller-owned buffer, no allocation
func (e *Env) LegalActions() [5]bool      // for action masking
func (e *Env) Replay() *replay.Replay     // the episode so far
```

Design choices:

- **The agent acts at decision points, not every tick.** One `Step` repeats the chosen direction for `FrameSkip` ticks (stopping early on death or level clear). A second mode, "step until the next intersection", would shorten episodes even further, because a direction can only change at an intersection or by reversing. Both modes record **ticks**, so replays do not depend on which one was used.
- **Events come from the existing typed event bus.** `internal/sim` subscribes to `DotEaten`, `GhostEaten`, `PlayerEaten`, `LevelCompleted`, and the others on each world, rather than diffing state.
- **Reward is configurable.** The default is the score change; options add a death penalty, a level-clear bonus, and a small per-step cost. The replay stores the raw game, not the reward, so a reward change never invalidates replays.
- **No allocations per step.** Observations are written into caller buffers and event flags are a bitmask, so the garbage collector stays quiet at millions of steps per minute.

### 3.2 Observations

The world holds exact, structured state, so agents learn from that rather than from pixels:

- **Grid tensor** (for convolutional networks): channels over the 28×31 tile grid: walls and door, dots, power pellets, Pac-Man, each ghost (one channel per ghost, so Blinky's and Clyde's different behavior can be learned), frightened ghosts, returning eyes, and fruit. Ghost heading is shown as an extra channel with the previous tile, so direction is visible from a single frame.
- **Scalar features** appended after the grid: fright time left, scatter or chase phase, dots remaining, level, lives, and Pac-Man's current direction.
- **Feature vector** (for small networks and evolutionary methods): maze-path distances (breadth-first search over the tile graph) from Pac-Man to the nearest dot, power pellet, fruit, and each ghost, in each of the four directions.

All encodings are computed in Go from the component data, so different agents can pick one without new plumbing.

### 3.3 Running many games at once

Because worlds share nothing, one process can run hundreds of `Env` values on a goroutine pool. A `VecEnv` steps N environments with one call, taking an array of N actions and filling contiguous observation, reward, and done arrays, then auto-resetting finished episodes.

### 3.4 Agents in Go and in Python

- **Go agents** (scripted baselines, tree search, evolutionary strategies, a small policy network) call `Env` directly. This is the fastest path and is ideal for search, because an Env can be reset and replayed from an input log to explore "what if" branches.
- **Python agents** (PyTorch, JAX, Stable-Baselines3, CleanRL) load the simulator as a **shared library** built with `go build -buildmode=c-shared`. A small C interface (`pf_vec_new`, `pf_vec_reset`, `pf_vec_step`, `pf_vec_save_replay`) lets the library write observations straight into NumPy arrays owned by Python, so a batched step costs one foreign call for all N games instead of one message per game. A thin Gymnasium `VectorEnv` wrapper sits on top. Avoiding a per-step network or pipe protocol is what keeps the speed in the hundreds of thousands of ticks per second.

### 3.5 Speed target

One tick is a handful of systems over 5 entities and a 868-tile grid. I would measure before promising numbers, with `go test -bench` on a random agent, but I expect a single core to simulate **well over 100,000 ticks per second** (more than 1,500 times real time), and throughput to scale with cores. A profile-guided pass on hot paths (per-tick allocations, query iteration, event dispatch) would come before any deeper optimization.

---

## Part 4: Capturing and Replaying Games

### 4.1 Replay file format (`.pfr`)

```text
magic "PFR1" | format version
header (JSON, length-prefixed):
  game version (git commit), mode (Ms. Pac-Man / Pac-Man), start level,
  PRNG seed, timing profile, art used (-ex or real), recorded at,
  run id, agent checkpoint, episode number
inputs: varint pairs (direction, ticks held) until the end of the game
checkpoints: every 600 ticks, a 64-bit hash of the world state
footer (JSON): final score, level reached, ticks, deaths, dots eaten, cause of end
```

- **Inputs are recorded at the `InputProvider` boundary.** A `RecordingInput` wrapper wraps any provider and logs what it returned on each `Sample()` call. Because it wraps any provider, the same code records agent games and **human games** (a `-record` flag on the normal game), which gives demonstration data for imitation learning and a human baseline to compare against.
- **State hashes catch desyncs.** The hash covers `GameState`, the maze grid, actor positions and states, and the PRNG cursor. If playback ever disagrees with a recorded hash, the player reports the first bad tick instead of silently showing a different game.
- **The header pins the game version.** A rule change makes old replays play out differently, so the player warns when the commit differs, and a small "replay corpus" test (below) catches changes that break replays by accident.

### 4.2 Organizing replays for a training run

```text
runs/<run-id>/
├── run.json                 # config, reward settings, start time, code version
├── episodes.jsonl           # one line per episode: number, checkpoint, score, level, ticks, replay path
├── checkpoints/             # agent weights
└── replays/
    ├── train/000000123.pfr  # every training episode (or a sample, if disk is tight)
    ├── eval/ckpt-0050/*.pfr # fixed-seed evaluation games for each checkpoint
    └── best/                # new high scores, copied as they happen
```

- **Evaluation games are the main way to see progress.** At every checkpoint, the trainer plays a fixed set of evaluation games (same seeds and start levels each time, no exploration noise). Watching `eval/ckpt-0010`, then `ckpt-0100`, then `ckpt-1000` on the same seed shows exactly what the agent learned.
- `episodes.jsonl` drives charts (score, level reached, and survival time by episode or checkpoint) without opening any replay.

### 4.3 Watching replays in the real game

New flags on the normal `pacfan` binary:

| Flag | Meaning |
| --- | --- |
| `-replay FILE` | Play one replay, or every `.pfr` in a directory in order |
| `-replay-speed N` | Playback speed multiplier (0.25 to 16) |
| `-record FILE` | Save the game a person plays |

How playback works:

1. Read the header and build the world with the recorded mode, start level, seed, and **timing profile**.
2. Replace `RaylibInput` with a `ReplayInput` that returns the recorded direction on each `Sample()` call. Keyboard input is still read for viewer controls only.
3. Run the normal renderer and audio. With the fast profile, the viewer adds **cosmetic holds** (for example, keeping the death frame or flashing maze on screen for a second) that pause the picture without advancing the simulation, so the game looks natural but the tick sequence is unchanged.
4. Check each recorded state hash as playback passes it.

Viewer controls: Space pauses, Right Arrow steps one tick while paused, `+`/`-` change speed, N and P jump to the next or previous replay in a directory. An on-screen line shows the run, checkpoint, episode, and tick, so a sequence of evaluation replays plays like a "learning over time" reel.

Optional later additions:

- **Video export**: render each frame to an off-screen `RenderTexture` and pipe it to `ffmpeg` to make MP4 or GIF clips for sharing (needs a GPU or software OpenGL, but not a visible window).
- **Side-by-side view**: two replays with the same seed (for example, an early and a late checkpoint) drawn next to each other.
- **Heat maps**: positions from many replays overlaid on the maze, showing where the agent goes and where it dies.

---

## Part 5: Protecting Determinism

Replays are only trustworthy if the same inputs always produce the same game. I would make this a tested guarantee:

1. **Audit for hidden nondeterminism.** Look for map iteration whose order affects behavior, any use of time or wall clock, floating-point arithmetic that could vary across platforms, and global `math/rand` use. Today the only randomness is the seeded PRNG ROM, which is a good starting point.
2. **Make the seed configurable.** Add `PRNGData.InitWithSeed(seed)` with 42 as the default, so training can vary ghost randomness across episodes while every episode stays reproducible.
3. **Round-trip test.** Play a few thousand ticks with a random agent while recording, replay the log in a fresh world, and require identical state hashes on **every** tick, in both maze modes and with both timing profiles.
4. **Replay corpus test.** Commit a few small replay files (one per mode, including deaths, eaten ghosts, fruit, and a level clear) with their recorded hashes. Any change to game rules that alters them fails this test, so changes to rules are always deliberate.
5. **Headless equals windowed.** Replay a headless-recorded game through the windowed code path (with a `NoopDrawer` in tests) and compare hashes, proving the two compositions run the same simulation.

---

## Part 6: Milestones

| Step | Work | Done when |
| --- | --- | --- |
| 1 | Pure-Go `sheet` loader, cached maze grids, seedable PRNG | Tests run the full game with no Raylib calls |
| 2 | `internal/sim` `Env` with fast timing profile, `ScriptedInput`, events, rewards | A random agent plays complete games headlessly; benchmark shows ticks per second |
| 3 | `RecordingInput`, `.pfr` writer and reader, state hashes, determinism tests | Round-trip and corpus tests pass |
| 4 | `-replay` and `-record` in the windowed game, viewer controls, cosmetic holds | A headless game plays back in the window with no hash mismatch |
| 5 | `VecEnv`, observation encoders, `cmd/pacfan-train` with a simple Go baseline agent (greedy dot-chaser that avoids ghosts) and run-directory layout | A training run produces checkpoints, evaluation replays, and `episodes.jsonl` |
| 6 | c-shared library and Python Gymnasium wrapper; a PPO example | A Python agent trains and its evaluation replays visibly improve |
| 7 (optional) | Video export, side-by-side and heat-map views, intersection stepping | Learning progress can be shared as clips |

Steps 1 to 4 are useful on their own, even without any ML: they give fast headless tests, bug reports that come with an exact reproduction ("here is the replay"), and recordings of human games.
