# Next Steps: Play-Only Headless Pac-Man / Ms. Pac-Man for ML

A step-by-step plan for a headless `pacfan` that machine-learning tools can drive to learn to play, grounded in the current codebase.

[`scriptable.md`](scriptable.md) and [`machine.md`](machine.md) remain useful background (observation encodings, reward shaping, replay ideas). **Where they disagree with this document, this document wins.**

---

## What Changed in This Revision

The earlier plan drove the real `scene.Manager` with a "fast timing" profile and a no-op audio player. This revision narrows the scope to **the playing levels only**:

| Topic | Earlier plan | Now |
| --- | --- | --- |
| Scenes | Drive `scene.Manager`, collapse delays to 1 tick | `internal/scene` is **not imported**. No attract, ready, dying, round-cleared, intermission, or game-over scenes |
| Audio | `NoopPlayer`, move `RaylibPlayer` to a subpackage | `internal/audio` is **not imported** at all |
| Timing | `TimingArcade` / `TimingFast` profiles | Gone. One tick = one gameplay frame; nothing else exists |
| Maze on the wire | `rows` on reset, `compact_maze` dot lists per step | All mazes sent **once as text** in `hello`; steps carry only the tiles eaten |
| Artwork | `real_art` bool in the replay header | PNG **filename + SHA-256 + maze-layout hash** in `hello` and in every replay |
| Replay playback | Replay through `game.Engine` scenes, hash-compare headless vs windowed | Viewer runs the **same** `sim.Env` and draws its world with `render.Renderer` |
| Binaries | `pacfan -headless` and `cmd/pacfan-headless` | Only `cmd/pacfan-headless` serves the API; `pacfan` gains `-replay` later |
| Players / high score | Inherited from the game | Always 1 player; no high-score file I/O |

---

## Decisions to Lock In First

### 1. The simulator is `system.Pipeline` plus three transition rules

`internal/sim.Env` owns a `donburi.World` and calls `Pipeline.Update` once per tick. After each tick it applies the only transitions that matter, immediately and with zero dead ticks:

- **`GameState.PlayerDying`** → call `system.ConsumeLifeOrSwitchPlayer` (with `PlayerCount` forced to 1). Lives left: play resumes on the next tick from the spawn positions. No lives left: `terminated`.
- **`GameState.LevelComplete`** → call `system.AdvanceToNextLevel`. The next tick is the first tick of the new level (or `terminated` when `end_on_level_clear` is set).
- **`GameState.GhostEatenPause > 0`** → keep ticking inside the same `Step` call until it reaches 0. This is gameplay, not a between-level scene: eaten ghosts' eyes keep travelling home during the 61-tick freeze, so skipping it would change ghost positions. `Pipeline.Update` already contains this path (`internal/system/pipeline.go`); the agent is simply not asked for a decision during it.

Consequences worth knowing up front:

- **Sim ticks are the canonical timeline.** The windowed arcade game routes the ghost-eaten freeze through `GhostEatenFreezeScene`, which resumes the full pipeline one tick later than the pipeline's inline path. The two therefore drift by one tick per ghost eaten. That is fine because replays are played back by `sim.Env` itself (Decision 5), never by the scene manager. The windowed arcade game is left untouched.
- **Ready-screen input latching is dropped.** Not needed: the first `Step` after a reset, death, or level change applies its action on the first play tick.

### 2. Keep audio and Raylib out of the dependency graph, not just silent

Today `internal/system` imports `internal/audio` for one function (`RegisterAudioEventHandlers` in `internal/system/audio.go`), and `internal/audio/raylib.go` imports Raylib, so every package above `system` links Raylib.

Fix: move `RegisterAudioEventHandlers` into `package audio` (it already imports `component`; adding `event` creates no cycle) and call it from `internal/game/engine.go`. `internal/system` then has no audio import, and `internal/sim` depends only on:

```text
config  component  event  archetype  system  sheet
```

No `audio`, `scene`, `render`, `game`, or Raylib. A `make` target enforces this.

### 3. Mazes are parsed from the PNG once at startup and travel as text

`component.MazeData` already classifies 8×8 pixel masks into tiles (`LoadLevel`) and serialises them one character per tile (`Rows()`, `Cell.Char()`):

| Char | Tile | | Char | Tile |
| --- | --- | --- | --- | --- |
| `' '` | empty | | `.` | dot |
| `#` | wall | | `O` | power pellet |
| `-` | ghost-house door | | `@` | tunnel |

At startup the headless server decodes the game PNG with stdlib `image/png` and builds a `MazeSet`: all five layouts (indices `0..3` Ms. Pac-Man, `4` classic Pac-Man) as tile grids, each with its dot count and ghost-house geometry. After that the PNG is never touched again:

- Level loads copy an 868-byte template instead of sampling ~55,000 pixels.
- The `hello` response sends all five mazes as 31 strings of 28 characters (about 5 KB, once per connection).
- `reset` and level changes send only `maze_index`.
- Each `step` sends only `eaten`: the tile coordinates cleared during that step, taken from `event.DotEaten` / `event.PowerUpEaten` (which already carry tile `X`, `Y`). The client keeps its own copy of the grid. Walls, doors, and tunnels never change during a level.

### 4. The artwork file is part of the environment's identity

The tracked placeholder `assets/game.png` and the untracked original `assets/game-real.png` contain **different maze layouts** (see `assets/README.md`), so the PNG decides the rules of the world, not just how it looks. A policy trained or a replay recorded on one cannot be evaluated or replayed on the other.

The server therefore reports, in `hello` and in every replay header:

- `art.file` — base filename of the game PNG (e.g. `game.png`), so a viewer knows which graphics to load.
- `art.sha256` — hash of the PNG bytes.
- `maze_hash` — hash of the five parsed maze texts. This is the real compatibility key: two PNGs with different pixels but identical layouts are interchangeable for simulation.

Source of the PNG: the embedded asset chosen by the existing preference (`game-real.png` if present, else `game.png`), overridable with `-art FILE` to read any PNG from disk at startup. The font PNG and shader are irrelevant headless.

### 5. A replay is a seed plus an action list, replayed by `sim.Env`

Because the simulator is deterministic, a game is fully described by `{maze_hash, pacman, level, seed, options}` plus the action applied on each tick. Replays are recorded inside `sim.Env` and played back by constructing a fresh `sim.Env` and feeding the actions back in.

The windowed replay viewer is **`sim.Env` + `render.Renderer.DrawWorld`**: no `scene` package, no audio. Death animations, "READY!" banners, and maze flashes are optional cosmetic holds the viewer draws *between* sim ticks without advancing the simulation.

### 6. Seeding

`PRNGData` fills a ROM table from a fixed seed (`42`) and `Reset()` rewinds its index on every life and level. With one fixed seed, ghost behaviour is a pure function of the player's actions, and an agent can memorise a route instead of learning to play. `reset` takes a `seed` that reseeds the ROM table: training varies it per episode, evaluation uses a fixed seed list.

### 7. Action numbering follows `config.Direction`

`0=right, 1=left, 2=up, 3=down, 4=noop` everywhere (Go, JSON, replays, Python). Ghost order is `component.ActorID` order: `blinky, pinky, inky, clyde`.

---

## Target Package Layout

```text
assets/
  embed.go                 # package assets: //go:embed, GameAssetPath (pure Go)
cmd/
  pacfan-headless/main.go  # NDJSON server; zero Raylib, audio, or scene imports
internal/
  audio/                   # + RegisterAudioEventHandlers (moved from system); windowed game only
  component/
    maze.go                # + MazeTemplate, MazeSet, ParseMazeSet, template-backed LoadLevel
    prng.go                # + InitWithSeed
  sheet/                   # Pure-Go PNG decode -> IsPixelLit, file name, SHA-256
  sim/                     # Env, Action, events, HashWorld, replay record/read
  api/                     # NDJSON protocol, Session, stdio server
ml/                        # Python Gymnasium env, baselines, trainer
main.go                    # Windowed pacfan; later: -replay
```

---

## Wire Protocol Sketch (NDJSON, one request line → one response line)

Values below are illustrative.

```jsonc
// -> {"cmd":"hello"}
// <- sent once; everything static lives here
{"ok":true,"protocol":1,
 "art":{"file":"game.png","sha256":"9f2c..."},"maze_hash":"51ab...",
 "width":28,"height":31,
 "mazes":[{"index":0,"dots":220,"rows":["############################","#......##..........##......#", "..."]}, "..."],
 "actions":["right","left","up","down","noop"],
 "ghosts":["blinky","pinky","inky","clyde"]}

// -> {"cmd":"reset","pacman":false,"level":1,"seed":7,"episodic_life":false,
//     "end_on_level_clear":false,"max_ticks":0,"record":"runs/x/ep-0001.pfr"}
// <- {"ok":true,"tick":0,"maze_index":0,"state":{...}}

// -> {"cmd":"step","action":2,"mode":"tile"}        // mode: "frames" (with "frames":N) | "tile" | "intersection"
// <- {"ok":true,"tick":8,"ticks":8,"reward":10,"terminated":false,"truncated":false,
//     "events":["dot"],"eaten":[[6,23]],"state":{...}}
//    "maze_index" appears again only when the level changed during this step

// -> {"cmd":"observe","maze":true}                  // debug/resync: current grid as rows
// -> {"cmd":"close"}
```

`state` is small and has no maze in it:

```jsonc
{"level":1,"score":10,"lives":3,"dots_remaining":219,
 "wave":"scatter","fright_frames":0,"ghosts_eaten":0,
 "player":{"x":108,"y":184,"dir":1,"legal":[true,true,false,false]},
 "ghosts":[{"x":108,"y":88,"dir":1,"state":"scatter"}, "..."],
 "fruit":{"active":false}}
```

`reward` is the raw score delta for the step; shaping belongs in the Python wrapper. `events` values: `dot`, `power`, `ghost`, `fruit`, `extra_life`, `death`, `level_clear`.

---

## Phase 1: Play-Only Simulation Core

**Goal:** a pure-Go `sim.Env` that plays full games deterministically with no Raylib, audio, or scene code linked.

### Step 1.1 — Cut the audio/Raylib link out of `internal/system`
- [x] Move `RegisterAudioEventHandlers` from `internal/system/audio.go` to `internal/audio`; update `internal/game/engine.go` and any tests.
- [x] Add `assets/embed.go` (`package assets`) exposing the embedded FS and a pure-Go `GameAssetPath`; `render.GameAssetPath` delegates to it. Update `main.go` to use `assets.FS`.
- [x] **Verify:** `go list -deps ./internal/system ./internal/archetype | grep -E 'raylib|internal/audio'` prints nothing; `make check` passes.

### Step 1.2 — Seedable PRNG and start-at-level
- [x] `internal/component/prng.go`: `DefaultPRNGSeed = 42`, `InitWithSeed(seed int64)`; `Init()` delegates to it so the windowed game is unchanged.
- [x] `internal/system/level.go`: `StartGameAtLevel(w, level, isPixelLit)` (level clamped to `1..255`); `RestartGame` becomes `StartGameAtLevel(w, 1, ...)`.
- [x] Tests: same seed → same byte sequence; different seeds diverge; starting at level N loads the right maze index and level spec.

### Step 1.3 — Pure-Go sheet and text maze set
- [x] `internal/sheet`: `Load(fsys fs.FS, path string)` and `LoadFile(path string)` decode the PNG once into a lit-pixel bitmap; expose `IsPixelLit(x, y)` (same rule as `render.Art.IsPixelLit`: any of R, G, B non-zero), `Name()` (base filename), and `SHA256()`.
- [x] `internal/component/maze.go`: split the pixel classification out of `LoadLevel` into `ParseMazeTemplate(mazeIndex, isPixelLit) MazeTemplate` (grid, dot count, ghost house, with tunnels marked) and `ParseMazeSet(isPixelLit) *MazeSet` for all five, plus `MazeSet.Hash()` and `MazeTemplate.Rows()`.
- [x] `MazeData.UseTemplates(*MazeSet)`: when set, `LoadLevel` copies the template and ignores `isPixelLit`. This keeps the `isPixelLit` parameter threaded through `internal/system/level.go` and the scenes unchanged, so the windowed game and its ~3,000 lines of tests are not disturbed.
- [x] Tests: template-backed `LoadLevel` equals pixel-sampled `LoadLevel` tile-for-tile for all five mazes; text round-trips (`Rows()` → parse → identical grid). Replace the duplicated `png.Decode` helpers in `system_test.go`, `scene_test.go`, and `renderer_test.go` with `internal/sheet` where it is a drop-in.

### Step 1.4 — `internal/sim`
- [x] `input.go`: `Action` (`Right=0 … Noop=4`) and `ScriptedInput` implementing `system.InputProvider` (never sets pause, mute, or start keys).
- [x] `events.go`: subscribe to `DotEaten`, `PowerUpEaten`, `GhostEaten`, `FruitEaten`, `ExtraLifeAwarded`, `PlayerEaten`, `LevelCompleted`; collect an event bitmask and the eaten-tile list per step into reusable buffers.
- [x] `env.go`:
  - `Config{Mazes *component.MazeSet, PacmanMode, StartLevel, Seed, EpisodicLife, EndOnLevelClear, MaxTicks}`.
  - `NewEnv(cfg)` builds the world the way `game.NewEngine` does minus art, audio, renderer, high score, and scenes: `CreateGameState`, `CreateMaze` (+ `UseTemplates`), `CreatePRNG`, `SpawnPlayer`, `SpawnAllGhosts`, `RegisterEventHandlers`, `NewPipeline(scriptedInput)`.
  - `Reset(opts)`, `Step(action, mode, frames) StepResult`, `LegalActions() [4]bool`, `World()`, `Tick()`.
  - One internal `tick()` that runs `Pipeline.Update` and then applies the three transition rules from Decision 1. Everything else (step modes, replay, viewer) goes through it.
  - Step modes: `frames` (N ticks), `tile` (until Pac-Man is centred on a new tile, or stopped by a wall), `intersection` (until a centred tile with a choice other than straight-on or reverse). All modes stop early on death, level clear, or termination, always run through a ghost-eaten freeze, and have a tick cap so pushing into a wall cannot hang.
  - `EpisodicLife`: a death sets `terminated` for the learner; the next `Reset` continues the same game if lives remain.
- [x] `hash.go`: `HashWorld(w) uint64` (FNV-1a over gameplay fields of `GameState`, the maze grid, actor transforms / motion / ghost state / dot counters, fruit state, `PRNG.Index`; excludes `Zoom` and `HighScore`).

### Step 1.5 — Determinism, rule coverage, benchmark
- [x] Audit `internal/system` and `internal/component` for nondeterminism. Already checked: no `time.Now`, no map iteration, and `math/rand` is used only to fill the PRNG ROM. Still to confirm: donburi query iteration order is stable across worlds built the same way (the per-tick hash test below is the proof).
- [x] Determinism test: two `Env`s, same seed and action stream, 5,000+ ticks, Pac-Man and Ms. Pac-Man; `HashWorld` equal on every tick. Different seeds must diverge.
- [x] Rule tests: dots, power pellets, ghost eaten (freeze runs inside one step; eyes move), fruit, extra life, death with and without `EpisodicLife`, level clear → next maze index, `MaxTicks` truncation, wall-push never hangs in any step mode.
- [x] `BenchmarkEnvTick` (ticks/sec, allocs/op) as the baseline. Known allocation source to fix if it shows up: `donburi.NewQuery` called per use in `internal/system/level.go`.
- [x] Makefile: `check-headless-deps` fails if `go list -deps ./internal/sim` matches `raylib|internal/audio|internal/scene|internal/render`; add it to `check`.

**Done when:** `make check` passes including the dependency check and per-tick determinism tests, and a Go test plays a scripted game from reset to game over.

---

## Phase 2: Wire Protocol, Headless Binary, and Replay Recording

**Goal:** `pacfan-headless` plays full episodes over piped NDJSON and can write a replay file for any episode.

### Step 2.1 — `internal/api`
- [x] `protocol.go`: request/response structs for `hello`, `reset`, `step`, `observe`, `close` per the sketch above. `action` accepts `0..4` or `"right"|"left"|"up"|"down"|"noop"`.
- [x] `session.go`: owns one `sim.Env`; builds the cached `hello` payload from the `MazeSet` and sheet identity; emits `maze_index` on reset and level change only.
- [x] `server.go`: `ServeNDJSON(r, w, session)` with `bufio` + `encoding/json`; malformed input yields `{"ok":false,"error":...}` and the loop continues; exits cleanly on `close` or EOF.

### Step 2.2 — Replay file (`internal/sim`, extension `.pfr`)
A small line-oriented text file, easy to read from Python and to diff:

```text
{"pfr":1,"art":{"file":"game.png","sha256":"..."},"maze_hash":"...","pacman":false,"level":1,"seed":7,"options":{...},"meta":{"run_id":"...","checkpoint":"...","episode":12,"git_commit":"..."}}
a 2x8 1x24 3x16 ...          # run-length encoded: action x ticks
h 600 9c41d2e07a3b1f55       # HashWorld checkpoint at tick 600
a ...
{"end":"game_over","ticks":5321,"score":18340,"level":3,"deaths":3}
```

- [x] `Recorder`: appends the action for every tick inside `Env.tick()` (including freeze ticks), a hash line every 600 ticks, and the footer on termination/close.
- [x] `ReadReplay` + `Env.Replay(r)`: rejects a `maze_hash` mismatch outright, warns on an `art.sha256`-only mismatch, and reports the first tick whose hash differs.
- [x] `reset` accepts `record` (path) and `meta` (opaque object copied into the header).

### Step 2.3 — `cmd/pacfan-headless`
- [x] Flags: `-art FILE` (default: embedded `game-real.png` else `game.png`), `-pacman`, `-level`, `-seed` (defaults for `reset`), `-verify FILE.pfr` (replay a file headlessly and exit non-zero on a hash mismatch), `-dump-mazes` (print the five text mazes and hashes, then exit).
- [x] Makefile: `build-headless`; extend `check-headless-deps` to `./cmd/pacfan-headless ./internal/api`.

### Step 2.4 — Tests and benchmark
- [x] Round-trip `hello → reset → step… → close` through `ServeNDJSON`; malformed JSON; unknown command; step before reset.
- [x] Client-side maze reconstruction: apply `eaten` to the `hello` maze for a whole game and compare with `observe {maze:true}` at every step.
- [x] Record → replay in a fresh `Env` → identical `HashWorld` on every tick, both modes. Check in 2–3 tiny `.pfr` fixtures recorded on the tracked `game.png` (dots, ghost eaten, fruit, death, level clear) so rule regressions fail `go test ./...`.
- [x] `BenchmarkNDJSONStep`: end-to-end steps/sec and bytes per step response.

**Done when:** `go list -deps ./cmd/pacfan-headless | grep -E 'raylib|audio|scene'` is empty, a shell pipe can play a game to game over, and `pacfan-headless -verify` passes on the checked-in fixtures.

---

## Phase 3: Python Gymnasium Environment and Baselines

**Goal:** standard RL libraries train against `pacfan-headless`, and a first policy visibly learns.

### Step 3.1 — `ml/` scaffold
- [ ] `ml/pyproject.toml`: `gymnasium`, `numpy`; `[train]` extra with `torch`, `stable-baselines3`, `sb3-contrib`, `tensorboard`. `ml/README.md` with build, venv, sanity-check, and training commands.

### Step 3.2 — `ml/pacfan_env.py`
- [ ] `PacFanEnv(gym.Env)` over `subprocess.Popen`: sends `hello` once, caches the five mazes as NumPy arrays, selects by `maze_index`, and applies `eaten` each step. Options: `binary`, `art`, `pacman`, `start_level`, `step_mode`, `frames_per_step`, `episodic_life`, `end_on_level_clear`, `max_ticks`, `obs_mode`, `record_dir`.
- [ ] `reset(seed=...)` forwards the Gymnasium seed as the game seed; with no seed given, draw one from the env's RNG so episodes differ (Decision 6).
- [ ] `action_masks()` from `state.player.legal` (for `MaskablePPO`); context-manager cleanup; surface child stderr if the process dies.
- [ ] Observation encoders: spatial grid `(C, 31, 28)` float32 (walls/door, dots, power, tunnels, player, per-ghost dangerous / frightened / eyes, fruit, plus a small scalar vector) and a BFS feature vector (path distance and first step to nearest dot, power pellet, fruit, each ghost). See `scriptable.md` §5.1.
- [ ] Reward wrapper: default `0.01 × score delta`; optional death penalty, level-clear bonus, step penalty.

### Step 3.3 — Validation and zero-ML baselines
- [ ] `ml/tests/test_env.py`: `check_env`, same seed + same actions → same trajectory, `AsyncVectorEnv` with several workers.
- [ ] `ml/baselines.py`: `RandomLegalAgent` and `GreedyBFSAgent` (nearest dot or frightened ghost, avoid dangerous ghosts within K tiles). Log mean/max score, levels cleared, ticks survived, and env steps/sec at 1, 4, 8 workers.

### Step 3.4 — Training and evaluation
- [ ] `ml/train_ppo.py`: `MaskablePPO` with a small CNN sized for 31×28 grids, run directory:
  ```text
  runs/<run-id>/
    run.json          # hyperparameters, reward config, git commit, art file + maze_hash
    episodes.jsonl    # per-episode metrics
    checkpoints/
    eval/ckpt-XXXX/   # fixed-seed eval metrics + .pfr replays
    best/
  ```
- [ ] Fixed-seed evaluation at each checkpoint (full games, deterministic policy, recorded to `.pfr`). `ml/eval.py` loads a checkpoint and prints summary statistics.

**Done when:** `pytest ml/tests` passes, `GreedyBFSAgent` reliably clears most of level 1, and a short PPO run writes the run directory and beats `RandomLegalAgent` on the fixed eval seeds.

---

## Phase 4: Watching Recorded Games in the Real Window

**Goal:** `pacfan -replay FILE_OR_DIR` shows any `.pfr` with the real graphics.

- [ ] `internal/game`: a `ReplayViewer` that builds a `sim.Env` from the replay header, feeds it the recorded actions at the arcade clock rate, and draws `env.World()` with the existing `render.Renderer.DrawWorld`. Set `GameState.Zoom` from `-zoom` (it is excluded from `HashWorld`).
- [ ] Art selection: load the embedded PNG whose base name equals `art.file` (or `-art FILE`); parse its `MazeSet` and refuse to play if `maze_hash` differs from the header, naming the file the replay needs.
- [ ] `main.go` flags: `-replay FILE_OR_DIR` (directory = sorted playlist), `-replay-speed N` (`0.25..16`). No audio initialisation in this mode.
- [ ] Controls read straight from Raylib: `Space` pause, `→` single tick while paused, `+`/`-` speed, `N`/`P` next/previous file. HUD line with run id, checkpoint, episode, tick, speed. Hash checkpoints verified as playback passes them.
- [ ] Optional polish: cosmetic holds on death and level clear (draw with `DrawOptions{PlayerDying…}` / `FlashMaze` for a moment without ticking the sim).
- [ ] Optional: `-sim` human play mode — the same viewer loop with `render.RaylibInput` instead of a replay and `-record FILE`, giving human demonstrations in exactly the format and timeline the agents use (for imitation learning).
- [ ] Optional: live spectating. `pacfan -api tcp://127.0.0.1:PORT` runs the same viewer loop with actions arriving from an `api.Session` on a loopback socket; the main thread keeps rendering and ticks only when a `step` arrives.

**Done when:** evaluation replays from Phase 3 play in the window with no hash mismatches, and a replay recorded on different maze layouts is refused with a clear message.

---

## Phase 5: Throughput (Only If Profiling Says So)

Run this phase only if Phase 3 profiling shows the environment, not the neural network, is the bottleneck.

- [ ] Profile a training step across 8–32 workers: Go simulation vs JSON vs pipe I/O vs PyTorch.
- [ ] Remove per-tick allocations found by `go test -cpuprofile/-memprofile` on `internal/sim`.
- [ ] Go-native observation encoders writing into caller-owned `[]float32`, and `sim.VecEnv` stepping N worlds on a goroutine pool with auto-reset.
- [ ] Bridge, one of: a `step_batch` command with length-prefixed binary frames over stdio (no CGo), or a `c-shared` library wrapped with `ctypes` (one C call per batched step).
- [ ] Extras: MP4/GIF export of replays via an off-screen render texture and `ffmpeg`; side-by-side replays on one seed; position and death heat maps over a replay directory.

---

## Execution Order

| Milestone | Deliverables | Gate |
| --- | --- | --- |
| **1A** (1.1–1.3) | Audio-free `system`, `assets.FS`, seedable PRNG, `internal/sheet`, text `MazeSet` | No `raylib`/`audio` under `internal/system`; template/pixel parity test; `make check` |
| **1B** (1.4–1.5) | `sim.Env`, events, `HashWorld`, benchmark | Per-tick determinism + rule tests; dependency check |
| **2** (2.1–2.4) | `internal/api`, `.pfr` record/verify, `cmd/pacfan-headless` | Piped game to game over; fixtures verify; client maze reconstruction test |
| **3A** (3.1–3.3) | `PacFanEnv`, encoders, random + greedy baselines | `check_env` and vector-env tests; baseline scores logged |
| **3B** (3.4) | PPO trainer, run directory, fixed-seed eval with replays | Beats random baseline on fixed seeds |
| **4** | `pacfan -replay` viewer (+ optional human record, live spectate) | Eval replays play back with zero hash mismatches |
| **5** | Profile-guided batching / native bridge | Measured steps/sec gain; Phase 1–4 tests still pass |

Milestones 1A through 2 are the critical path: after them any language that can write a line of JSON to a pipe can train on the game, and every episode it plays can be saved and later watched.
