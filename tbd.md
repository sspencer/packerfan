# Pac-Man Dossier Fidelity Checklist (`tbd.md`)

Gap analysis of `pacfan`'s gameplay logic verified against [The Pac-Man Dossier](https://www.gamedeveloper.com/design/the-pac-man-dossier) by Jamey Pittman.

## Already Aligned with the Dossier

For context, these core arcade mechanics are already closely modeled:

- 16-frame speed bitmasks (`Speed20`–`Speed105`) and Level Specifications Table A.1 (`internal/config/level.go`)
- Scatter/Chase wave schedule for Levels 1, 2–4, and 5+ (`internal/config/level.go`)
- Classic Blinky, Pinky (`Up` overflow `(-4, -4)`), Inky (`Up` overflow `(-2, -2)` + doubled Blinky vector), and Clyde (`>= 8` tiles) chase targeting (`internal/system/ghost_ai.go`)
- Intersection tie-breaking priority (`Up → Left → Down → Right`) and no-reversal rule outside mode changes (`internal/config/direction.go`, `internal/system/ghost_ai.go`)
- Reproducible 8 KB ROM-indexed PRNG with clockwise search for frightened ghosts (`internal/component/prng.go`, `internal/system/ghost_ai.go`)
- Pac-Man dot-eating (`1` frame) and energizer-eating (`3` frames) movement pauses plus diagonal corner-snapping (`internal/system/events.go`, `internal/system/movement.go`)
- Escalating ghost-eating streak scoring (`200`, `400`, `800`, `1600`) and intermission cadence after Levels 2, 5, 9, 13, 17 (`internal/config/constants.go`, `internal/config/level.go`)

---

## Effort Legend

- **`S` (Small)**: Localized fix or small feature in 1–2 files
- **`M` (Medium)**: Multi-file state/system change or new gameplay mechanic
- **`L` (Large)**: Major new subsystem (e.g., audio engine or full attract/2P mode)

---

## Fixes & Improvements

| Done | Category | Effort | Issue & Dossier Gap | Code Reference |
| :---: | --- | :---: | --- | --- |
| [x] | Ghost AI & Waves | S | **Pause Scatter/Chase wave timer during Frightened mode**: `FrameCount` keeps advancing while `IsFrightMode` is active, letting waves expire or transition while ghosts are blue. Per the Dossier, the wave timer pauses during Frightened mode and resumes afterward. | `internal/system/game_state.go` |
| [x] | Ghost AI & Waves | S | **Reverse ghosts on energizers during no-fright levels (17, 19+)**: `PowerUpEaten` skips reversing ghosts when `FrightenedFrames <= 0`. On those levels ghosts should not turn blue, but active (`Scatter`/`Chase`) ghosts must still reverse direction. | `internal/system/events.go` |
| [x] | Ghost AI & Waves | S | **Cruise Elroy Scatter override & post-death suspension**: Once Blinky reaches Elroy 1/2 he should keep targeting Pac-Man even during `Scatter` waves; after Pac-Man loses a life, Elroy must remain suspended until Clyde exits the ghost house. | `internal/system/ghost.go`, `internal/system/ghost_ai.go`, `internal/component/game.go` |
| [x] | Ghost AI & Waves | S | **Restrict red-zone `IsGhostWall` to upward turns only**: `isValidGhostTile` blocks entering `(12,10)`, `(15,10)`, `(12,22)`, and `(15,22)` from *any* direction, preventing ghosts from moving downward through those shafts. It should only block choosing `Up` at the four red-zone intersections. | `internal/system/ghost_ai.go`, `internal/component/maze.go` |
| [x] | Ghost AI & Waves | S | **Exact Scatter corner target coordinates**: Blinky targets `(28, -3)` instead of `(25, -3)` and Inky/Clyde target row `31` instead of `32` (in 0-based 28×31 maze coordinates). | `internal/archetype/ghost.go` |
| [x] | Ghost AI & Waves | S | **Use center-aligned player tile for ghost targeting**: Chase AI uses `playerTransform.Pos.ToTile()` (top-left truncation) while dot eating uses `(Pos + Half) / Tile`. Aligning ghost targeting to center-tile coordinates matches the Dossier's tile ownership model. | `internal/system/ghost_ai.go` |
| [x] | Ghost AI & Waves | M | **One-tile-ahead intersection decision lookahead**: Ghosts currently evaluate turns only when centered on the intersection tile itself; original arcade ghosts compute their next turn one tile ahead upon entering the preceding tile. | `internal/system/ghost.go`, `internal/system/ghost_ai.go` |
| [x] | Ghost House | M | **Personal dot counters, post-death global counter, and inactivity timer**: Replace the `DotsEaten`/`FrameCount` heuristic in `shouldLeaveHouse` with (1) per-ghost personal counters in strict Pinky → Inky → Clyde order (Level 1: `0/30/60`, Level 2: `0/0/50`, Level 3+: `0/0/0`), (2) the post-death global counter releasing Pinky/Inky/Clyde at `7/17/32` dots, and (3) a resetting no-dot timeout (`4s` Levels 1–4, `3s` Level 5+). | `internal/system/ghost_ai.go`, `internal/component/game.go`, `internal/system/events.go` |
| [x] | Ghost House | M | **Frightened state for ghosts inside the house**: Eating an energizer should turn `InHouse`/`LeavingHouse` ghosts blue (and flash), and ghosts exiting before Frightened mode ends (unless already eaten during that energizer) should enter the maze `Frightened` instead of `Scatter`/`Chase`. | `internal/system/events.go`, `internal/system/ghost.go`, `internal/render/renderer.go` |
| [x] | Ghost House | S | **Exit ghost house heading `Left` by default**: `updateGhostLeavingHouse` picks `Left`/`Right` randomly via PRNG; in original Pac-Man, ghosts always exit heading `Left` (unless reversed by a mode change while inside). | `internal/system/ghost.go` |
| [x] | Ghost House | S | **Increase eaten ghost eyes speed (~200% / 2 px per frame)**: `ghostSpeed` currently uses `Speed80` (1 px/frame) for `Eaten` and `EnteringHouse`, which is slower than high-level normal ghosts; arcade eyes race back to the pen at roughly double normal speed. | `internal/system/ghost_ai.go` |
| [x] | Ghost House | S | **Keep previously eaten ghost eyes moving during ghost-eaten freeze**: During the score popup freeze after eating a ghost, Pac-Man and active/frightened ghosts halt, but any other already-eaten eyes continue racing back to the ghost house. | `internal/scene/ghost_eaten.go`, `internal/system/ghost.go` |
| [x] | Movement & Collision | S | **Freeze Pac-Man mouth animation when stopped against a wall**: `updatePlayerAnimation` currently runs every frame regardless of whether `canMove` is true; in the original game Pac-Man stops chomping when blocked. | `internal/system/player.go` |
| [x] | Movement & Collision | S | **Increment `SpeedIndex` on `MoveActor` early center return**: Returning early when `move > 1 && IsCentered(transform.Pos)` skips advancing `motion.SpeedIndex`, replaying the same speed-mask bit on the next frame. | `internal/system/movement.go` |
| [x] | Movement & Collision | S | **Match ~1-second ghost-eaten freeze duration**: `GhostEatenPause` is `30` frames (~0.5 s) instead of the arcade's ~`60` frames (~1.0 s). | `internal/config/constants.go` |
| [x] | Movement & Collision | S | **Same-tile collision parity (optional)**: `CollisionSystem` uses a 6×6 inset AABB instead of same-tile occupancy (which prevents the arcade pass-through quirk but differs slightly at tile boundaries). Consider matching tile-based collision or making it configurable. | `internal/system/collision.go` |
| [x] | Movement & Collision | M | **Fixed-timestep `60.606061 Hz` arcade clock**: Replace `rl.SetTargetFPS(61)` with a fixed-timestep accumulator so simulation speed stays exact regardless of monitor refresh rate. | `main.go` |
| [x] | Bonus & Scoring | M | **Bonus fruit spawning, timer, collection, and score popup**: Spawn the level's `BonusSymbol` below the ghost house at `70` and `170` dots eaten for ~9–10 seconds, award `BonusPoints` on pickup, and briefly display the fruit score sprite in the maze. | `internal/config/level.go`, `internal/component/game.go`, `internal/system/`, `internal/render/renderer.go` |
| [x] | Bonus & Scoring | S | **Extra life (1UP) at 10,000 points**: Award one bonus life the first time `Score` reaches `10,000` in a game session. | `internal/component/game.go`, `internal/system/events.go` |
| [x] | HUD & Visuals | S | **Fix frightened ghost warning flash count**: `drawGhost` begins flashing at `FrightFlashes * 14` frames instead of `2 * FrightFlashes * 14`, producing only 3 white flashes on 5-flash levels (and 2 on 3-flash levels). | `internal/render/renderer.go` |
| [x] | HUD & Visuals | S | **Blink energizers and `1UP` HUD label**: Power pellets should flash during active play (solid during `READY!`), and the `"1UP"` label should blink on/off. | `internal/render/renderer.go` |
| [x] | HUD & Visuals | S | **Bottom-right level/fruit history HUD**: Display up to the 7 most recent level bonus symbols from right to left along the bottom-right HUD. | `internal/render/renderer.go` |
| [x] | HUD & Visuals | M | **Two-stage `"PLAYER ONE"` / `"READY!"` game-start sequence**: On a new game, display `"PLAYER ONE"` + `"READY!"` with actors hidden for the first ~2 s, then spawn actors and hold `"READY!"` for ~2 s (keep single `"READY!"` on respawns and subsequent levels). | `internal/scene/ready.go`, `internal/render/renderer.go` |
| [x] | HUD & Visuals | L | **Title / attract mode, 2-player support, and high-score persistence**: Add the arcade attract screen (ghost cast & scoring legend, demo play), optional 1P/2P alternating play, and saving `HighScore` across runs. | `internal/scene/`, `internal/render/` |
| [x] | Audio | L | **Arcade sound effects and dynamic siren**: Wire Raylib audio for the start jingle, alternating waka-waka dot chomp, dot-count-escalating siren, frightened hum, retreating eyes hum, eat ghost/fruit, extra life, death, and intermission music. | `internal/render/` (or new `internal/audio/`), `assets/` |

---

## Remaining Polish, Subtle Bugs & Ms. Pac-Man Fidelity

The core Pac-Man Dossier checklist above is complete (25/25). The items below capture subtle bugs, arcade UX polish, Ms. Pac-Man-specific fidelity gaps, and documentation sync found in a full post-completion audit:

| Done | Category | Effort | Issue & Details | Code Reference |
| :---: | --- | :---: | --- | --- |
| [x] | Audio & Scenes | S | **Don't replay start jingle on 2P turn switches**: `PacmanDyingScene` uses `NewGameStartReadyScene` whenever alternating players, replaying the 4-second `SoundStart` jingle and hiding actors for 2s on every turn swap. Only the initial game start should play `SoundStart`; turn switches should show `"PLAYER ONE"`/`"PLAYER TWO"` + `"READY!"` without the start jingle. | `internal/scene/pacman_dying.go`, `internal/scene/ready.go` |
| [x] | Events & Audio | S | **Flush follow-up `ExtraLifeAwarded` before early return on `GhostEaten`**: When eating a ghost pushes score past `10,000`, `Pipeline.Update` returns on `GhostEatenPause > 0` before a second `ProcessAllEvents`, delaying `SoundExtraLife` until after the 1-second freeze ends. | `internal/system/pipeline.go` |
| [x] | HUD & UX | S | **Visual `"PAUSED"` indicator & audio pause/resume**: Pressing `Space` silences background loops but leaves no visual pause cue on screen, and active one-shot jingles (`SoundStart`, `SoundDeath`, intermissions) continue playing while paused instead of pausing/resuming. | `internal/render/renderer.go`, `internal/audio/raylib.go`, `internal/scene/` |
| [ ] | HUD & UX | S | **Mode-appropriate window title & in-game convenience keys**: `main.go` hardcodes window title `"Pacman"` even in Ms. Pac-Man mode; add `"Ms. Pac-Man"` / `"Pac-Man"` title plus an in-game mute toggle key (e.g., `M`). | `main.go`, `internal/render/input.go`, `internal/audio/` |
| [ ] | Audio | S | **Use `*_firstloop.wav` intro clips for sirens and eyes**: `assets/ms-pacman/` ships `siren0_firstloop.wav`–`siren3_firstloop.wav` and `eyes_firstloop.wav` for cleaner transitions into the repeating loop WAVs. | `internal/audio/audio.go`, `internal/audio/raylib.go` |
| [ ] | Ghost AI & Waves | S | **Ms. Pac-Man first-scatter random wandering**: In arcade Ms. Pac-Man, Blinky and Pinky wander pseudo-randomly during the first Scatter wave instead of heading to fixed corners, breaking memorized Pac-Man patterns. | `internal/system/ghost.go`, `internal/system/ghost_ai.go` |
| [ ] | Visuals | M | **Use authentic Act 2 & Act 3 intermission sprites from `assets/game.png`**: Wire the unused sprites already present in `assets/game.png`—Pac-Man Act 2 nail/snagged Blinky (`y=1088..1120`), Pac-Man Act 3 patched/naked Blinky (`y=1104..1136`), and Ms. Pac-Man Act 3 stork/bundle/Junior (`y=176..192`) plus `intermission1_bump.wav`. | `internal/scene/intermission.go`, `internal/audio/` |
| [ ] | Ms. Pac-Man | L | **Ms. Pac-Man bouncing tunnel-to-tunnel bonus fruit & point table**: In Ms. Pac-Man mode, spawn fruit at `64` and `176` dots using Ms. Pac-Man's values (`100/200/500/700/1000/2000/5000`, random on Level 8+), navigating corridors from tunnel to tunnel with `fruit_bounce.wav`. | `internal/config/level.go`, `internal/component/game.go`, `internal/system/`, `internal/render/renderer.go` |
| [ ] | Documentation | S | **Sync `docs/ecs.md` with completed features**: Update `docs/ecs.md` to cover `internal/audio`, `AttractScene`, 2-player sessions, high-score persistence, `FixedStepClock`, `GhostHouse`, `SameTileCollide`, bonus fruit, and 1-second `GhostEatenPause`. | `docs/ecs.md` |
