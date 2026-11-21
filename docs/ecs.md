# ECS & Scene Architecture Overview

The game is built around `github.com/yohamta/donburi` so that **zero mutable package-level global state** exists anywhere in the codebase. Every piece of runtime state—actors, maze tiles, score, 1P/2P sessions, bonus fruit, timers, wave schedules, and even the arcade PRNG—lives inside a `donburi.World` instance, while high-level game phases (attract screen, ready countdown, active play, freezes, death animation, level transitions, intermissions, and game over) are coordinated by a lightweight scene state machine in `internal/scene`.

Core design principles:

1. **Data vs. Behavior Separation**: Components in `internal/component` are plain data structs (with small reset/lookup helpers). Per-frame gameplay rules live in focused systems in `internal/system`, and phase-level transitions live in `internal/scene`.
2. **World-Scoped State**: Global singletons (`GameState`, `Maze`, `PRNG`) are stored as singleton entities in `donburi.World`, and gameplay events use Donburi's world-scoped event bus (`donburi/features/events`). Multiple worlds can run concurrently (e.g., in tests) with complete isolation.
3. **Simulation Decoupled from Raylib**: Only `main.go`, `internal/render`, and `internal/audio/raylib.go` import `raylib-go`. Input polling (`system.InputProvider`), scene rendering (`scene.Drawer`), audio playback (`audio.Player`), and spritesheet pixel sampling (`isPixelLit`) are injected via interfaces/callbacks so the ECS simulation, scene state machine, and audio logic run headlessly in `go test`.
4. **Archetype-Friendly Entities**: All four ghosts share an identical component layout so Donburi stores them contiguously in a single archetype in canonical update/draw order (`Blinky`, `Inky`, `Pinky`, `Clyde`).

---

## Project Layout

```text
pacfan/
├── main.go                          # Entry point: CLI flags, asset checks, Raylib window, fixed-timestep arcade loop
├── assets/                          # Chroma-key shader (tracked) + local spritesheet, bitmap font & WAVs (embedded at build time)
└── internal/
    ├── config/                      # Shared constants, math types, enums, clock, and arcade lookup tables
    │   ├── constants.go             # Tile/sprite sizes, board dimensions, scores, extra life, fruit & scene durations, FPS
    │   ├── clock.go                 # FixedStepClock accumulator pacing simulation at exact 60.606061 Hz
    │   ├── direction.go             # Direction enum, Opposite/IsOpposite/ClockwiseNext, ghost tie-break order
    │   ├── ghost_state.go           # GhostState enum (Scatter, Chase, Frightened, Eaten, EnteringHouse, InHouse, LeavingHouse)
    │   ├── vector.go                # Integer 2D Vector with ToTile/ToCenterTile/ToPixel conversions & squared distance
    │   ├── level.go                 # Speed bitmasks (incl. SpeedEaten), Table A.1, wave phases, Ms. Pac-Man bonuses, intermissions
    │   ├── *_string.go              # Generated stringer helpers (BonusSymbol, Direction, GhostState)
    │   └── config_test.go           # Unit tests for level specs, wave schedules, clock pacing, and intermission rules
    ├── component/                   # ECS data components & singleton world resources
    │   ├── actor.go                 # ActorID, Actor, Transform, Spawn, Motion, Animation, ResetToStart
    │   ├── player.go                # PlayerTag, PlayerInput, ResetPlayer
    │   ├── ghost.go                 # GhostTag, GhostAIKind, Ghost (incl. lookahead, DotCounter, FrightenedInHouse), ResetGhost
    │   ├── game.go                  # GameState singleton & PlayerSession (1P/2P boards, fruit state, wave/fright/house counters)
    │   ├── maze.go                  # Maze singleton (28x31 tile grid, pixel-mask loader, GhostHouse geometry, TunnelRows, ghost walls)
    │   ├── prng.go                  # PRNG singleton (deterministic 8 KB simulated arcade ROM + cursor)
    │   └── actorid_string.go        # Generated stringer helper
    ├── event/                       # Typed Donburi events (donburi/features/events)
    │   └── events.go                # DotEaten, PowerUpEaten, GhostEaten, PlayerEaten, GhostStateChanged, FrightModeEnded, LevelCompleted, FruitEaten, FruitBounced, ExtraLifeAwarded
    ├── archetype/                   # Entity factories assembling components into a donburi.World
    │   ├── world.go                 # CreateGameState, CreateMaze, CreatePRNG
    │   ├── player.go                # SpawnPlayer
    │   └── ghost.go                 # SpawnBlinky, SpawnInky, SpawnPinky, SpawnClyde, SpawnAllGhosts
    ├── audio/                       # Decoupled arcade audio interface, loop selector & Raylib WAV player
    │   ├── audio.go                 # Player/NoopPlayer, fs.Stat MissingAssets/HasAnyAssets checks, SirenLoopForDots, SelectBackgroundLoop
    │   ├── raylib.go                # RaylibPlayer (waka-waka pitch alternation, *_firstloop.wav intros, pause/resume, mute)
    │   └── audio_test.go            # Siren escalation, loop priority, and fs.Stat asset verification tests
    ├── system/                      # ECS simulation systems (zero Raylib dependencies)
    │   ├── pipeline.go              # Per-frame ordered system scheduler, event flushes, and UpdateEatenGhosts
    │   ├── input.go                 # InputProvider interface, InputState (arrows/WASD, 1/2, Enter, Space, M), & InputSystem
    │   ├── game_state.go            # Frame/wave counters (pausing wave timer in fright), fright & fruit timers, level-clear fallback
    │   ├── collision.go             # Center-aligned SameTileCollide (player vs. ghosts & bonus fruit) + AABB helper
    │   ├── movement.go              # 16-frame arcade speed bitmask stepper, corner-snapping, tunnel wrap
    │   ├── ghost_ai.go              # Classic targeting, Cruise Elroy, Ms. Pac-Man first-scatter PRNG, house dot counters/timeout, speeds
    │   ├── ghost.go                 # Ghost animation, lookahead turns, clamped house entry/exit/bounce, UpdateEatenGhosts
    │   ├── fruit.go                 # FruitSystem navigating Ms. Pac-Man's bouncing tunnel-to-tunnel bonus fruit
    │   ├── player.go                # Player animation (freezes on wall), intersection turn buffering, wall checks, pickups
    │   ├── events.go                # Subscribes all typed gameplay event handlers onto a donburi.World
    │   ├── audio.go                 # RegisterAudioEventHandlers bridging ECS events to audio.Player
    │   ├── level.go                 # StartNewGame, RestartGame, AdvanceToNextLevel, ConsumeLifeOrSwitchPlayer (1P/2P), ResetActors
    │   └── system_test.go           # Headless ECS unit & integration tests
    ├── scene/                       # High-level game phase state machine (zero Raylib dependencies)
    │   ├── manager.go               # Scene interface & Manager (Exit/Enter lifecycle and deferred transitions)
    │   ├── context.go               # Shared Context, Drawer interface, DrawOptions, AttractView, IntermissionView, audio sync helpers
    │   ├── attract.go               # AttractScene (cast roll-call, scoring legend, chase demo, 1P/2P selection)
    │   ├── ready.go                 # ReadyScene (single READY!, 2P turn banner without jingle, two-stage PLAYER ONE start intro)
    │   ├── playing.go               # PlayingScene (active Pipeline.Update loop, dynamic audio loops & phase transition routing)
    │   ├── ghost_eaten.go           # GhostEatenFreezeScene (61-frame freeze displaying score sprite while other eyes keep moving)
    │   ├── round_cleared.go         # RoundClearedScene (hold, white maze-wall flashes, level advance, intermission routing)
    │   ├── intermission.go          # IntermissionScene (authentic Pac-Man & Ms. Pac-Man Acts 1-3 sprites & bump SFX)
    │   ├── pacman_dying.go          # PacmanDyingScene (collision hold, 11-frame death anim + SFX, 1P/2P respawn or Game Over)
    │   ├── game_over.go             # GameOverScene ("GAME OVER" banner returning to AttractScene after 3s or key press after 1s)
    │   └── scene_test.go            # Headless scene lifecycle, 2P alternation, attract & audio transition tests
    ├── render/                      # Raylib presentation & hardware input adapter
    │   ├── art.go                   # CheckAssets + embedded asset loader (textures, image, chroma-key & inline white-flash shaders)
    │   ├── font.go                  # 8x8 uppercase bitmap arcade font renderer
    │   ├── input.go                 # RaylibInput adapter implementing system.InputProvider
    │   ├── renderer.go              # Renderer implementing scene.Drawer (world, HUD, blinking pellets/1UP/2UP, fruit, PAUSED, attract, intermissions)
    │   └── renderer_test.go         # Sprite coordinate, flash count, blinking, fruit & spritesheet pixel validation tests
    └── game/                        # Composition root tying World, Pipeline, Scene Manager, Art, Audio, HighScore, and Renderer together
        ├── engine.go                # Engine lifecycle (NewEngine, Update, Draw, Unload)
        ├── highscore.go             # HighScore load/save persistence under os.UserConfigDir()
        └── highscore_test.go        # High-score persistence round-trip & graceful fallback tests
```

---

## Entities & Components

### Singleton Resource Entities

| Entity | Component | Purpose |
| --- | --- | --- |
| Game State | `component.GameState` | Zoom, Pac-Man/Ms. Pac-Man mode, pause flag, 1P/2P session slots (`PlayerCount`, `ActivePlayer`, `PlayerScores`, `Players[2]`), score/high score/lives/`ExtraLifeAwarded`, level, `FrameCount`, `WaveFrames` (paused during fright), dot counts, `ElroySuspended`, `UseGlobalDotCounter`/`GlobalDotCounter`/`FramesSinceLastDot`, active `LevelSpecTable`, frightened timer/flashes, global `GhostState` wave, ghost-eaten streak & popup freeze state, bonus fruit state (`FruitActive`, `FruitMoving`, `FruitSymbol`, `FruitPoints`, `FruitPos`, `FruitScoreTimer`), and scene-transition flags (`LevelComplete`, `PlayerDying`, `GameOver`) |
| Maze | `component.Maze` | `28×31` tile grid (`Empty`, `Wall`, `Door`, `Dot`, `Power`, `Tunnel`), spritesheet maze index mapping (`4` for Pac-Man; `0..3` for Ms. Pac-Man across levels), `GhostHouse()` pixel geometry derived from parsed `Door` tiles, `TunnelRows()`, and classic Pac-Man upward-restricted ghost tiles (`IsGhostWall`) |
| PRNG | `component.PRNG` | Reproducible 8192-byte simulated arcade ROM (seeded with `42`, recurrence `Index = (Index*5 + 1) % 8192`) used for frightened turning, Ms. Pac-Man first-scatter wandering, and bonus fruit duration/routing; rewound via `Reset()` on level advance, respawn, and restart |

> **Design Note on the Maze Grid**: Edible dots and power pellets stay in the `28×31` `Maze` tile grid rather than spawning ~244 static entities. This preserves `O(1)` tile lookups for movement, wall/door collision, tunnel checks, dot eating, 2-player board snapshots (`CloneCells`), and rendering.

### Actor Entities

| Archetype | Components |
| --- | --- |
| **Player** | `PlayerTag`, `Actor`, `Transform`, `Spawn`, `Motion` (`CornerSnap: true`), `Animation`, `PlayerInput` |
| **Ghost** (Blinky, Inky, Pinky, Clyde) | `GhostTag`, `Actor`, `Transform`, `Spawn`, `Motion` (`CornerSnap: false`), `Animation`, `Ghost` |

---

## Event System

Gameplay transitions and sound effects are decoupled via typed events from `github.com/yohamta/donburi/features/events` (defined in `internal/event/events.go` and subscribed in `internal/system/events.go` and `internal/system/audio.go`):

| Event | Published By | Handled Effect |
| --- | --- | --- |
| `DotEaten` | `PlayerSystem` | Clears cell, `AddScore(10)` (updating `HighScore` + publishing `ExtraLifeAwarded` at `10,000`), updates dot counters & ghost-house personal/global counter, spawns bonus fruit at `70`/`170` (Pac-Man) or `64`/`176` (Ms. Pac-Man) dots, adds 1-frame `DotPause`, publishes `LevelCompleted` if `DotsRemaining == 0`, plays `SoundChomp` |
| `PowerUpEaten` | `PlayerSystem` | Clears cell, `AddScore(50)`, updates dot & house counters, resets `NumGhostsEaten = 0`, adds 3-frame `DotPause`, spawns fruit / `LevelCompleted` if applicable, activates fright (marking active ghosts `Frightened` + reversed and house ghosts `FrightenedInHouse`), or reverses active ghosts on no-fright levels (`17`, `19+`), plays `SoundChomp` |
| `GhostEaten` | `CollisionSystem` | Marks ghost `IsEaten = true`, sets `State = Eaten` (`Reverse = false`, `HasNextDir = false`), increments `NumGhostsEaten`, `AddScore(200/400/800/1600)`, populates 61-frame `GhostEatenPause` popup fields, plays `SoundEatGhost` |
| `PlayerEaten` | `CollisionSystem` | Marks player `IsEaten = true` and sets `GameState.PlayerDying = true` |
| `GhostStateChanged` | `GameStateSystem` | Updates `GameState.GhostState` (`Scatter`/`Chase`), switches & reverses `Scatter`/`Chase` ghosts, and flags `InHouse`/`LeavingHouse` ghosts to exit `Right` |
| `FrightModeEnded` | `GameStateSystem` | Clears fright flags/timers, clears `FrightenedInHouse`, and returns `Frightened` ghosts to `GameState.GhostState` |
| `LevelCompleted` | `DotEaten`/`PowerUpEaten`, `GameStateSystem` | Sets `GameState.LevelComplete = true` (idempotent), halting simulation so `PlayingScene` transitions to `RoundClearedScene` |
| `FruitEaten` | `CollisionSystem` | Despawns active fruit, calls `AddScore(FruitPoints)`, starts 2-second score popup, plays `SoundEatFruit` |
| `FruitBounced` | `FruitSystem` | Triggers `SoundFruitBounce` once per bounce cycle in Ms. Pac-Man mode |
| `ExtraLifeAwarded` | `AddScore` helper | Triggers `SoundExtraLife` immediately on the frame score crosses `10,000` |

---

## Scene Lifecycle & State Machine

`Engine.Update()` and `Engine.Draw()` delegate to `scene.Manager` (`internal/scene/manager.go`), which manages clean `Exit()` → `Enter()` transitions before and after each update:

```text
  AttractScene ──(1 / 2 / Enter / Space)──▶ ReadyScene (2-stage intro + SoundStart)
       ▲                                       ▲  │
       │ (3s timeout or                        │  └──(ready elapses)──▶ PlayingScene ◀──┐
       │  any key after 1s)    (1P/2P respawn) │                          │   │   │      │
  GameOverScene ◀──(no lives)── PacmanDyingScene                          │   │   └──▶ GhostEatenFreezeScene
                                      ▲                                   │   │           (61 frames)
                                (PlayerDying)                             │   └──(LevelComplete)──▶ RoundClearedScene
                                                                          │                            │         │
                                                                          │                   (intermission)  (otherwise)
                                                                          │                            ▼         │
                                                                          └────────────────── IntermissionScene ─┴─▶ ReadyScene
```

| Scene | Behavior & Timing | Transitions To |
| --- | --- | --- |
| **`AttractScene`** | Shows ghost cast roll-call, scoring legend, looping demo chase animation, and `1` / `2` player selection prompt. Stops all audio on entry; plays `SoundCredit` on game start. | `ReadyScene` (two-stage game-start intro) |
| **`ReadyScene`** | Three variants: `NewGameStartReadyScene` (Stage 1: `"PLAYER ONE"` + `"READY!"` with hidden actors + `SoundStart`; Stage 2: `"READY!"` with actors), `NewPlayerTurnReadyScene` (2P turn switch: `"PLAYER ONE"`/`"PLAYER TWO"` + `"READY!"`, actors visible, no start jingle), and `NewReadyScene` (respawn/next level: `"READY!"` only). | `PlayingScene` |
| **`PlayingScene`** | Runs `Pipeline.Update(world)`, syncs pause/mute audio, drives dynamic background loops (`LoopEyes` > `LoopFright` > `LoopSiren0..4`), and checks transition flags in priority order: `PlayerDying`, `LevelComplete`, `GhostEatenPause > 0`. | `PacmanDyingScene`, `RoundClearedScene`, or `GhostEatenFreezeScene` |
| **`GhostEatenFreezeScene`** | Freezes simulation for `61` frames (~1.0 s), displays bonus score sprite (`200`/`400`/`800`/`1600`), and calls `Pipeline.UpdateEatenGhosts` so previously eaten eyes keep returning to the house (playing `LoopEyes` if any). | `PlayingScene` |
| **`RoundClearedScene`** | Holds cleared maze for `61` frames (~1 s), hides ghosts and flashes maze walls white for `4` cycles, then calls `system.AdvanceToNextLevel`. | `IntermissionScene` (after levels `2`, `5`, `9`, `13`, `17`, and every 4 levels after `17` in Ms. Pac-Man) or `ReadyScene` |
| **`IntermissionScene`** | Plays authentic Acts 1–3 cutscenes (Pac-Man nail/snag & patched/sheetless Blinky; Ms. Pac-Man clapperboard, Act 1 meet + `SoundIntermission1Bump`, Act 2 chase, Act 3 stork/bundle/Junior) with act music. | `ReadyScene` |
| **`PacmanDyingScene`** | `61`-frame collision hold → plays `SoundDeath` and 11-frame non-looping death animation (`77` frames) → `30`-frame empty-maze beat. Calls `system.ConsumeLifeOrSwitchPlayer`. | `ReadyScene` (same player or 2P turn switch) or `GameOverScene` |
| **`GameOverScene`** | Shows red `"GAME OVER"` over the empty maze; returns to `AttractScene` after `183` frames (~3 s) or on any key press after `61` frames (~1 s). | `AttractScene` |

---

## Game Loop & Simulation Pipeline

`main.go` drives `Engine.Update()` through `config.FixedStepClock` (`internal/config/clock.go`), pacing simulation ticks at exact **60.606061 Hz** regardless of monitor refresh rate.

During active gameplay (`PlayingScene`), `system.Pipeline.Update(world)` runs in deterministic order:

```text
system.Pipeline.Update(world)
  1. InputSystem.Sample() + ApplyPause()   ──▶ if Paused, LevelComplete, PlayerDying, or GameOver, return early
  2. [Standalone headless fallback]        ──▶ if GhostEatenPause > 0, buffer input, UpdateEatenGhosts, decrement, return until 0
  3. GameStateSystem.Update()              ──▶ FrameCount, fruit timers, level-clear fallback, fright countdown OR WaveFrames + wave check
  4. CollisionSystem.Update()              ──▶ SameTileCollide (player vs. bonus fruit and player vs. ghosts)
  5. events.ProcessAllEvents() (×2)        ──▶ handle collisions + flush follow-up ExtraLifeAwarded before early return
  6. GhostSystem.Update()                  ──▶ animation, house counters/timeout, clamped house states, lookahead AI, movement, entrance crossing
  7. FruitSystem.Update()                  ──▶ Ms. Pac-Man bouncing tunnel-to-tunnel fruit movement & FruitBounced events
  8. events.ProcessAllEvents()
  9. InputSystem.ApplyPlayerInput()        ──▶ write directional key into PlayerInput component
 10. PlayerSystem.Update()                 ──▶ turn buffering, wall checks, corner-snap movement, animation (only when moving), pickups, wrap
 11. events.ProcessAllEvents() (×2)        ──▶ handle dot/powerup pickups + flush follow-up LevelCompleted / ExtraLifeAwarded
```

### Key Simulation Mechanics

- **Bitmask Arcade Movement (`internal/system/movement.go`)**: 32-bit speed patterns (`Speed20`–`Speed105`, plus `SpeedEaten = 0x8000FFFF` at 2 px/frame for eaten ghost eyes). Always advances `SpeedIndex` even on early tile-center stops; pauses during `DotPause`; applies diagonal corner-snapping for Pac-Man.
- **Same-Tile Collision (`internal/system/collision.go`)**: Uses center-aligned tile occupancy (`SameTileCollide`: `pPos.ToCenterTile() == gPos.ToCenterTile()`) for both ghost and bonus fruit collisions.
- **Ghost AI, Lookahead & House Logic (`internal/system/ghost_ai.go`, `internal/system/ghost.go`)**:
  - **One-Tile-Ahead Lookahead**: Ghosts execute any reversal or pre-planned `NextDir` upon reaching `PlannedTile`, then precompute the turn for the upcoming tile using center-aligned player coordinates (`ToCenterTile`) and `Up → Left → Down → Right` tie-breaking.
  - **Red Zones & Scatter**: `IsGhostWall` blocks only `Up` turns into `(12,10)`, `(15,10)`, `(12,22)`, `(15,22)` in classic Pac-Man mode. Exact Dossier scatter corners (`(25,-3)`, `(2,-3)`, `(27,32)`, `(0,32)`). In Ms. Pac-Man mode, Blinky and Pinky wander pseudo-randomly during the first Scatter wave.
  - **Cruise Elroy**: Blinky chases Pac-Man even during `Scatter` waves once Elroy 1/2 triggers, and suspends Elroy after a life loss until Clyde exits the house.
  - **House Release & Geometry**: Personal dot counters in strict Pinky → Inky → Clyde order (`0/30/60` Level 1, `0/0/50` Level 2, `0/0/0` Level 3+), post-death global counter (`7/17/32`), and resetting no-dot timeout (`4s` Levels 1–4, `3s` Level 5+). Geometry is derived from parsed `Door` tiles (`MazeData.GhostHouse()`), using segment-crossing entrance detection (`crossedHorizontalPoint`) and overshoot-clamped axis stepping (`stepAxisToward`). Ghosts exit heading `Left` by default (or `Right` if reversed while inside) and exit `Frightened` if flagged `FrightenedInHouse` while an energizer is active.
- **Bonus Fruit (`internal/system/events.go`, `internal/system/fruit.go`)**: Classic Pac-Man spawns stationary fruit below the ghost house at `70`/`170` dots for 9.3–10 s. Ms. Pac-Man spawns bouncing fruit at `64`/`176` dots using its `100/200/500/700/1000/2000/5000` table (random on Level 8+), traveling from entry tunnel past the center and out an exit tunnel.

---

## Audio & Rendering Pipeline

### Audio (`internal/audio/`)

`audio.MissingAssets` / `audio.HasAnyAssets` verify optional WAV files via `fs.Stat` before initializing Raylib's audio device; missing files warn on `stderr` and degrade silently (or disable via `--mute` / toggle in-game with `M`). `RaylibPlayer` alternates chomp pitch for waka-waka cadence, plays optional `*_firstloop.wav` intro clips before seamlessly cycling `siren0..4.wav` / `eyes.wav`, and pauses/resumes all active sounds cleanly with `Space`.

### Rendering (`internal/render/renderer.go`)

`Renderer` implements `scene.Drawer` with three entry points:

1. **`DrawWorld(w, opts)`** (five Raylib 2D camera passes):
   - **Maze & Banner Pass (`mazeCamera`)**: Empty maze background (with `Art.FlashShader` when flashing), dots and blinking power pellets (`EnergizerVisible`, solid in `READY!`/`GAME OVER`), and `"PLAYER ONE"`/`"PLAYER TWO"`, `"READY!"`, `"GAME OVER"`, and `"PAUSED"` overlays.
   - **HUD Labels & Scores Passes (`textCamera`)**: Blinking active `"1UP"` / `"2UP"` header, `"HIGH SCORE"`, and right-aligned scores.
   - **Bottom HUD Pass (`textCamera`)**: Remaining life icons (`Lives - 1`) on the left and up to 7 level bonus fruit icons right-to-left on the right.
   - **Sprites Pass (`spriteCamera` + `assets/chroma_key.fs`)**: Bonus fruit (with vertical bounce offset in Ms. Pac-Man) or fruit score popup, player (mouth animation or `PlayerDeathSprite`), ghosts (normal, blue/white warning flashes via `FrightenedGhostFlashWhite`, or eyes), and ghost bonus score popups.
2. **`DrawIntermission(w, view)`**: Shared HUD, centered act title/subtitle, `"PAUSED"` overlay when paused, and scripted cutscene sprites.
3. **`DrawAttract(w, view)`**: Dual `1UP`/`2UP` + `HIGH SCORE` HUD, staggered cast roll-call portraits & names, scoring legend, animated demo chase sprites, and highlighted 1P/2P prompt.
