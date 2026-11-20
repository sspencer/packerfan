# Next Steps: Headless Pac-Man / Ms. Pac-Man API & ML Training Plan

This document turns the design ideas in [`scriptable.md`](scriptable.md) and [`machine.md`](machine.md) into a concrete, step-by-step engineering plan grounded in the current `pacfan` codebase.

It follows the **four implementation phases from `scriptable.md`** while incorporating the **zero-Raylib build guarantee, cached maze loading, unified scene-driven simulation, deterministic `.pfr` replays, and native batching path from `machine.md`**.

---

## Reconciling `scriptable.md` and `machine.md` (Decisions to Lock In First)

Before writing code, five small architectural choices unify both documents so there is only **one** simulation engine to maintain:

1. **Layer `internal/api` on top of a shared pure-Go `internal/sim` core.**
   - `internal/sim` owns headless world setup, `TimingProfile`, `Env` (`Reset`, `Step`, `Observe`, `LegalActions`), event collection, world-state hashing, and replay recording/reading.
   - `internal/api` wraps `internal/sim.Env` with the lockstep NDJSON wire protocol (`stdio`, Unix socket, TCP) and `RemoteInput` for live spectating.
   - This lets both Python NDJSON agents (`scriptable.md`) and native Go / `VecEnv` / `c-shared` agents (`machine.md`) share one deterministic core.

2. **Drive the real `scene.Manager` with a `FastTiming` profile — and keep `GhostEatenPause`.**
   - `machine.md` notes that driving `scene.Manager` (instead of duplicating death/level-advance transitions around `system.Pipeline`) prevents rule drift, and keeping the 61-frame `GhostEatenFreezeScene` matters because previously eaten ghost eyes continue moving via `Pipeline.UpdateEatenGhosts`.
   - In `FastTiming`, non-interactive delays (`PlayerOneDuration`, `ReadyDuration`, `RoundClearedHoldFrames`, `RoundClearedFlashes`, `PlayerDyingHoldFrames`, `PlayerDyingFrameTicks`, `PlayerDyingPostFrames`) are collapsed to `1` tick, intermissions return `nil`, and `Env.Step` automatically fast-forwards through non-interactive scenes (including `GhostEatenFreezeScene`) without asking the agent for decisions—while still recording exact ticks so headless games replay **frame-for-frame identically** in the windowed Raylib player.

3. **Break the hidden Raylib dependency in `internal/audio`.**
   - Currently, `internal/audio/raylib.go` sits in `package audio` without a subpackage or build tag, which causes `internal/audio`, `internal/system`, and `internal/scene` to transitively import `github.com/gen2brain/raylib-go/raylib`.
   - Moving `RaylibPlayer` into a subpackage (e.g., `internal/audio/raylibaudio`) makes `internal/audio`, `internal/system`, `internal/scene`, `internal/sim`, and `internal/api` **100% pure Go with zero Raylib/CGo/OpenGL linkage**.

4. **Standardize action numbering on `config.Direction`.**
   - `internal/config/direction.go` already defines `Right=0, Left=1, Up=2, Down=3`.
   - Using `0=right, 1=left, 2=up, 3=down, 4=noop` everywhere (matching `scriptable.md`) avoids error-prone translation tables between Go, JSON, replays, and Python.

5. **Provide both CLI modes cleanly.**
   - `cmd/pacfan-headless` (pure Go, zero Raylib, deployable to headless Linux/Docker training servers).
   - `main.go` (`pacfan` binary) also accepts `-headless -api stdio` (skipping `rl.InitWindow` / audio initialization) for local convenience, plus `-api tcp://...`, `-record`, and `-replay` for windowed spectating and replay playback.

---

## Target Package Architecture

```text
assets/
  embed.go                       # Shared //go:embed FS + pure-Go ExampleArtFS (no Raylib)
cmd/
  pacfan-headless/main.go        # Pure-Go headless binary (zero Raylib imports; CI-verified)
internal/
  audio/
    audio.go                     # Pure-Go Player interface, NoopPlayer, loop selection
    raylibaudio/raylib.go        # Moved RaylibPlayer (only imported by windowed game)
  component/
    prng.go                      # + NewPRNGDataWithSeed(seed), InitWithSeed(seed)
    maze.go                      # + LoadTemplate / CachedMaze support
  sheet/                         # Pure-Go image/png spritesheet decoder & 5-maze grid cache
  sim/                           # Core headless Env, TimingProfile, EventCollector, StateHash
    replay/                      # .pfr binary RLE replay format, RecordingInput, ReplayInput
  api/                           # NDJSON protocol, Session, stdio/unix/tcp Server, RemoteInput
ml/                              # Python Gymnasium env, baseline agents, MaskablePPO trainer
main.go                          # Windowed pacfan + -headless, -api, -record, -replay flags
```

---

## Phase 1: Core Headless API

**Goal:** Run `pacfan` headlessly with zero Raylib dependencies, deterministic seeding, cached pure-Go maze loading, a lockstep NDJSON API over `stdio`, and automated determinism + throughput benchmarks.

### Step 1.1 — Decouple Raylib from `internal/audio` and share embedded assets
- [ ] Move `internal/audio/raylib.go` into `internal/audio/raylibaudio/raylib.go` (`package raylibaudio`) and update `internal/game/engine.go` to call `raylibaudio.New(...)`.
- [ ] Add `assets/embed.go` (`package assets`) with `//go:embed *.png *.fs pacman ms-pacman` (or `//go:embed *`) exposing `assets.FS` and pure-Go `assets.GameAssetPath(fs.FS)` / `assets.FontAssetPath(fs.FS)` helpers that prefer `game-real.png` / `text-real.png` and fall back to the placeholder `game.png` / `text.png` (migrating `GameAssetPath` and `FontAssetPath` out of `internal/render/art.go`, keeping thin aliases if callers reference them).
- [ ] Update `main.go` to use `assets.FS`.
- [ ] **Verify:** Run `go list -deps ./internal/audio ./internal/system ./internal/scene | grep raylib` and confirm **no output**; run `make check`.

### Step 1.2 — Add configurable PRNG seeding and start-level support
- [ ] In `internal/component/prng.go`, add `DefaultPRNGSeed = 42`, `NewPRNGDataWithSeed(seed int64) PRNGData`, and `(p *PRNGData) InitWithSeed(seed int64)` (with existing `Init()` delegating to `InitWithSeed(DefaultPRNGSeed)`).
- [ ] In `internal/archetype/world.go`, add `CreatePRNGWithSeed(w donburi.World, seed int64) *donburi.Entry` (keeping `CreatePRNG(w)` as default seed `42`).
- [ ] In `internal/system/level.go`, add `StartGameAtLevel(w donburi.World, level, playerCount int, isPixelLit func(x, y int) bool)` (clamping `level` to `1..255`) so curriculum learning can start episodes on any level without duplicating `RestartGame` logic.
- [ ] Add unit tests verifying identical seeds produce identical `PRNG` sequences and different seeds diverge deterministically.

### Step 1.3 — Pure-Go spritesheet decoder and cached maze loader (`internal/sheet`)
- [ ] Create `internal/sheet/sheet.go` using stdlib `image/png` + `fs.FS` to decode the spritesheet chosen by `GameAssetPath` (`assets/game-real.png` when present, otherwise `assets/game.png`) once per process.
- [ ] Expose `IsPixelLit(x, y int) bool` matching `render.Art.IsPixelLit` (`r > 0 || g > 0 || b > 0`).
- [ ] Pre-parse and cache all 5 maze layouts (`mazeIndex` `0..3` for Ms. Pac-Man, `4` for classic Pac-Man) as read-only `[config.GameHeight][config.GameWidth]component.Cell` templates plus their `dotsRemaining` count and `GhostHouse` geometry.
- [ ] Add `(m *MazeData) LoadCachedLevel(level int, loader func(mazeIndex int) (MazeTemplate, bool)) int` (or an optional fast path in `internal/component/maze.go`) so headless `Reset` copies 868 bytes instead of sampling ~55,000 pixels.
- [ ] Replace duplicated `png.Decode` helpers in `internal/system/system_test.go` and `internal/scene/scene_test.go` with `internal/sheet` where appropriate, and add a parity test confirming cached templates match pixel-sampled `MazeData.LoadLevel` tile-for-tile across all 5 mazes.

### Step 1.4 — Build the core headless simulation engine (`internal/sim`)
- [ ] Create `internal/sim/input.go`:
  - `Action` enum (`ActionRight=0, ActionLeft=1, ActionUp=2, ActionDown=3, ActionNoop=4`).
  - `ScriptedInput` implementing `system.InputProvider` (holds the next `Action`, zero allocations).
- [ ] Create `internal/sim/timing.go`:
  - `TimingProfile` enum (`TimingArcade`, `TimingFast`) and helper `ApplyTiming(ctx *scene.Context, profile TimingProfile)` that configures `scene.Context` overrides (`ReadyDuration=1`, `PlayerOneDuration=1`, `RoundClearedHoldFrames=1`, `RoundClearedFlashes=1`, `PlayerDyingHoldFrames=1`, `PlayerDyingFrameTicks=1`, `PlayerDyingPostFrames=1`, `IntermissionFactory=nil` for `TimingFast`).
- [ ] Create `internal/sim/events.go`:
  - `EventFlags` bitmask + ordered `[]string` event recorder subscribed to `event.DotEaten`, `event.PowerUpEaten`, `event.GhostEaten`, `event.FruitEaten`, `event.ExtraLifeAwarded`, `event.PlayerEaten`, `event.LevelCompleted` on the world's event bus.
- [ ] Create `internal/sim/hash.go`:
  - `HashWorld(w donburi.World) uint64` (FNV-1a 64-bit hash over `GameState` gameplay fields, `MazeData.Grid`, player & ghost transforms/states/motion/dot counters, fruit state, and `PRNG.Index`) for determinism verification and replay desync detection.
- [ ] Create `internal/sim/env.go`:
  - `EnvConfig`: `PacmanMode`, `StartLevel`, `Seed`, `Timing`, `EpisodicLife`, `MaxSteps`, `MaxTicks`, `UseExampleArt`.
  - `StepMode`: `StepFrames`, `StepTile`, `StepIntersection`.
  - `Env`: `NewEnv(sheet *sheet.Sheet, cfg EnvConfig) *Env`, `Reset(opts ResetOptions)`, `Step(action Action, mode StepMode, frames int) StepResult`, `LegalActions() [4]bool`, `World() donburi.World`, `Tick() int`, `StepCount() int`.
  - Wire headless composition root: `Drawer: nil` (or `NoopDrawer`), `Audio: audio.NoopPlayer{}`, `Input: ScriptedInput`, no high-score disk I/O, start directly at `scene.NewReadyScene` (bypassing `AttractScene`).
  - Handle non-interactive scenes cleanly during `Step`: advance through `ReadyScene`, `GhostEatenFreezeScene`, `PacmanDyingScene`, and `RoundClearedScene` automatically while breaking immediately on `player_eaten`, `level_completed`, or `GameOver` so the agent receives timely rewards/terminations.
  - Implement `StepTile` (advance until Pac-Man reaches a new centered tile or stops at a wall / dies / clears level) and `StepIntersection` (advance until Pac-Man reaches a centered tile with ≥2 legal non-reverse moves or hits a corner/wall/event), with a safety tick cap to prevent infinite loops when stalled against a wall.

### Step 1.5 — Implement the NDJSON wire protocol & `stdio` server (`internal/api`)
- [ ] Create `internal/api/protocol.go`:
  - Request structs: `Command` dispatching `"reset"`, `"step"`, `"observe"`, `"close"`.
  - Accept both integer (`0..4`) and string (`"right"`, `"left"`, `"up"`, `"down"`, `"noop"`) values for `action`.
  - Response struct (`ok`, `error`, `step`, `tick`, `reward`, `terminated`, `truncated`, `events`, `state`) and `Snapshot(w donburi.World, compactMaze bool, includeStaticMaze bool) StateSnapshot` matching the schema in `scriptable.md` §4.2:
    - Game globals (`level`, `pacman_mode`, `score`, `high_score`, `lives`, `dots_eaten`, `dots_remaining`, `wave_mode`, `wave_frames`, `is_fright`, `fright_frames`, `num_ghosts_eaten`, `elroy_stage`).
    - `player` (`x`, `y`, `tile_x`, `tile_y`, `dir`, `dot_pause`, `legal_actions`).
    - `ghosts` in canonical order (`blinky`, `pinky`, `inky`, `clyde` — verify order matches `component.ActorID` and document explicitly in the schema).
    - `fruit` (`active`, `moving`, `symbol`, `points`, `x`, `y`, `tile_x`, `tile_y`, `timer`).
    - `maze` (`width`, `height`, `rows` via `MazeData.Rows()`, plus `compact_maze` mode sending `dots` & `power` coordinates on non-reset/non-level-change steps).
- [ ] Create `internal/api/session.go`: manages one `sim.Env`, tracks whether static maze walls need to be sent (on `reset` and level change), and executes `Handle(req Request) Response`.
- [ ] Create `internal/api/server.go`: `ServeNDJSON(r io.Reader, w io.Writer, session *Session) error` using `bufio.Scanner` (with raised max token size if needed) + `bufio.Writer` + `encoding/json`, returning clean JSON error responses on malformed input and exiting cleanly on `"close"` or `EOF`.

### Step 1.6 — Wire CLI flags and dedicated pure-Go headless binary
- [ ] Create `cmd/pacfan-headless/main.go`:
  - Flags: `-api` (default `stdio`), `-pacman`, `-seed`, `-level`, `-timing` (`fast`|`arcade`).
  - Zero Raylib imports.
- [ ] Update `main.go` (`pacfan`) to support `-headless`, `-api`, and `-seed` flags so `pacfan -headless -api stdio` works out of the box without calling `rl.InitWindow` or initializing audio.
- [ ] Update `Makefile` with `build-headless` and a `check-headless-deps` target asserting `go list -deps ./cmd/pacfan-headless ./internal/sim ./internal/api | grep raylib` returns empty.

### Step 1.7 — Determinism audit, tests, and throughput benchmarks
- [ ] Audit `internal/system`, `internal/component`, and `internal/scene` for nondeterminism (map iteration order, `time.Now()`, global `math/rand`, entity iteration order). Document any findings and fix before locking hashes.
- [ ] Add `internal/sim/env_test.go` and `internal/api/api_test.go`:
  - **Determinism test:** Run two independent `Env` instances with identical seed and action sequence for 5,000+ ticks in both Pac-Man and Ms. Pac-Man modes; assert identical `HashWorld` on **every** tick.
  - **Rule coverage tests:** Verify dot eating, power pellets, ghost eating (including 61-tick eye movement), fruit spawn/eat, extra life, death (`episodic_life` true vs false), level clear, and `max_steps` truncation.
  - **Step mode tests:** Verify `frames`, `tile`, and `intersection` modes terminate cleanly and never hang when pushing into a wall.
  - **Protocol tests:** Round-trip NDJSON `reset` → `step` → `observe` → `close`, malformed JSON error handling, `compact_maze` output.
  - **Benchmarks:** `BenchmarkEnvStep` (raw Go ticks/sec and allocs/op) and `BenchmarkNDJSONStep` (end-to-end JSON round-trip steps/sec) to establish baseline numbers.

**Phase 1 Done When:**
- `go list -deps ./cmd/pacfan-headless | grep raylib` produces zero matches.
- `pacfan-headless -api stdio` (and `-pacman`) runs full episodes via piped JSON commands.
- `make check` passes with determinism tests and benchmark baselines recorded.

---

## Phase 2: Python Gymnasium Environment & Baseline Agents

**Goal:** Connect standard Python RL libraries (Stable-Baselines3 / CleanRL / PyTorch) to `pacfan-headless`, validate the pipeline with fast scripted/heuristic baselines, and train an initial policy that demonstrably learns to clear dots and evade ghosts.

### Step 2.1 — Python package scaffold (`ml/`)
- [ ] Create `ml/pyproject.toml` (or `ml/requirements.txt`) pinning `gymnasium`, `numpy`, and optional `[train]` extras (`torch`, `stable-baselines3`, `sb3-contrib`, `tensorboard`).
- [ ] Create `ml/README.md` documenting how to build `pacfan-headless`, set up a virtualenv, run sanity checks, and start training.

### Step 2.2 — Implement the Gymnasium wrapper (`ml/pacfan_env.py`)
- [ ] Implement `PacFanEnv(gym.Env)` communicating with `pacfan-headless -api stdio` over `subprocess.Popen` (`stdin`/`stdout` line-buffered):
  - Configurable `binary`, `pacman`, `start_level`, `step_mode` (`"frames"` | `"tile"` | `"intersection"`), `frames_per_step`, `episodic_life`, `compact_maze`, `max_steps`, `obs_mode` (`"grid"` | `"features"`), and reward shaping parameters.
  - Support `action_masks() -> np.ndarray` method (required by `sb3-contrib`'s `MaskablePPO`) derived directly from `state["player"]["legal_actions"]`.
  - Support automatic subprocess cleanup in `close()`, context manager (`__enter__`/`__exit__`), and clear stderr diagnostics if the child process exits unexpectedly.
- [ ] Implement both observation encoders from `scriptable.md` §5.1 / `machine.md` §3.2:
  - **Spatial Grid `(C, 31, 28)`** (`float32`): walls/door, dots, power pellets, tunnels, player, per-ghost dangerous channels + previous-tile velocity/heading trail, frightened ghosts (scaled by `fright_frames`), returning eyes, active fruit — plus a `Dict` observation option appending normalized global scalars (`fright_frames`, `wave_mode`, `dots_remaining`, `player.dir` one-hot, `lives`, `level`).
  - **BFS Feature Vector** (`float32`): shortest maze-path distances & first-step directions to nearest dot, power pellet, active fruit, and each ghost (dangerous vs frightened), for fast MLP / neuroevolution / sanity baselines.
- [ ] Implement configurable reward shaping wrapper (`gym.RewardWrapper`):
  - Default scaled arcade score (`0.01 * score_delta`).
  - Optional death penalty (`-5.0`), level-clear bonus (`+10.0`), and tiny step penalty (`-0.01`).

### Step 2.3 — Validation suite and heuristic baselines (`ml/baselines.py`)
- [ ] Run `gymnasium.utils.env_checker.check_env(PacFanEnv(...))` in a pytest test (`ml/tests/test_env.py`) to verify spaces, dtypes, seeding reproducibility, reset/step contracts, and `AsyncVectorEnv` parallel execution.
- [ ] Implement two zero-ML baselines for benchmarking and reward calibration:
  - `RandomLegalAgent`: uniformly samples from `legal_actions`.
  - `GreedyBFSAgent`: chases nearest dot/frightened ghost via BFS while avoiding dangerous ghosts within distance `K`.
- [ ] Log baseline mean/max score, dots eaten, survival ticks, and throughput (env steps/sec across 1, 4, 8 workers).

### Step 2.4 — Training script, run directory layout, and checkpoint evaluation (`ml/train_ppo.py`)
- [ ] Implement a training script using `AsyncVectorEnv` + `MaskablePPO` (from `sb3-contrib`, with a custom small CNN suited to `31×28` grids, since Atari NatureCNN expects `84×84` images).
- [ ] Adopt the structured run-directory layout from `machine.md` §4.2:
  ```text
  runs/<run-id>/
    run.json                 # hyperparameters, reward config, git commit, start time
    episodes.jsonl           # episode metrics: episode, step, ckpt, score, level, ticks
    checkpoints/             # saved model weights
    eval/                    # fixed-seed deterministic eval metrics (and .pfr replays in Phase 3)
    best/                    # highest-scoring checkpoint
  ```
- [ ] Run fixed-seed evaluation rollouts (`episodic_life=False`, deterministic policy, fixed list of seeds) at regular checkpoint intervals so learning curves are apples-to-apples comparable.
- [ ] Add a `ml/eval.py` script that loads a checkpoint, runs evaluation episodes, and prints summary statistics.

**Phase 2 Done When:**
- `pytest ml/tests` passes `check_env` and determinism tests across `AsyncVectorEnv`.
- `GreedyBFSAgent` reliably scores thousands of points and clears dots quickly.
- A short `MaskablePPO` run trains end-to-end, writes `runs/<run-id>/episodes.jsonl` + checkpoints, and beats `RandomLegalAgent` on fixed-seed evaluation.

---

## Phase 3: Live Visual Spectating & Deterministic Replays

**Goal:** Watch any trained agent play live in the real Raylib window with audio, AND record/replay both headless ML games and human games frame-for-frame using compact `.pfr` replay files.

### Step 3.1 — Replay file format & recording/playback primitives (`internal/sim/replay`)
- [ ] Define the `.pfr` ("PacFan Replay") format from `machine.md` §4.1:
  - Magic header `"PFR1"` + format version.
  - Length-prefixed JSON header: `git_commit`, `pacman_mode`, `start_level`, `seed`, `timing_profile`, `real_art` (whether `game-real.png` was loaded), `recorded_at`, `run_id`, `checkpoint`, `episode`.
  - Run-length encoded input stream: varint `(action, ticks_held)` pairs recorded at the `system.InputProvider` boundary.
  - Periodic 64-bit `sim.HashWorld` checkpoints (e.g., every 600 ticks).
  - JSON footer: `final_score`, `level_reached`, `total_ticks`, `deaths`, `dots_eaten`, `end_reason`.
- [ ] Implement `RecordingInput` (wraps any `system.InputProvider` — works for both headless agents and human `RaylibInput`).
- [ ] Implement `ReplayInput` (implements `system.InputProvider` from a decoded `.pfr` stream).
- [ ] Wire optional replay recording into `sim.Env` and the NDJSON `reset`/`save_replay` (or `record_path` in `reset`) API so Phase 2 evaluation runs automatically save `runs/<run-id>/replays/eval/ckpt-XXXX/*.pfr` and `best/*.pfr`.

### Step 3.2 — Determinism & replay corpus tests
- [ ] **Round-trip test:** Play random/scripted games for thousands of ticks while recording, replay in a fresh world, and assert identical `HashWorld` on **every** tick for both Pac-Man and Ms. Pac-Man and both `TimingFast` and `TimingArcade`.
- [ ] **Headless-equals-windowed test:** Replay a headless-recorded `.pfr` through `game.Engine`'s scene pipeline (with a `NoopDrawer`) and verify zero hash mismatches.
- [ ] **Replay corpus test:** Check in 2–3 tiny `.pfr` fixtures (e.g., `testdata/replays/*.pfr`, covering dots, ghost eaten, fruit, death, level clear) so accidental rule regressions fail `go test ./...`.

### Step 3.3 — Windowed replay playback & human recording (`main.go` + `internal/game`)
- [ ] Extend `game.Options` so `game.Engine` accepts an optional injected `system.InputProvider`, `TimingProfile`, `Seed`, `StartLevel`, and `SkipHighScorePersistence` (for replays).
- [ ] Add CLI flags to `pacfan` (`main.go`):
  - `-record FILE`: wraps `RaylibInput` in `RecordingInput` and writes `.pfr` on exit (great for human baselines and imitation learning!).
  - `-replay FILE_OR_DIR`: loads one `.pfr` or iterates through all `.pfr` files in a directory in sorted order.
  - `-replay-speed N`: playback speed multiplier (`0.25` to `16.0`, default `1.0`).
- [ ] Implement replay viewer UX in the windowed loop:
  - Read header, warn if `git_commit` differs, configure world with recorded mode/seed/level/timing profile.
  - Add **cosmetic holds** for `TimingFast` replays (briefly pause rendering on death/round-clear without advancing simulation ticks so replays look natural while preserving exact tick hashes).
  - Verify each recorded state hash as playback passes it and surface any mismatch immediately.
  - Viewer controls (read directly from Raylib keyboard without polluting `ReplayInput`): `Space` pause, `Right Arrow` single-tick step while paused, `+`/`-` speed, `N`/`P` next/previous replay in directory.
  - Minimal HUD overlay showing `run_id`, `checkpoint`, `episode`, `tick`, and speed.

### Step 3.4 — Live visual spectating over local socket (`-api tcp://` and `unix://`)
- [ ] Extend `internal/api/server.go` to listen on `unix:///path/to.sock` or `tcp://127.0.0.1:PORT` (reject or warn on non-loopback binds by default for safety).
- [ ] Create `internal/api/remote_input.go`: thread-safe `RemoteInput` implementing `system.InputProvider`, bridging socket commands to the Raylib main-thread game loop (which must stay on the OS main thread on macOS).
- [ ] Support two live-spectating behaviors in `main.go` when `-api` is set without `-headless`:
  - **Lockstep visual mode** (default): main loop renders at 60.6 Hz (scaled by `-speed` if provided) and advances simulation frames in lockstep with incoming `step` commands from the connected Python policy (`ml/eval.py --connect tcp://127.0.0.1:7777`).
  - Human can still press `Space` (pause), `M` (mute), or close the window cleanly.

**Phase 3 Done When:**
- A human game recorded with `pacfan -record human.pfr` replays identically with `pacfan -replay human.pfr`.
- Headless evaluation replays from Phase 2 play back in the Raylib window with zero hash mismatches and working viewer controls.
- A trained Python agent can play live in the Raylib window via `pacfan -api tcp://127.0.0.1:7777 -pacman -zoom 3`.

---

## Phase 4: High-Throughput Batching & Native Scaling (Optional)

**Goal:** Only execute this phase if Phase 2 profiling shows NDJSON IPC / Python subprocess serialization is the training bottleneck (versus neural network forward/backward time).

### Step 4.1 — Profile first and set target bottlenecks
- [ ] Profile end-to-end training step time (Go simulation vs JSON encode/decode vs pipe I/O vs PyTorch inference/update) across 8–32 workers.
- [ ] Run `go test -cpuprofile` / `-memprofile` on `internal/sim` and eliminate per-tick heap allocations (e.g., cached `donburi.Query` instances in `internal/system/level.go` and `internal/sim`, zero-alloc observation writing into caller-owned `[]float32` buffers).

### Step 4.2 — Go-native `VecEnv` and observation encoders (`internal/sim`)
- [ ] Implement Go-native observation encoders (`ObserveGrid(dst []float32)` and `ObserveFeatures(dst []float32)`) directly inside `internal/sim` with zero per-step allocations.
- [ ] Implement `sim.VecEnv`: manages `N` independent `*sim.Env` worlds on a worker goroutine pool, takes `actions []Action`, steps all `N` worlds in parallel, auto-resets terminated/truncated envs, and writes directly into contiguous `obs`, `rewards`, `terminated`, `truncated`, and `legalMasks` buffers.
- [ ] Add `cmd/pacfan-train` (optional pure-Go CLI for ultra-fast Go baselines, MCTS/expectimax search, or evolutionary strategies).

### Step 4.3 — Choose the fast Python bridge (two options, pick based on profiling)
- **Option A — `c-shared` Go library (`machine.md` §3.4, fastest for Python):**
  - Build `libpacfan.so` / `libpacfan.dylib` via `go build -buildmode=c-shared ./cmd/pacfan-capi` exporting a minimal C ABI (`pf_vec_new`, `pf_vec_reset`, `pf_vec_step`, `pf_vec_save_replay`, `pf_vec_free`).
  - Wrap with `ctypes` / `cffi` in `ml/pacfan_vec_env.py` implementing `gymnasium.vector.VectorEnv`, writing observations directly into caller-owned NumPy arrays (one C call per batched step across all `N` environments).
- **Option B — Binary framed / batched `step_batch` over `stdio` (`scriptable.md` Phase 4, zero CGo):**
  - Add a length-prefixed binary mode (`-api stdio-bin`) and `step_batch` command stepping `N` worlds inside a single Go process over one pipe, avoiding CGo build complexity while eliminating JSON overhead.

### Step 4.4 — Optional visualization & analysis extras (`machine.md` §4.3)
- [ ] **Video export:** Render `.pfr` replays to an off-screen Raylib `RenderTexture` and pipe raw frames to `ffmpeg` to produce shareable MP4/GIF progression reels.
- [ ] **Side-by-side comparison:** Render two `.pfr` replays with the same seed (e.g., early vs. late checkpoint) side-by-side.
- [ ] **Heat maps:** Aggregate player position densities and death locations across a directory of `.pfr` replays onto a maze overlay.

**Phase 4 Done When:**
- `BenchmarkVecEnvStep` scales linearly across CPU cores with near-zero allocations per step.
- Batched Python stepping measurably reduces environment-step wall time compared to Phase 2 `AsyncVectorEnv` + NDJSON.

---

## Suggested Execution Order & Checkpoints

| Milestone | Key Deliverables | Verification Gate |
| --- | --- | --- |
| **1A** (Steps 1.1–1.3) | Raylib-free `internal/audio`, shared `assets.FS`, seedable `PRNG`, `internal/sheet` cache | `go list -deps` has zero `raylib` in core pkgs; `make check` green |
| **1B** (Steps 1.4–1.7) | `internal/sim.Env`, `internal/api` NDJSON server, `cmd/pacfan-headless`, benchmarks | Determinism tests + NDJSON round-trip + benchmark baseline green |
| **2A** (Steps 2.1–2.3) | `ml/pacfan_env.py`, grid & BFS encoders, `RandomLegalAgent` & `GreedyBFSAgent` | `check_env` & `AsyncVectorEnv` pytest green; baseline scores logged |
| **2B** (Step 2.4) | `ml/train_ppo.py`, `runs/<run-id>/` layout, fixed-seed eval checkpoints | Policy improves over random baseline on fixed-seed eval |
| **3A** (Steps 3.1–3.3) | `.pfr` replays, `RecordingInput`/`ReplayInput`, `-record`/`-replay` viewer + corpus tests | Round-trip, corpus, and headless-vs-windowed hash tests green |
| **3B** (Step 3.4) | `-api tcp://` & `unix://`, `RemoteInput` live visual spectating | Python agent plays live in Raylib window at 60.6 Hz |
| **4** (Steps 4.1–4.4) | Profile-guided zero-alloc `VecEnv`, `c-shared` or binary bridge, video/heatmap tools | Measurable steps/sec gain over Phase 2 without breaking Phase 1–3 tests |
