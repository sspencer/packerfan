# Ideas

Candidate features for pacfan, roughly grouped by theme. Each entry says what it is, why it's worth doing, and how it fits the current architecture. Effort is a rough guess: **S** (a day or two), **M** (about a week), **L** (multi-week).

## Top picks

If only a handful get built, these give the most for the effort and build on what already exists:

1. **Procedural maze generator** (L): the feature you suggested. It reuses the text-maze pipeline, and it also helps the ML side by giving it unseen test mazes.
2. **Maze editor** (M): falls out of the generator work, since both need a renderer that draws walls from cells.
3. **Live spectate / human-vs-AI ghost** (M): the replay viewer and API already do most of this.
4. **Ghost replays and personal-best racing** (S/M): `.pfr` already captures everything needed.
5. **Rule variants as config** (M): one place to toggle speeds, AI tweaks and scoring, usable by both the game and the ML env.

---

## 1. Mazes

### 1.1 Procedural maze generator (L)

Generate new, playable mazes from a seed.

**Why:** Replayability for players, and generalization for the ML side. Agents trained on five fixed mazes memorize routes; a held-out set of generated mazes is a much better test.

**How it fits:**
- Mazes already travel as text templates (`component.MazeTemplate`, `ParseMazeTemplateFromRows`, `NewMazeSetFromTemplates`). A generator just has to output 28x31 text rows. Ghost-house detection (`computeGhostHouseFromTemplate`) and tunnel handling already work on templates.
- The main gap is **rendering**. `Renderer.drawMaze` blits each maze's walls from the spritesheet. Generated mazes need a wall renderer that picks wall pieces from each cell's neighbours (the classic "autotile" approach). Draw that once into a render texture per level, so the frame cost stays the same.
- `maze_hash` already identifies a maze set, so replays of generated mazes can store the generator seed and version, or simply the text rows, in the header.

**Generator approach (one that is known to work for this genre):**
1. Work on half the board (14 columns) and mirror it, as arcade mazes are left/right symmetric.
2. Place fixed features first: ghost house and door, the player start, and 1–2 tunnel rows.
3. Fill the rest by packing irregular wall "blocks" (Tetris-like pieces) on a coarse grid. Corridors are the gaps between blocks, so every corridor is exactly one tile wide.
4. Validate, and retry with the next seed until every check passes:
   - every path tile can be reached from every other;
   - no dead ends (every path tile has at least 2 open neighbours), since dead ends are unfair against ghosts;
   - no 2x2 open areas;
   - a sensible number of intersections, dots (~220–250) and power-pellet spots near the corners.
5. Score candidates (corridor-length spread, loop count, distance from the start to the pellets) and keep the best of N.

**Extras:**
- `--maze-seed N` to pick a maze, and `--random-mazes` for a new maze every level.
- A daily-seed mode (see 3.4).
- Expose `maze_seed` in the API `reset` so training can sample mazes per episode (a curriculum: small/simple first).

### 1.2 Maze editor (M)

An in-game or standalone editor: paint walls, dots, pellets, tunnels and the ghost house on the 28x31 grid, with live validation (same checks as the generator) and save to text.

**How it fits:** Uses the same cell-based wall renderer as 1.1. Saved mazes are plain text files loaded with `--maze FILE` or `--maze-dir DIR`. Could start as "edit a text file, hot-reload in the game" before adding mouse painting.

### 1.3 Maze packs from text files (S)

Load a set of mazes, with optional colours and fruit lists, from a directory of text files instead of only from the spritesheet. Lets people share maze sets without touching PNGs. A small step that can come before 1.1, since it forces the text-to-render path to exist.

### 1.4 Larger / non-standard boards (L)

Wider mazes with a scrolling camera, or multiple connected screens. Much bigger change: `config.GameWidth/GameHeight` are compile-time constants used widely. Worth it only after 1.1 proves popular.

---

## 2. Gameplay modes and variants

### 2.1 Rule variants as a config file (M)

Collect the tunables that are currently constants (speeds per level, fright time, scatter/chase timings, dot limits, the Cruise Elroy thresholds, scoring, extra-life score) into a `rules` struct with named presets: `arcade` (default), `easy`, `hard`, `no-fright` and so on. Load with `--rules FILE`.

**Why:** Difficulty options for players; reward and dynamics variety for ML. Store the rules hash in the `.pfr` header, as `maze_hash` is, so replays stay verifiable.

### 2.2 Time-attack / score-attack mode (M)

A fixed time limit (for example 5 minutes) on a maze that changes as you clear sections, with speed rising as you chain ghosts and fruit. Separate high-score table. It plays very differently from the arcade game and is a good showcase for generated mazes.

### 2.3 Practice mode (S)

Start at any level (`StartGameAtLevel` already exists), with infinite lives, a slowdown toggle, and an optional overlay showing each ghost's target tile. Great for learning patterns, and easy to build with the existing debug data.

### 2.4 Ghost-target / AI overlay (S)

Toggle a debug view that draws each ghost's current target tile and planned direction, the scatter/chase phase timer, and the dot counters. Useful for players learning the AI, for debugging rule changes, and when watching ML replays.

### 2.5 Two-player simultaneous co-op or versus (M/L)

Both players in the maze at once: co-op (shared lives, split score) or versus (most dots wins). The ECS should handle a second player entity; ghost targeting needs a rule for which player each ghost chases (nearest, or alternating).

### 2.6 Play as a ghost (M)

A human controls one ghost (keyboard or second gamepad) while the AI, or a trained agent, controls the player. The API already separates game logic from input, so a ghost-control input source is a natural extension.

---

## 3. Replays, competition and sharing

### 3.1 Record human games (S)

`docs/machine.md` already plans `--record FILE`. Recording what a person plays, in the same `.pfr` format, means human games can be watched, verified with `pacfan-headless --verify`, and used as imitation-learning data.

### 3.2 Ghost racing against a replay (M)

Play while a translucent "ghost" of a previous run (your best, or an AI's) moves through the same seed. Because the game is deterministic, the replay can run alongside in a second `sim.Env` and only its player position needs drawing.

### 3.3 Verified high-score table (S/M)

`highscore.go` stores a single number today. Keep a top-10 table with initials, and attach the `.pfr` for each entry so a score can be re-verified headlessly. The game being deterministic makes cheat-resistant scores almost free.

### 3.4 Daily challenge (S, after 1.1)

Seed the maze and PRNG from the date, so everyone plays the same game that day. Share results as a short string (score, level, seed), or as the `.pfr` for proof.

### 3.5 Replay export to GIF/MP4 (M)

Render a `.pfr` offscreen at native resolution and write frames to an animated GIF (pure Go, `image/gif`) or pipe them to `ffmpeg`. Handy for sharing great runs or interesting agent behaviour. Could run headless using the pure-Go sheet and a software renderer, or with a hidden Raylib window.

### 3.6 Replay scrubbing and bookmarks (S)

The viewer already pauses and single-steps. Add seeking (rebuild the game from the start, or from periodic snapshots taken while playing) and auto-bookmarks on deaths, ghost eats and level clears, so a long agent game can be jumped through quickly.

---

## 4. AI and machine learning

### 4.1 Live spectate over TCP (M)

`docs/scriptable.md` sketches `--api tcp://...` for the windowed game. Watching an agent play live at 60 Hz (or sped up) is a better demo than replays, and lets you adjust things while it plays.

### 4.2 Ghost agents (M)

Let the API control ghosts as well as the player, either one ghost or all four. This enables multi-agent training (a learned player against learned ghosts) and research into "fair" ghost AI.

### 4.3 Built-in bot opponents (S/M)

Ship the GreedyBFS baseline, or a small trained policy exported to ONNX or a hand-written Go version, as an attract-mode demo player instead of the scripted one, and as the opponent in 3.2. Its behaviour has to stay deterministic so replays verify.

### 4.4 Generalization benchmark (S, after 1.1)

A fixed suite of N generated mazes with published seeds, plus a script that reports mean score on held-out mazes. Gives the ML side a clear success measure beyond "score on maze 1".

### 4.5 Vectorized in-process env (L)

Today each Python env is its own subprocess using NDJSON. A cgo/`c-shared` build or a batched `step_many` command could run many games per process and avoid most of the serialization cost. Do this only if profiling shows the pipe is the bottleneck.

---

## 5. Presentation and feel

### 5.1 CRT / scanline shader (S)

The `Canvas` already draws into an offscreen 224x288 texture, so a post-process shader pass (scanlines, slight curvature, bloom, colour bleed) is a single change in `Canvas.Present`. Add a `--crt` flag and a hotkey to toggle it.

### 5.2 Bezel art and info panel (S)

Option to fill unused fullscreen margins with bezel artwork or a live info panel (high scores, level, controls) instead of black bars. `FitViewport` already computes the margins.

### 5.3 Gamepad support (S)

Raylib has gamepad input built in. Map the D-pad and left stick to directions, and Start/A to start and pause. A large usability win for living-room or arcade-cabinet setups.

### 5.4 Rebindable controls and a settings menu (M)

An in-game options screen for volume, controls, rules preset, CRT on/off and display mode, saved to a config file alongside the high score. Settles the growing number of command-line flags.

### 5.5 Theme packs (M)

The game already swaps spritesheets with `--art`. Extend it to a theme directory: sprites plus palettes, sounds and maze colours, selectable at runtime. Pairs well with generated mazes, which can take any palette.

---

## 6. Accessibility

- **Game speed slider** (S): run the arcade clock at 50–100% speed for players who need more reaction time. The fixed-step clock already takes its tick rate as a parameter (`config.NewFixedStepClock(hz)`).
- **High-contrast / colour-blind palettes** (S): ghosts are told apart mostly by colour. Add optional shape markers or high-contrast colours.
- **Turn buffering assist** (S): keep a queued turn for longer before the intersection, which makes cornering easier without changing the rules much (make it optional, and off for verified scores).
- **Audio cues for ghost proximity** (M): stereo-panned or pitch-shifted cues as ghosts approach.

---

## 7. Engineering and tooling

- **Web build** (M/L): build to WebAssembly so the game runs in a browser. Raylib supports web targets, though raylib-go's support needs checking. The headless sim is pure Go and would compile to WASM easily for a browser-side replay viewer.
- **Determinism fuzzing in CI** (S): random action sequences, recorded and replayed, asserting hash equality, run on every PR. Guards the property that replays, verified scores and ML all depend on.
- **Profiling dashboard** (S): `make bench` that compares results with the previous commit (`benchstat`) to catch performance regressions in the simulation's hot path.
- **Plugin-style ghost AI** (M): an interface for ghost targeting so variant AIs (for 2.1 and 4.2) don't need edits scattered through `ghost_ai.go`.
