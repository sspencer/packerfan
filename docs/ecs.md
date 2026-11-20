# ECS & Scene Architecture Overview

The game is built around `github.com/yohamta/donburi` so that **zero mutable package-level global state** exists anywhere in the codebase. Every piece of runtime state—actors, maze tiles, score, lives, timers, wave schedules, and even the arcade PRNG—lives inside a `donburi.World` instance, while high-level game phases (ready countdown, active play, freezes, death animation, level transitions, intermissions, and game over) are coordinated by a lightweight scene state machine in `internal/scene`.

Core design principles:

1. **Data vs. Behavior Separation**: Components in `internal/component` are plain data structs (with small reset/lookup helpers). Per-frame gameplay rules live in focused systems in `internal/system`, and phase-level transitions live in `internal/scene`.
2. **World-Scoped State**: Global singletons (`GameState`, `Maze`, `PRNG`) are stored as singleton entities in `donburi.World`, and gameplay events use Donburi's world-scoped event bus (`donburi/features/events`). Multiple worlds can run concurrently (e.g., in tests) with complete isolation.
3. **Simulation Decoupled from Raylib**: Only `main.go` and `internal/render` import `raylib-go`. Input polling (`system.InputProvider`), scene rendering (`scene.Drawer`), and spritesheet pixel sampling (`isPixelLit`) are injected via interfaces/callbacks so both the ECS simulation and scene state machine run headlessly in `go test`.
4. **Archetype-Friendly Entities**: All four ghosts share an identical component layout so Donburi stores them contiguously in a single archetype in canonical update/draw order (`Blinky`, `Inky`, `Pinky`, `Clyde`).

---

## Project Layout

```text
pacfan/
├── main.go                          # Entry point: CLI flags, //go:embed assets, Raylib window & loop
├── assets/                          # Chroma-key shader (tracked) + local spritesheet & bitmap font (embedded at build time)
└── internal/
    ├── config/                      # Shared constants, math types, enums, and arcade lookup tables
    │   ├── constants.go             # Tile/sprite sizes, board dimensions, scores, scene/pause durations, FPS, GhostScore
    │   ├── direction.go             # Direction enum, Opposite/IsOpposite/ClockwiseNext, ghost tie-break order
    │   ├── ghost_state.go           # GhostState enum (Scatter, Chase, Frightened, Eaten, EnteringHouse, InHouse, LeavingHouse)
    │   ├── vector.go                # Integer 2D Vector with tile/pixel conversions & squared distance
    │   ├── level.go                 # Arcade speed bitmasks, level spec table (1..255), wave phase table, IntermissionForLevel
    │   ├── *_string.go              # Generated stringer helpers (BonusSymbol, Direction, GhostState)
    │   └── config_test.go           # Unit tests for level specs, wave schedules, and intermission rules
    ├── component/                   # ECS data components & singleton world resources
    │   ├── actor.go                 # ActorID, Actor, Transform, Spawn, Motion, Animation, ResetToStart
    │   ├── player.go                # PlayerTag, PlayerInput, ResetPlayer
    │   ├── ghost.go                 # GhostTag, GhostAIKind, Ghost, ResetGhost
    │   ├── game.go                  # GameState singleton (score, lives, timers, fright, ghost-eaten popup, phase flags)
    │   ├── maze.go                  # Maze singleton (28x31 tile grid, pixel-mask loader, tunnels, ghost walls)
    │   ├── prng.go                  # PRNG singleton (deterministic 8 KB simulated arcade ROM + cursor)
    │   └── actorid_string.go        # Generated stringer helper
    ├── event/                       # Typed Donburi events (donburi/features/events)
    │   └── events.go                # DotEaten, PowerUpEaten, GhostEaten, PlayerEaten, GhostStateChanged, FrightModeEnded, LevelCompleted
    ├── archetype/                   # Entity factories assembling components into a donburi.World
    │   ├── world.go                 # CreateGameState, CreateMaze, CreatePRNG
    │   ├── player.go                # SpawnPlayer
    │   └── ghost.go                 # SpawnBlinky, SpawnInky, SpawnPinky, SpawnClyde, SpawnAllGhosts
    ├── system/                      # ECS simulation systems (zero Raylib dependencies)
    │   ├── pipeline.go              # Per-frame ordered system scheduler & event flushes
    │   ├── input.go                 # InputProvider interface, InputState, & InputSystem (pause, any-key, directional input)
    │   ├── game_state.go            # Frame counter, Scatter/Chase wave schedule, fright countdown, level-clear fallback
    │   ├── collision.go             # Inset 6x6 AABB player-vs-ghost collision detection
    │   ├── movement.go              # 16-frame arcade speed bitmask stepper, corner-snapping, tunnel wrap
    │   ├── ghost_ai.go              # Classic Blinky/Pinky/Inky/Clyde targeting, house exit thresholds, PRNG turning, speeds
    │   ├── ghost.go                 # Ghost animation, InHouse/LeavingHouse/EnteringHouse/Eaten states, AI, movement
    │   ├── player.go                # Player animation, intersection turn buffering, wall checks, dot/powerup pickups
    │   ├── events.go                # Subscribes all typed event handlers onto a donburi.World
    │   ├── level.go                 # Session/round lifecycle helpers (ResetActors, AdvanceToNextLevel, ConsumeLifeAndRespawn, RestartGame)
    │   └── system_test.go           # Headless ECS unit & integration tests
    ├── scene/                       # High-level game phase state machine (zero Raylib dependencies)
    │   ├── manager.go               # Scene interface & Manager (Exit/Enter lifecycle and deferred transitions)
    │   ├── context.go               # Shared Context, Drawer interface, DrawOptions, IntermissionView/IntermissionSprite
    │   ├── ready.go                 # ReadyScene ("READY!" pre-round countdown)
    │   ├── playing.go               # PlayingScene (active Pipeline.Update loop & phase transition routing)
    │   ├── ghost_eaten.go           # GhostEatenFreezeScene (30-frame freeze displaying ghost bonus score sprite)
    │   ├── round_cleared.go         # RoundClearedScene (hold, white maze-wall flashes, level advance, intermission routing)
    │   ├── intermission.go          # IntermissionScene (scripted Acts 1-3 cutscenes for Pac-Man and Ms. Pac-Man)
    │   ├── pacman_dying.go          # PacmanDyingScene (collision hold, 11-frame death animation, respawn or Game Over)
    │   ├── game_over.go             # GameOverScene ("GAME OVER" banner & any-key restart to Level 1)
    │   └── scene_test.go            # Headless scene lifecycle & transition tests
    ├── render/                      # Raylib presentation & hardware input adapter
    │   ├── art.go                   # Embedded asset loader (textures, image, chroma-key shader, inline white-flash shader)
    │   ├── font.go                  # 8x8 uppercase bitmap arcade font renderer
    │   ├── input.go                 # RaylibInput adapter implementing system.InputProvider (arrows, WASD, Space, any-key)
    │   ├── renderer.go              # Renderer implementing scene.Drawer (world cameras, HUD, lives, sprites, death, intermissions)
    │   └── renderer_test.go         # Sprite coordinate & spritesheet pixel validation tests
    └── game/                        # Composition root tying World, Pipeline, Scene Manager, Art, and Renderer together
        └── engine.go                # Engine lifecycle (NewEngine, Update, Draw, Unload)
```

---

## Entities & Components

### Singleton Resource Entities

| Entity | Component | Purpose |
| --- | --- | --- |
| Game State | `component.GameState` | Zoom, Pac-Man/Ms. Pac-Man mode, pause flag, score, high score, lives, level, frame counter, dot counts, active `LevelSpecTable`, frightened timer/flashes, global `GhostState` wave, ghost-eaten streak & popup freeze state (`NumGhostsEaten`, `GhostEatenPause`, `GhostEatenX/Y`, `GhostEatenEntity`, `GhostEatenScore`), and scene-transition flags (`LevelComplete`, `PlayerDying`, `GameOver`) |
| Maze | `component.Maze` | `28×31` tile grid (`Empty`, `Wall`, `Door`, `Dot`, `Power`, `Tunnel`), spritesheet maze index mapping (`4` for Pac-Man; `0..3` for Ms. Pac-Man across levels), tunnel rows, and classic Pac-Man upward-restricted ghost tiles (`IsGhostWall`) |
| PRNG | `component.PRNG` | Reproducible 8192-byte simulated arcade ROM (seeded with `42`, recurrence `Index = (Index*5 + 1) % 8192`) used by frightened ghosts and ghost-house exit direction selection; rewound via `Reset()` on level advance, respawn, and restart |

> **Design Note on the Maze Grid**: Edible dots and power pellets stay in the `28×31` `Maze` tile grid rather than spawning ~244 static entities. This preserves `O(1)` tile lookups for movement, wall/door collision, tunnel speed checks, dot eating, and rendering.

### Actor Entities

| Archetype | Components |
| --- | --- |
| **Player** | `PlayerTag`, `Actor`, `Transform`, `Spawn`, `Motion` (`CornerSnap: true`), `Animation`, `PlayerInput` |
| **Ghost** (Blinky, Inky, Pinky, Clyde) | `GhostTag`, `Actor`, `Transform`, `Spawn`, `Motion` (`CornerSnap: false`), `Animation`, `Ghost` |

---

## Event System

Gameplay transitions are decoupled via typed events from `github.com/yohamta/donburi/features/events` (defined in `internal/event/events.go` and subscribed in `internal/system/events.go`):

| Event | Published By | Handled Effect |
| --- | --- | --- |
| `DotEaten` | `PlayerSystem` | Clears maze cell, adds `10` pts (+ updates `HighScore`), updates `DotsEaten` / `DotsRemaining`, adds 1-frame player `DotPause`, and publishes `LevelCompleted` if `DotsRemaining == 0` |
| `PowerUpEaten` | `PlayerSystem` | Clears maze cell, adds `50` pts (+ updates `HighScore`), updates dot counters, resets `NumGhostsEaten = 0`, adds 3-frame `DotPause`, publishes `LevelCompleted` if `DotsRemaining == 0`, and (when `Spec.FrightenedFrames > 0`) activates frightened mode and turns/reverses ghosts in `Chase`, `Scatter`, or `Frightened` |
| `GhostEaten` | `CollisionSystem` | Marks the ghost `IsEaten = true`, transitions it to `Eaten` (`Reverse = false`), increments `NumGhostsEaten`, awards escalating points (`200`, `400`, `800`, `1600`, updating `HighScore`), and populates the 30-frame `GhostEatenPause` / popup coordinates, entity, and score |
| `PlayerEaten` | `CollisionSystem` | Marks the player `IsEaten = true` and sets `GameState.PlayerDying = true` |
| `GhostStateChanged` | `GameStateSystem` | Updates global `GameState.GhostState` (`Scatter`/`Chase`) and switches + flags reversal on all ghosts currently in `Scatter` or `Chase` |
| `FrightModeEnded` | `GameStateSystem` | Clears `IsFrightMode` / `FrightFrames` / `FrightFlashes` and returns all `Frightened` ghosts to `GameState.GhostState` |
| `LevelCompleted` | `DotEaten` / `PowerUpEaten` handlers, `GameStateSystem` (fallback) | Sets `GameState.LevelComplete = true` (idempotent), halting simulation so `PlayingScene` transitions to `RoundClearedScene` |

---

## Scene Lifecycle & State Machine

`Engine.Update()` and `Engine.Draw()` delegate directly to `scene.Manager` (`internal/scene/manager.go`), which manages clean `Exit()` → `Enter()` transitions before and after each scene update:

```text
                   ┌──────────────────────────────┐
                   ▼                              │
              ReadyScene ──(122 frames)──▶ PlayingScene ◀──┐
                ▲     ▲                     │   │   │      │
       (lives>0)│     │ (any key            │   │   └──(GhostEatenPause>0)──▶ GhostEatenFreezeScene
                │     │  after 1s)          │   │                               (30 frames)
        PacmanDyingScene                    │   │
           ▲        │                       │   └──(LevelComplete)──▶ RoundClearedScene
           │        └──(lives==0)──▶ GameOverScene                     │           │
  (PlayerDying)                                             (intermission act)   (otherwise)
                                                                       ▼           │
                                                               IntermissionScene ──┴──▶ ReadyScene
```

| Scene | Behavior & Timing | Transitions To |
| --- | --- | --- |
| **`ReadyScene`** | Displays maze, reset actors, and `"READY!"` for `122` frames (~2 s). Supports pause and pre-buffers directional input. | `PlayingScene` |
| **`PlayingScene`** | Runs `system.Pipeline.Update(world)` each frame, then checks `GameState` flags in priority order: `PlayerDying`, `LevelComplete`, `GhostEatenPause > 0`. | `PacmanDyingScene`, `RoundClearedScene`, or `GhostEatenFreezeScene` |
| **`GhostEatenFreezeScene`** | Freezes simulation for `30` frames (~0.5 s), hides the player and eaten ghost, and draws the bonus score sprite (`200`/`400`/`800`/`1600`). Supports pause and directional input buffering. | `PlayingScene` |
| **`RoundClearedScene`** | Holds the cleared maze for `61` frames (~1 s), hides ghosts and flashes maze walls white for `4` cycles (`14` frames on / `14` frames off each), then calls `system.AdvanceToNextLevel`. | `IntermissionScene` (after levels `2`, `5`, `9`, `13`, `17`, and every 4 levels after `17` in Ms. Pac-Man mode) or `ReadyScene` |
| **`IntermissionScene`** | Plays scripted Act 1 (`366` frames, ~6.0 s), Act 2 (`336` frames, ~5.5 s), or Act 3 (`336` frames, ~5.5 s) cutscenes for either Pac-Man or Ms. Pac-Man (with clapperboard slates `"THEY MEET"`, `"THE CHASE"`, `"JUNIOR"`). Supports pause. | `ReadyScene` |
| **`PacmanDyingScene`** | Three phases: `61`-frame collision hold (ghosts visible), `11 × 7 = 77`-frame non-looping death animation (ghosts hidden), and `30`-frame empty-maze beat. Calls `system.ConsumeLifeAndRespawn`. | `ReadyScene` (if lives remain; dots & score preserved) or `GameOverScene` (when lives reach `0`) |
| **`GameOverScene`** | Shows `"GAME OVER"` over the empty maze. After `61` frames (~1 s), any key press calls `system.RestartGame` (resets to Level 1, 3 lives, score `0`, preserving `HighScore`). | `ReadyScene` |

---

## Game Loop & Simulation Pipeline

The engine runs at **61 FPS** (approximating the original arcade hardware clock of `60.606061 Hz`). Each frame executes `Engine.Update()` (`scene.Manager.Update()`) followed by `Engine.Draw()` (`scene.Manager.Draw()`).

During active gameplay (`PlayingScene`), `system.Pipeline.Update(world)` runs in a deterministic fixed order:

```text
system.Pipeline.Update(world)
  1. InputSystem.Sample() + ApplyPause()   ──▶ if Paused, LevelComplete, PlayerDying, or GameOver, return early
  2. [Standalone headless fallback]        ──▶ if GhostEatenPause > 0, buffer input, decrement, return until 0
  3. GameStateSystem.Update()              ──▶ frame tick, level-clear fallback, wave schedule, fright countdown
  4. CollisionSystem.Update()              ──▶ inset 6×6 AABB overlap check (player vs. ghosts; breaks on first hit)
  5. events.ProcessAllEvents()             ──▶ if GhostEatenPause > 0, LevelComplete, or PlayerDying, return early
  6. GhostSystem.Update()                  ──▶ animation, house states, AI targeting, turning, movement, wrap, house entry
  7. events.ProcessAllEvents()
  8. InputSystem.ApplyPlayerInput()        ──▶ write directional key into PlayerInput component
  9. PlayerSystem.Update()                 ──▶ animation, turn logic, wall checks, corner-snap movement, dot/powerup pickup, wrap
 10. events.ProcessAllEvents() (×2)        ──▶ handle dot/powerup pickups + flush follow-up LevelCompleted immediately
```

### Key Simulation Mechanics

- **Bitmask Arcade Movement (`internal/system/movement.go`)**: Actor speed is encoded as a 32-bit arcade speed pattern (`Speed20` through `Speed105`). Bit 31 selects base pixels-per-frame (`0` or `1`), while bits `15..0` cycle over 16 frames to add sub-pixel fractional movement. When `DotPause > 0`, movement pauses for that frame; when moving 2 pixels in a frame, stepping stops early if the actor reaches an exact tile center (`IsCentered`) so turns are never skipped. Pac-Man also applies corner-snapping toward the orthogonal tile centerline during turns.
- **Collision Detection (`internal/system/collision.go`)**: Uses an inner `6×6` pixel hitbox (`5`-pixel inset per side of each `16×16` sprite) so adjacent-tile grazes do not trigger hits while preventing head-on pass-throughs. Colliding with a `Frightened` ghost emits `GhostEaten`; colliding with a `Chase` or `Scatter` ghost emits `PlayerEaten`.
- **Player Turning (`internal/system/player.go`)**: Opposite reversals happen immediately; perpendicular turns succeed within a 3-pixel intersection threshold (`offset <= 3` or `offset >= 5` inside an 8-pixel tile) when the adjacent target tile is neither `Wall` nor `Door`.
- **Ghost AI & State Lifecycle (`internal/system/ghost_ai.go`, `internal/system/ghost.go`)**:
  - **Tie-Breaking & Restrictions**: At tile centers, non-frightened ghosts pick the valid non-reverse direction minimizing squared tile distance to their target, breaking ties in classic arcade order `Up → Left → Down → Right`. In classic Pac-Man mode (`Scatter`/`Chase`), ghosts cannot enter upward-restricted tiles (`MazeData.IsGhostWall`).
  - **Blinky** directly targets Pac-Man's pixel position and accelerates at Cruise Elroy 1 and Elroy 2 remaining-dot thresholds.
  - **Pinky** targets 4 tiles ahead of Pac-Man's tile (reproducing the classic arcade `Up` overflow quirk: `(-4, -4)`).
  - **Inky** takes a pivot 2 tiles ahead of Pac-Man's tile (with the same `Up` quirk `(-2, -2)`) and doubles the vector from Blinky's tile to that pivot.
  - **Clyde** targets Pac-Man's tile when at least 8 tiles away (`DistanceSq >= 64`), otherwise retreats to its bottom-left scatter corner.
  - **Frightened** ghosts pick directions using the world-scoped arcade PRNG (`component.PRNG`), searching clockwise (`Up → Right → Down → Left`) while disallowing immediate reversal.
  - **House & Eaten Lifecycle**: Ghosts start in `InHouse` (except Blinky in `Scatter`), bounce vertically at tunnel speed until their level-dependent dot/time release threshold (`shouldLeaveHouse`) is met, ascend via `LeavingHouse` to `(108, 88)`, pick left or right via the PRNG, and join `GameState.GhostState`. When eaten (`Eaten`), eyes travel at `Speed80` to `(108, 88)`, transition to `EnteringHouse` to descend into their home slot (`EatenTarget`), then regenerate (`IsEaten = false`) and immediately begin `LeavingHouse`.

---

## Rendering Pipeline

`internal/render/renderer.go` implements `scene.Drawer` with two entry points:

### `DrawWorld(w, opts)`

Renders the game world across five Raylib 2D camera passes:

1. **Maze & Banner Pass (`mazeCamera`)**: Draws the level's empty maze background from `assets/game.png` offset by `TopOffset` (3 tiles)—applying the inline white-flash shader (`Art.FlashShader`) when `opts.FlashMaze` is set—then draws remaining `Dot` and `Power` cells from `component.Maze`. Optionally overlays `"READY!"` (yellow) or `"GAME OVER"` (red) using the chroma-key shader and bitmap font.
2. **HUD Labels Pass (`textCamera`)**: Draws `"1UP"` and `"HIGH SCORE"` using the 8×8 bitmap font (`assets/text.png`).
3. **HUD Scores Pass (`textCamera`)**: Draws right-aligned `Score` and `HighScore`.
4. **Lives Pass (`textCamera`)**: Draws `Lives - 1` life icons below the maze (when `Lives > 1`).
5. **Sprites Pass (`spriteCamera` + `assets/chroma_key.fs`)**: Uses the black chroma-key fragment shader to draw:
   - The player (unless `opts.HidePlayer` or a ghost score popup is active), selecting either directional mouth animation or `PlayerDeathSprite` (11-frame collapse+starburst for Pac-Man; 4-way spin for Ms. Pac-Man).
   - Ghosts (unless `opts.HideGhosts`, skipping `GhostEatenEntity` during score popups), rendering normal directional poses, blue/white frightened flashing, or directional eaten eyes.
   - The ghost-eaten bonus score sprite (`200`, `400`, `800`, `1600`) at `(GhostEatenX, GhostEatenY)` when `opts.ShowGhostScore` or `GhostEatenPause > 0`.

### `DrawIntermission(w, view)`

Draws the shared HUD (`drawHUD`), optional centered act `Title` / `Subtitle` text via `mazeCamera`, and the cutscene's `IntermissionView.Sprites` (supporting both 16×16 and 32×32 sprites) via `spriteCamera` with the chroma-key shader.

### Pixel-Mask Maze Loading

On startup (`game.NewEngine`) and whenever advancing levels or restarting (`system.AdvanceToNextLevel`, `system.RestartGame`), `component.MazeData.LoadLevel` samples 8×8 pixel bitmasks directly from the loaded `assets/game.png` image via `art.IsPixelLit` to classify every tile as `Empty`, `Dot`, `Power`, `Door`, or `Wall`, then marks tunnel tiles for the active maze layout.
