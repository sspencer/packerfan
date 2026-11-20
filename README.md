# Pac-Man (`pacfan`)

Dot-eating maze game with ghosts built in Go using [Raylib](https://github.com/gen2brain/raylib-go) and the [Donburi](https://github.com/yohamta/donburi) Entity Component System (ECS).

For copyright reasons, no artwork is included in the repository; spritesheet assets were originally sourced from [The Spriters Resource](https://www.spriters-resource.com/) (see [assets/README.md](assets/README.md)).

## Running the Game

```sh
go mod tidy
go run .
```

### CLI Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-zoom` | `3` | Integer display scale factor (clamped to `1..10`) |
| `-pacman` | `false` | Use classic Pac-Man maze & sprites instead of Ms. Pac-Man |

### Controls

- **Arrow Keys** (`Up`, `Down`, `Left`, `Right`) or **WASD** (`W`, `A`, `S`, `D`): Steer Pac-Man / Ms. Pac-Man
- **Space**: Pause / resume simulation

### Useful Make Targets

```sh
make build   # Build the ./pacfan binary
make run     # Build and launch ./pacfan
make test    # Run all unit and headless ECS simulation tests
make check   # Run go fmt, go vet, and go test ./...
make clean   # Remove build artifacts
```

---

## ECS Architecture Overview

The game has been rewritten around `github.com/yohamta/donburi` so that **zero mutable package-level global state** exists anywhere in the codebase. Every piece of runtime state—actors, maze tiles, score, timers, wave schedules, and even the arcade PRNG—lives inside a `donburi.World` instance.

Core design principles:

1. **Data vs. Behavior Separation**: Components in `internal/component` are pure data structs. All game rules live in focused systems in `internal/system`.
2. **World-Scoped State**: Global singletons (`GameState`, `Maze`, `PRNG`) are stored as singleton entities in `donburi.World`, and events use Donburi's world-scoped event bus (`donburi/features/events`). Multiple worlds can run concurrently (e.g., in tests) with complete isolation.
3. **Simulation Decoupled from Raylib**: Only `internal/render` (and wiring in `main.go` / `internal/game`) imports `raylib-go`. Input polling (`system.InputProvider`) and spritesheet pixel sampling (`isPixelLit`) are injected via interfaces/callbacks so the entire simulation runs headlessly in `go test`.
4. **Archetype-Friendly Entities**: All four ghosts share an identical component layout so Donburi stores them contiguously in a single archetype in canonical update/draw order (`Blinky`, `Inky`, `Pinky`, `Clyde`).

---

## Project Layout

```text
pacfan/
├── main.go                          # Entry point: CLI flags, //go:embed assets, Raylib window & loop
├── assets/                          # Embedded spritesheet, bitmap font, chroma-key shader, audio
└── internal/
    ├── config/                      # Shared constants, math types, enums, and arcade lookup tables
    │   ├── constants.go             # Tile/sprite dimensions, board size, scores, pauses, FPS
    │   ├── direction.go             # Direction enum, Opposite/IsOpposite/ClockwiseNext, ghost tie-break order
    │   ├── ghost_state.go           # GhostState enum (Scatter, Chase, Frightened, Eaten, house states)
    │   ├── vector.go                # Integer 2D Vector with tile/pixel conversions & squared distance
    │   ├── level.go                 # Arcade speed bitmasks, level spec table (1..255), wave phase table
    │   └── *_string.go              # Generated stringer helpers
    ├── component/                   # Pure ECS data components & singleton world resources
    │   ├── actor.go                 # ActorID, Actor, Transform, Spawn, Motion, Animation
    │   ├── player.go                # PlayerTag, PlayerInput
    │   ├── ghost.go                 # GhostTag, GhostAIKind, Ghost
    │   ├── game.go                  # GameState singleton (score, lives, timers, fright & wave state)
    │   ├── maze.go                  # Maze singleton (28x31 tile grid, pixel-mask loader, tunnels, ghost walls)
    │   ├── prng.go                  # PRNG singleton (simulated 8 KB arcade ROM + index)
    │   └── actorid_string.go        # Generated stringer helper
    ├── event/                       # Typed Donburi events (donburi/features/events)
    │   └── events.go                # DotEaten, PowerUpEaten, GhostEaten, PlayerEaten, GhostStateChanged, FrightModeEnded, LevelCompleted
    ├── archetype/                   # Entity factories assembling components into a donburi.World
    │   ├── world.go                 # CreateGameState, CreateMaze, CreatePRNG
    │   ├── player.go                # SpawnPlayer
    │   └── ghost.go                 # SpawnBlinky, SpawnInky, SpawnPinky, SpawnClyde, SpawnAllGhosts
    ├── system/                      # ECS simulation systems (zero Raylib dependencies)
    │   ├── pipeline.go              # Per-frame ordered system scheduler & event flushes
    │   ├── input.go                 # InputProvider interface & InputSystem (pause + directional input)
    │   ├── game_state.go            # Frame counter, Scatter/Chase wave schedule, fright timer, level clear
    │   ├── collision.go             # Inset AABB player-vs-ghost collision detection
    │   ├── movement.go              # 16-frame arcade speed bitmask stepper, corner-snapping, tunnel wrap
    │   ├── ghost_ai.go              # Classic Blinky/Pinky/Inky/Clyde targeting, house exit, PRNG turning, speeds
    │   ├── ghost.go                 # Ghost animation, house bounce/exit, state updates, movement
    │   ├── player.go                # Player animation, intersection turn buffering, wall checks, dot pickups
    │   ├── events.go                # Subscribes all typed event handlers onto a donburi.World
    │   └── system_test.go           # Headless unit & integration tests
    ├── render/                      # Raylib presentation & hardware input adapter
    │   ├── art.go                   # Embedded asset loader (textures, image, chroma-key fragment shader)
    │   ├── font.go                  # 8x8 bitmap arcade font renderer
    │   ├── input.go                 # RaylibInput adapter implementing system.InputProvider
    │   └── renderer.go              # ECS Render system (cameras, maze, HUD, lives, shader-keyed sprites)
    └── game/                        # Composition root tying World, Pipeline, Art, and Renderer together
        └── engine.go                # Engine lifecycle (NewEngine, Update, Draw, Unload)
```

---

## Entities & Components

### Singleton Resource Entities

| Entity | Component | Purpose |
| --- | --- | --- |
| Game State | `component.GameState` | Zoom, Pac-Man/Ms. Pac-Man mode, pause flag, score, high score, lives, level, frame counter, dot counts, active `LevelSpecTable`, frightened timer/flashes, global `GhostState` wave |
| Maze | `component.Maze` | `28x31` tile grid (`Empty`, `Wall`, `Door`, `Dot`, `Power`, `Tunnel`), level index mapping, tunnel rows, ghost-restricted tiles |
| PRNG | `component.PRNG` | Reproducible 8192-byte simulated arcade ROM and cursor used by frightened ghosts and ghost-house exit direction selection |

> **Design Note on the Maze Grid**: Edible dots and power pellets stay in the `28x31` `Maze` tile grid rather than spawning ~244 static entities. This preserves `O(1)` tile lookups for movement, wall/door collision, tunnel speed checks, dot eating, and rendering.

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
| `DotEaten` | `PlayerSystem` | Clears maze cell, adds 10 pts (+ updates high score), updates dot counters, adds 1-frame player `DotPause` |
| `PowerUpEaten` | `PlayerSystem` | Clears maze cell, adds 50 pts, resets ghost-eaten streak, adds 3-frame `DotPause`, activates frightened mode and flags active ghosts to reverse |
| `GhostEaten` | `CollisionSystem` | Marks the ghost entity `IsEaten = true` and transitions its state to `Eaten` |
| `PlayerEaten` | `CollisionSystem` | Signals player capture by an active (`Chase`/`Scatter`) ghost |
| `GhostStateChanged` | `GameStateSystem` | Updates global wave state (`Scatter`/`Chase`) and reverses all ghosts currently in `Scatter`/`Chase` |
| `FrightModeEnded` | `GameStateSystem` | Clears frightened flags/timers and returns all `Frightened` ghosts to the active wave state |
| `LevelCompleted` | `GameStateSystem` | Pauses gameplay when `DotsRemaining == 0` |

---

## Game Loop & Logic Flow

The engine runs at **61 FPS** (approximating the original arcade hardware clock of `60.606061 Hz`). Each frame executes `Engine.Update()` followed by `Engine.Draw()`:

```text
Engine.Update()  [system.Pipeline.Update]
  1. InputSystem.Sample() + ApplyPause()   ──▶ if Paused, return early
  2. GameStateSystem.Update()              ──▶ frame tick, wave schedule, fright countdown, level-clear check
  3. CollisionSystem.Update()              ──▶ inset AABB overlap check (player vs. ghosts)
  4. events.ProcessAllEvents()
  5. GhostSystem.Update()                  ──▶ animation, house bounce/exit, AI targeting, turning, movement, wrap
  6. events.ProcessAllEvents()
  7. InputSystem.ApplyPlayerInput()        ──▶ write directional key into PlayerInput component
  8. PlayerSystem.Update()                 ──▶ animation, turn logic, wall checks, corner-snap movement, dot/powerup pickup, wrap
  9. events.ProcessAllEvents()
```

### Key Simulation Mechanics

- **Bitmask Arcade Movement (`internal/system/movement.go`)**: Actor speed is encoded as a 32-bit arcade speed pattern (`Speed20` through `Speed105`). Bit 31 selects base pixels-per-frame (0 or 1), while bits 15..0 cycle per frame to add sub-pixel fractional movement. Pac-Man also applies corner-snapping toward the orthogonal tile centerline during turns.
- **Player Turning (`internal/system/player.go`)**: Opposite reversals happen immediately; perpendicular turns succeed within a 3-pixel intersection threshold when the target tile is neither `Wall` nor `Door`.
- **Ghost AI (`internal/system/ghost_ai.go`)**:
  - **Blinky** directly targets Pac-Man's position and speeds up at Cruise Elroy 1 and Elroy 2 dot thresholds.
  - **Pinky** targets 4 tiles ahead of Pac-Man (reproducing the classic arcade `Up` overflow quirk: `(-4, -4)`).
  - **Inky** takes a pivot 2 tiles ahead of Pac-Man (with the same `Up` quirk `(-2, -2)`) and doubles the vector from Blinky's tile to that pivot.
  - **Clyde** targets Pac-Man when at least 8 tiles away (`DistanceSq >= 64`), otherwise retreats to its bottom-left scatter corner.
  - **Frightened** ghosts pick directions using the world-scoped arcade PRNG (`component.PRNG`), searching clockwise while disallowing immediate reversal.
  - **House States**: Ghosts start in `InHouse` (except Blinky in `Scatter`), bounce vertically until their dot/time release threshold is met, ascend via `LeavingHouse` to `(108, 88)`, and join the active wave.

---

## Rendering Pipeline

`internal/render/renderer.go` queries the `donburi.World` and draws the exact arcade layout across five Raylib 2D camera passes:

1. **Maze Pass (`mazeCamera`)**: Draws the level's maze background from `assets/game.png` offset by `TopOffset` (3 tiles), then draws remaining `Dot` and `Power` cells from `component.Maze`.
2. **HUD Labels Pass (`textCamera`)**: Draws `"1UP"` and `"HIGH SCORE"` using the 8×8 bitmap font (`assets/text.png`).
3. **HUD Scores Pass (`textCamera`)**: Draws right-aligned `Score` and `HighScore`.
4. **Lives Pass (`textCamera`)**: Draws `Lives - 1` life icons below the maze.
5. **Sprites Pass (`spriteCamera` + `assets/chroma_key.fs`)**: Uses the black chroma-key fragment shader to draw the player followed by all ghosts (handling normal directional poses, blue/white frightened flashing, and eaten eyes).

During startup (`game.NewEngine`), `component.MazeData.LoadLevel` reads 8×8 pixel bitmasks directly from the loaded `assets/game.png` image via `art.IsPixelLit` to classify every tile as `Empty`, `Dot`, `Power`, `Door`, or `Wall`.

---

## Background

- [The Pac-Man Dossier](https://www.gamedeveloper.com/design/the-pac-man-dossier)
- [Donburi ECS for Go](https://github.com/yohamta/donburi)
