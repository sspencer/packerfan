# Headless Pacfan: Simulation Server, API, Replays, and the Python AI

This guide explains how the headless build of Packer Fan (`pacfan-headless`) works, how to drive it from your own code over its JSON API, how to record and replay `.pfr` replay files, and how to run the Python AI (a Gymnasium environment, two simple non-learning baseline agents, and a MaskablePPO trainer) that is built on the API.

- [1. Overview](#1-overview)
- [2. Building and running](#2-building-and-running)
- [3. How the headless simulation works](#3-how-the-headless-simulation-works)
- [4. The NDJSON API](#4-the-ndjson-api)
- [5. Writing your own client](#5-writing-your-own-client)
- [6. Replay files (`.pfr`)](#6-replay-files-pfr)
- [7. Watching and verifying replays](#7-watching-and-verifying-replays)
- [8. Running the Python AI](#8-running-the-python-ai)
- [9. Benchmarks](#9-benchmarks)
- [10. Troubleshooting](#10-troubleshooting)

---

## 1. Overview

The normal game (`pacfan`) runs at the arcade's 60.606 frames per second in a Raylib window, with audio and a scene state machine (attract screen, ready screen, intermissions, game over). That is the wrong shape for an AI agent, which needs to play thousands of games far faster than real time and get the same result every time it repeats a game.

`pacfan-headless` reuses the **same ECS gameplay systems** (`internal/system`) but leaves out everything that is not gameplay:

| Piece | Normal game (`pacfan`) | Headless (`pacfan-headless`) |
| --- | --- | --- |
| Gameplay systems (movement, ghost AI, collisions, fruit, levels) | yes | yes, same code |
| Raylib window / rendering | yes | no |
| Audio | yes | no |
| Scenes (attract, ready, intermissions, game over) | yes | no; play only |
| Input | keyboard | one action per tick from the API |
| Timing | wall clock, 60.606 FPS | as fast as the CPU allows |
| Interface | window | NDJSON over stdin/stdout |

The pieces involved are:

| Path | Role |
| --- | --- |
| [`cmd/pacfan-headless/main.go`](../cmd/pacfan-headless/main.go) | Command-line entry point: flags, `--verify`, `--dump-mazes`, and the NDJSON server. |
| [`internal/sim`](../internal/sim) | `sim.Env`: a deterministic, play-only simulation (`Reset`, `Step`, `TickOne`), the world hash, and `.pfr` recording, parsing and verification. |
| [`internal/api`](../internal/api) | The NDJSON protocol: request/response types, the `Session` that maps commands to `sim.Env` calls, and the read/dispatch/write loop. |
| [`internal/game/replay_player.go`](../internal/game/replay_player.go), [`replay_viewer.go`](../internal/game/replay_viewer.go) | Plays `.pfr` files back in the normal game window (`pacfan --replay`). |
| [`ml/`](../ml) | The Python Gymnasium environment, baseline agents, PPO training and evaluation scripts. |

`make check` includes a `check-headless-deps` step that fails if `internal/sim`, `internal/api` or `cmd/pacfan-headless` ever import Raylib, audio, scenes or rendering, so the headless binary stays pure Go and builds without a C toolchain or GPU.

---

## 2. Building and running

```sh
make build-headless          # builds ./pacfan-headless
# or
go build -o pacfan-headless ./cmd/pacfan-headless
```

With no flags it starts the API server: it reads one JSON request per line on **stdin** and writes one JSON response per line on **stdout**. It is meant to be started as a subprocess by a client program, but you can also type requests into it (or pipe them in) by hand:

```sh
printf '%s\n' \
  '{"cmd":"reset","seed":7}' \
  '{"cmd":"step","action":"left","mode":"tile"}' \
  '{"cmd":"close"}' | ./pacfan-headless
```

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-a`, `--art FILE` | embedded `game-real.png` if present, else `game.png` | Game spritesheet PNG. The maze layouts are **parsed from this image**, so it decides which mazes are played (see [Artwork and `maze_hash`](#artwork-and-maze_hash)). |
| `-m`, `--ms` | `false` | Default to Ms. Pac-Man instead of classic Pac-Man when a `reset` does not say. |
| `-l`, `--level N` | `1` | Default starting level (`1..255`) for `reset`. |
| `-s`, `--seed N` | `42` | Default PRNG seed for `reset`. |
| `--verify FILE.pfr` | | Replay a `.pfr` file headlessly, check every hash checkpoint, print a summary and exit (non-zero on mismatch). No server is started. |
| `--dump-mazes` | | Print the art file name, its SHA-256, the `maze_hash`, and the five parsed text mazes, then exit. |

Values passed in a `reset` request always win over these defaults.

---

## 3. How the headless simulation works

### Ticks

One **tick** is one arcade frame (1/60.606 s of game time): one run of the ECS system pipeline. A tick takes microseconds, so the simulation runs thousands of times faster than real time. Every `step` request runs one or more ticks.

### Actions

There are five actions. The numbers match `config.Direction` and are used everywhere: in the API, in `.pfr` files, and in the Python environment.

| Number | Name | Meaning |
| --- | --- | --- |
| `0` | `right` | Steer right |
| `1` | `left` | Steer left |
| `2` | `up` | Steer up |
| `3` | `down` | Steer down |
| `4` | `noop` | No input this tick |

An action behaves like holding a joystick direction for the length of the step. Like the arcade, a direction that cannot be taken yet is remembered and taken at the next opening, and with `noop` Pac-Man keeps moving in its current direction until it hits a wall.

### Play-only rules

There are no scenes, so the moments the normal game handles with scenes are handled directly in `sim.Env`:

1. **Death**: a life is used up straight away (no death animation). If no lives are left the game is over and the episode ends (`terminated: true`).
2. **Level cleared**: the next level starts straight away (no intermission), unless `end_on_level_clear` is set, in which case the episode ends.
3. **Ghost eaten pause**: the short freeze after eating a ghost is always run inside the same `step`, so the agent is never asked to choose an action while the game is frozen.

The game is always one player.

### Step modes

A `step` applies one action for a number of ticks chosen by its `mode`:

| Mode | Stops when | Notes |
| --- | --- | --- |
| `frames` (default) | `frames` ticks have run (default 1), not counting ghost-eaten freeze ticks | Finest control; most requests per game. |
| `tile` | Pac-Man is centered on a new tile, or is stopped by a wall | At most 64 ticks. A good default for learning. |
| `intersection` | Pac-Man is centered on a decision tile (one with a side exit, or a wall straight ahead), or is stopped by a wall | At most 320 ticks. Fewest decisions per game; corridors are skipped. |

Every mode also stops early on a death, a level clear, the end of the episode, or the `max_ticks` limit. The response's `ticks` field says how many ticks actually ran.

### Determinism and seeds

Given the same mazes, the same `reset` options (mode, level, seed, episode options) and the same sequence of actions, the simulation produces exactly the same game every time. The only randomness (for example frightened ghosts' choice of turns) comes from a seeded PRNG stored in the ECS world. This is what makes `.pfr` replays possible: a replay stores only the starting options and the per-tick actions, plus hashes to prove the playback matches.

`sim.HashWorld` computes a 64-bit FNV-1a hash over all gameplay state (game state, maze grid, PRNG position, player and ghosts). Cosmetic values such as zoom and high score are excluded.

### Episodes and options

A `reset` starts an episode. Options that change what an episode is:

| Option | Default | Effect |
| --- | --- | --- |
| `episodic_life` | `false` | Each lost life ends the episode (`terminated: true`). The next `reset` (with the same mode and level and no `record`) **continues the same game** on the next life, keeping score, dots and level. After the last life, `reset` starts a new game. Common in RL training because deaths give a clear signal. |
| `end_on_level_clear` | `false` | Clearing a level ends the episode instead of starting the next level. |
| `max_ticks` | `0` (no limit) | The episode is cut off (`truncated: true`) once this many ticks have run since the game started. |

### Artwork and `maze_hash`

The maze layouts (walls, dots, power pellets, tunnels and the ghost house door) are not hard-coded: they are read from the pixels of the game spritesheet. The placeholder `assets/game.png` in the repository has different maze layouts from the original artwork (`assets/game-real.png`, if you have it locally).

So the server reports two identities:

- `art.sha256`: the SHA-256 of the PNG file.
- `maze_hash`: a hash of the five parsed maze layouts. **This is what matters for gameplay.** Two PNGs with different pictures but identical mazes have the same `maze_hash`.

Replays store both. A replay can only be played back with art whose `maze_hash` matches; an `art.sha256` difference alone just prints a warning. Use `./pacfan-headless --dump-mazes` (optionally with `--art FILE`) to see which mazes and hashes a PNG gives.

There are five mazes (indexes `0..4`):

| Index | Used for |
| --- | --- |
| `0` | Ms. Pac-Man levels 1-2 |
| `1` | Ms. Pac-Man levels 3-5 |
| `2` | Ms. Pac-Man levels 6-9, then alternating with maze 3 every four levels from level 14 |
| `3` | Ms. Pac-Man levels 10-13, then alternating with maze 2 |
| `4` | Classic Pac-Man (every level) |

---

## 4. The NDJSON API

### Transport

- **NDJSON**: every request is one JSON object on one line, and every request gets exactly one JSON response line, in order.
- Requests go to the server's **stdin**; responses come from its **stdout**. Diagnostics go to stderr.
- Every response has `"ok"`. On failure it is `false` with an `"error"` string, and the server keeps running:

  ```json
  {"ok":false,"error":"unknown command \"bogus\""}
  ```

- Blank lines are ignored. A request line may be up to 1 MiB.
- The server exits after `close`, or when stdin reaches end of file. Either way any replay being recorded is finished and closed.
- One server process runs one game. To play several games at once, start several processes (this is how the Python vector environments work).

### Commands at a glance

| Command | Purpose |
| --- | --- |
| `hello` | Get static information: protocol version, art identity, `maze_hash`, the five maze layouts, action and ghost names. |
| `reset` | Start (or continue) an episode; optionally start recording a `.pfr` replay. Must be sent before `step` or `observe`. |
| `step` | Apply an action for one step and get the reward, events and new state. |
| `observe` | Get the current state without advancing, optionally with the live maze grid. |
| `close` | Finish any recording and exit. |

### `hello`

```json
{"cmd":"hello"}
```

Response (maze rows shortened):

```json
{
  "ok": true,
  "protocol": 1,
  "art": {"file": "game.png", "sha256": "e608ed9e..."},
  "maze_hash": "e624e288...",
  "width": 28,
  "height": 31,
  "mazes": [
    {"index": 0, "dots": 224, "rows": ["############################", "#......##..........##......#", "..."]},
    "... 5 mazes in total ..."
  ],
  "actions": ["right", "left", "up", "down", "noop"],
  "ghosts": ["blinky", "pinky", "inky", "clyde"]
}
```

- `width`/`height`: the maze grid size in tiles (28 x 31). A tile is 8 x 8 pixels.
- `mazes[i].rows`: 31 strings of 28 characters each, as the maze looks at the start of a level:

  | Character | Tile |
  | --- | --- |
  | `#` | Wall |
  | `-` | Ghost house door (a wall for Pac-Man) |
  | `.` | Dot (10 points) |
  | `O` | Power pellet (50 points) |
  | `@` | Tunnel (ghosts slow down; the left and right edges connect) |
  | space | Empty floor |

Because the layouts never change, a client can read them once from `hello` and then track eaten dots from each `step`'s `eaten` list, without asking for the grid again.

### `reset`

```json
{"cmd":"reset","pacman":true,"level":1,"seed":7,"episodic_life":false,
 "end_on_level_clear":false,"max_ticks":0,
 "record":"runs/demo/ep-0001.pfr","meta":{"run_id":"demo","episode":1}}
```

Every field except `cmd` is optional; missing fields use the server's flag defaults.

| Field | Type | Description |
| --- | --- | --- |
| `pacman` | bool | Classic Pac-Man (`true`) or Ms. Pac-Man (`false`). Defaults to `true` unless the server was started with `--ms`. |
| `level` | int | Starting level, clamped to `1..255`. |
| `seed` | int | PRNG seed. Same seed + same actions = same game. |
| `episodic_life` | bool | See [Episodes and options](#episodes-and-options). |
| `end_on_level_clear` | bool | End the episode when a level is cleared. |
| `max_ticks` | int | Truncate after this many ticks (`0` = no limit). |
| `record` | string | Path of a `.pfr` file to record this episode to. Parent directories are created. Setting it always starts a **new game** (it is never an `episodic_life` continuation). |
| `meta` | any JSON | Stored as-is in the replay header. `run_id`, `checkpoint` and `episode` are shown in the replay viewer's status line. |

Response:

```json
{"ok":true,"tick":0,"maze_index":0,"state":{ "...": "see State below" }}
```

### `step`

```json
{"cmd":"step","action":"left","mode":"tile"}
{"cmd":"step","action":1,"mode":"frames","frames":8}
```

| Field | Type | Description |
| --- | --- | --- |
| `action` | int `0..4` or string | `right`, `left`, `up`, `down` or `noop`. Defaults to `noop`. |
| `mode` | string | `frames` (default), `tile` or `intersection`. See [Step modes](#step-modes). |
| `frames` | int | Tick count for `frames` mode (default 1). |

Response:

```json
{
  "ok": true,
  "tick": 12,
  "ticks": 8,
  "reward": 10,
  "terminated": false,
  "truncated": false,
  "events": ["dot"],
  "eaten": [[12, 23]],
  "state": { "...": "see State below" }
}
```

| Field | Description |
| --- | --- |
| `tick` | Ticks since the game started. |
| `ticks` | Ticks run by this step. |
| `reward` | Points scored during this step (the change in score). |
| `terminated` | The episode ended: game over, or a life lost with `episodic_life`, or a level cleared with `end_on_level_clear`. |
| `truncated` | The episode was cut off by `max_ticks`. |
| `events` | Things that happened during the step, any of: `dot`, `power`, `ghost`, `fruit`, `extra_life`, `death`, `level_clear`. |
| `eaten` | `[x, y]` tile coordinates of dots and power pellets eaten during this step. |
| `maze_index` | Only present when the level changed during the step: the maze now in use. Reset your dot tracking to that maze's starting layout. |

Once an episode has ended, further `step` requests do nothing and keep returning `terminated`/`truncated` until the next `reset`.

### `observe`

```json
{"cmd":"observe","maze":true}
```

Returns `tick`, `maze_index` and `state` without advancing time. With `"maze": true` it also returns `rows`: the **current** maze grid, using the same characters as `hello` but with eaten dots and pellets shown as spaces. Useful for checking a client's own dot tracking, or for simple clients that would rather not track dots themselves.

### `close`

```json
{"cmd":"close"}
```

Responds `{"ok":true}`, finishes any replay being recorded, and exits.

### State

`reset`, `step` and `observe` all return the same `state` object:

```json
{
  "level": 1,
  "score": 10,
  "lives": 3,
  "dots_remaining": 223,
  "wave": "scatter",
  "fright_frames": 0,
  "ghosts_eaten": 0,
  "player": {"x": 97, "y": 184, "dir": 1, "legal": [true, true, false, true]},
  "ghosts": [
    {"x": 96, "y": 88, "dir": 1, "state": "scatter"},
    {"x": 108, "y": 106, "dir": 2, "state": "leaving_house"},
    {"x": 92, "y": 110, "dir": 3, "state": "in_house"},
    {"x": 124, "y": 110, "dir": 3, "state": "in_house"}
  ],
  "fruit": {"active": false}
}
```

| Field | Description |
| --- | --- |
| `level` | Current level (1-based). |
| `score`, `lives`, `dots_remaining` | As shown on screen. `lives` is the number of lives left including the one being played. |
| `wave` | The ghosts' current global mode: `scatter` or `chase`. |
| `fright_frames` | Ticks of frightened mode left after a power pellet (`0` when not frightened). |
| `ghosts_eaten` | Ghosts eaten during the current power pellet (decides the next ghost's points: 200, 400, 800, 1600). |
| `player.x`, `player.y` | Position in **pixels** within the maze. The tile is `((x + 4) // 8, (y + 4) // 8)`. |
| `player.dir` | Direction of travel, numbered like the actions (`0`=right, `1`=left, `2`=up, `3`=down). |
| `player.legal` | Which of the four directions `[right, left, up, down]` can be taken right now. Use it as an action mask. |
| `ghosts` | Always four, in the order Blinky, Pinky, Inky, Clyde. `state` is one of `scatter`, `chase`, `frightened`, `eaten` (eyes going home), `entering_house`, `in_house`, `leaving_house`. |
| `fruit` | `{"active": false}`, or when a bonus fruit is on screen: `x`, `y` (pixels), `symbol` (`cherries`, `strawberry`, `peach`, `apple`, `grapes`, `galaxian`, `bell`, `key`, ...), `points` and `timer`. |

---

## 5. Writing your own client

Any language that can start a subprocess and read and write lines can drive the server. The loop is always:

1. Start `pacfan-headless` with pipes for stdin and stdout.
2. Optionally send `hello` and keep the maze layouts.
3. Send `reset`.
4. Repeatedly choose an action from `state` and send `step`, until `terminated` or `truncated`.
5. Go back to 3 for another episode, or send `close`.

Write each request followed by `\n` and **flush** after every request, then read exactly one response line. Check `ok` on every response.

### Minimal Python client (no dependencies)

This plays one game with a trivial policy: keep going if possible, otherwise take the first legal turn. Run it from the repository root after `make build-headless`.

```python
import json
import subprocess

proc = subprocess.Popen(
    ["./pacfan-headless"],
    stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, bufsize=1,
)

def rpc(**req):
    proc.stdin.write(json.dumps(req) + "\n")
    proc.stdin.flush()
    resp = json.loads(proc.stdout.readline())
    if not resp["ok"]:
        raise RuntimeError(resp["error"])
    return resp

hello = rpc(cmd="hello")
print("maze_hash", hello["maze_hash"])

resp = rpc(cmd="reset", seed=123, record="runs/demo/first-game.pfr",
           meta={"run_id": "demo", "episode": 1})
state = resp["state"]
done = False
while not done:
    legal = state["player"]["legal"]       # [right, left, up, down]
    current = state["player"]["dir"]
    action = current if legal[current] else legal.index(True)
    resp = rpc(cmd="step", action=action, mode="intersection")
    state = resp["state"]
    if "death" in resp["events"]:
        print(f"died at tick {resp['tick']}, lives left {state['lives']}")
    done = resp["terminated"] or resp["truncated"]

print("final score", state["score"], "level", state["level"])
rpc(cmd="close")
proc.wait()
```

Then watch the game it recorded:

```sh
go run . --replay runs/demo/first-game.pfr
```

### Tips

- **Choose a step mode for the job.** `intersection` gives the fewest, most meaningful decisions; `tile` is a good general choice; `frames` gives the most control but makes the most requests.
- **Mask actions with `player.legal`.** Actions into walls are not errors, but they waste steps.
- **Track dots locally.** Start from `hello.mazes[maze_index].rows` after `reset`, clear each `[x, y]` in `eaten` after every step, and reload the starting layout when `maze_index` appears in a step response. `observe` with `"maze": true` is there to check your tracking.
- **Use seeds.** Give each evaluation episode a fixed seed so different agents (or checkpoints) are compared on the same games.
- **Parallelism.** One process is one game. Run several processes for throughput.
- **Go clients.** In Go you can skip the protocol and use `sim.Env` directly (`sim.NewEnv`, `Reset`, `Step`, `sim.NewFileRecorder`), or call `api.ServeNDJSON` on any reader and writer (a socket, for example). See [`cmd/pacfan-headless/main.go`](../cmd/pacfan-headless/main.go) and [`internal/api/api_test.go`](../internal/api/api_test.go).

---

## 6. Replay files (`.pfr`)

A `.pfr` (**P**ac**F**an **R**eplay) file records one episode as its starting options plus the action used on every tick. Because the simulation is deterministic, that is enough to play the game back exactly. Files are small, plain text, and easy to read.

### Recording

- **From the API**: put `"record": "path/to/file.pfr"` in a `reset`. The file is finished when the episode ends (game over, truncation, or a life lost / level cleared that ends it), when another `reset` starts a new game, or on `close` / end of input. With `episodic_life`, continuing resets keep writing to the same file, so one file holds the whole game across lives.
- **From Python**: pass `record_dir=` to `PacFanEnv`, or `options={"record": path, "meta": {...}}` to `env.reset()`. `eval.py --record-dir` and the evaluations in `train_ppo.py` record automatically. See [section 8](#8-running-the-python-ai).

### Format

```text
{"pfr":1,"art":{"file":"game.png","sha256":"e608ed9e..."},"maze_hash":"e624e288...","pacman":false,"level":1,"seed":7,"options":{},"meta":{"fixture":true}}
a 0x112 2x27 1x99 2x26 0x95 2x27 1x45 2x27 1x26 2x27 1x81 2x8
h 600 92236aab61404451
a 2x28 0x126 1x16 2x4 0x12 3x27 0x27 3x27 0x72 2x29 1x41 2x26 1x24 2x26 0x24 2x91
h 1200 2792ed3819509119
...
h 3949 488ab9e1e13872fa
{"end":"game_over","ticks":3949,"score":3480,"level":1,"deaths":3}
```

| Line | Meaning |
| --- | --- |
| First line (JSON) | **Header.** `pfr` format version (1), `art` file and SHA-256, `maze_hash`, `pacman`, starting `level`, `seed`, `options` (`episodic_life`, `end_on_level_clear`, `max_ticks` when set), and the free-form `meta` from the `reset`. |
| `a <action>x<count> ...` | **Action runs**: `2x27` means "action 2 (up) for 27 ticks". At most 16 runs per line. |
| `h <tick> <hash>` | **Checkpoint**: the expected `HashWorld` value (16 hex digits) after that tick. Written every 600 ticks and at the last tick. |
| Last line (JSON) | **Footer.** `end` reason (`game_over`, `life_lost`, `level_clear`, `truncated`, `terminated` or `closed`), total `ticks`, final `score` and `level`, and `deaths`. |

Blank lines and lines starting with `#` are ignored, so you can annotate files by hand.

Every tick is recorded, including ghost-eaten freeze ticks and ticks run inside multi-tick `tile`/`intersection` steps, so a replay does not depend on which step mode recorded it.

Example replays live in [`internal/sim/testdata/`](../internal/sim/testdata). They were recorded with the placeholder `assets/game.png` mazes.

---

## 7. Watching and verifying replays

### Watching in the game window

```sh
go run . --replay path/to/game.pfr               # one file
go run . --replay runs/ppo-demo/eval             # a directory: every .pfr in it, recursively, sorted
go run . --replay game.pfr --replay-speed 4       # start at 4x speed
go run . --replay game.pfr --art assets/game.png  # use a specific spritesheet
go run . --replay game.pfr --zoom 4
```

(or `make build` and run `./pacfan --replay ...`.)

| Flag | Default | Description |
| --- | --- | --- |
| `-r`, `--replay PATH` | | A `.pfr` file, or a directory to play every `.pfr` file in it (searched recursively, sorted by path) as a playlist. |
| `-s`, `--replay-speed X` | `1` | Starting playback speed, `0.25` to `16`. |
| `-a`, `--art FILE` | (see below) | Spritesheet PNG to use. |
| `-z`, `--zoom N` | `3` | Window scale. |

Controls:

| Key | Action |
| --- | --- |
| **Space** | Pause / resume |
| **Right arrow** (while paused) | Step forward one tick (hold to repeat) |
| **=** / **+** | Double the speed (up to 16x) |
| **-** | Halve the speed (down to 0.25x) |
| **N** / **P** | Next / previous file in a directory playlist (wraps around) |
| **Esc** or close the window | Quit |

The bar at the bottom of the window shows the replay's `run_id`, `checkpoint` and `episode` from `meta` (or the file name if there are none), the playlist position, `t=<tick>/<total>` and the speed. Hash checkpoints are checked during playback: if the game ever differs from the recording, playback stops and the bar shows the mismatch in red. The viewer has no audio and does not show the attract or intermission screens.

**Choosing the artwork.** With no `--art`, the viewer uses the embedded asset whose file name matches the replay header's `art.file` (for example `game.png` or `game-real.png`), or the default asset if there is no match. If the `maze_hash` of that art does not match the replay, it refuses to play and tells you which art file the replay needs, in which case pass `--art` with a PNG that has the right mazes.

### Verifying without a window

```sh
./pacfan-headless --verify path/to/game.pfr
# OK ticks=3949 score=3480 level=1 checkpoints=7 hash=488ab9e1e13872fa
```

`--verify` replays the file as fast as possible, checks every checkpoint, and exits with status 0 and an `OK` line, or non-zero with `VERIFY FAILED: ...`. It is useful in scripts and CI, and for checking that a gameplay code change has not changed how existing replays play out.

`--verify` always uses the art given by `--art`, or the default embedded art; it does **not** pick art from the header. For example, if you have `assets/game-real.png` locally, the bundled test replays (recorded with the placeholder art) need:

```sh
./pacfan-headless --art assets/game.png --verify internal/sim/testdata/mspacman_play.pfr
```

---

## 8. Running the Python AI

The [`ml/`](../ml) directory contains a Python package that wraps the API in a [Gymnasium](https://gymnasium.farama.org/) environment and trains agents on it. See also [`ml/README.md`](../ml/README.md).

| File | Contents |
| --- | --- |
| [`ml/pacfan_env.py`](../ml/pacfan_env.py) | `PacFanEnv` (Gymnasium environment that runs one `pacfan-headless` subprocess) and `RewardShapingWrapper`. |
| [`ml/baselines.py`](../ml/baselines.py) | Two non-learning agents (random, and a BFS-based greedy agent) and a throughput benchmark. |
| [`ml/train_ppo.py`](../ml/train_ppo.py) | Trains a MaskablePPO agent (Stable-Baselines3 + sb3-contrib) with periodic checkpoints and recorded evaluations. |
| [`ml/eval.py`](../ml/eval.py) | Evaluates a saved checkpoint on fixed seeds, optionally recording `.pfr` replays. |
| [`ml/tests/test_env.py`](../ml/tests/test_env.py) | Environment tests (Gymnasium `check_env`, determinism, observation modes, vector environments). |

### 8.1 Setup

From the repository root:

```sh
make build-headless              # the Python code finds ./pacfan-headless automatically

cd ml
python3 -m venv .venv
source .venv/bin/activate        # fish: source .venv/bin/activate.fish
pip install -e ".[dev]"          # gymnasium, numpy, pytest
pip install -e ".[dev,train]"    # add torch, stable-baselines3, sb3-contrib, tensorboard for training
```

Python 3.10 or newer is required. The binary is looked for at `<repo>/pacfan-headless`, then on `PATH`; every script also takes `--binary PATH`.

Remember to run `make build-headless` again after changing Go code; the Python side runs whatever binary is there.

### 8.2 Check the environment

```sh
pytest
```

### 8.3 Baselines

```sh
python baselines.py --episodes 5 --step-mode tile
python baselines.py --benchmark-workers 1 4 8     # also measure steps/second with 1, 4 and 8 parallel processes
```

This runs two agents on seeds `1000, 1001, ...` and prints score and level statistics:

- **RandomLegalAgent**: picks a random legal direction each step.
- **GreedyBFSAgent**: uses breadth-first search through the maze to head for frightened ghosts, fruit and dots, and avoids dangerous ghosts within `--avoid-dist` tiles.

These give a floor that a learned agent should beat.

| Option | Default | Description |
| --- | --- | --- |
| `--episodes` | `5` | Episodes per agent. |
| `--step-mode` | `tile` | `frames`, `tile` or `intersection`. |
| `--max-ticks` | `12000` | Truncation limit per episode. |
| `--avoid-dist` | `3` | GreedyBFSAgent's danger distance, in tiles. |
| `--ms` | off | Ms. Pac-Man instead of classic Pac-Man. |
| `--art` | | Spritesheet passed to the server as `--art`. |
| `--benchmark-workers N ...` | | Run the throughput benchmark with these worker counts. |
| `--benchmark-steps` | `300` | Steps per worker in the benchmark. |

### 8.4 Train with MaskablePPO

```sh
python train_ppo.py \
  --total-timesteps 100000 \
  --n-envs 4 \
  --step-mode tile \
  --eval-freq 25000 \
  --run-dir ../runs/ppo-demo
```

`train_ppo.py` runs `--n-envs` environments (each its own `pacfan-headless` process), wraps them with reward shaping and action masking, and trains a `MaskablePPO` policy. Every `--eval-freq` timesteps it saves a checkpoint, plays `--eval-episodes` games on fixed seeds starting at `--eval-seed-start`, records each as a `.pfr`, and keeps the best checkpoint by mean score.

Main options:

| Option | Default | Description |
| --- | --- | --- |
| `--total-timesteps` | `100000` | Training length, in agent steps (not ticks). |
| `--n-envs` | `4` | Parallel environments / server processes. |
| `--step-mode` | `tile` | `frames`, `tile` or `intersection`. |
| `--frames-per-step` | `4` | Ticks per step in `frames` mode. |
| `--obs-mode` | `grid` | `grid` (CNN over maze channels + scalars) or `bfs` (54 hand-made features). |
| `--episodic-life` / `--no-episodic-life` | on | End an episode on every lost life. |
| `--end-on-level-clear` | off | End an episode when a level is cleared. |
| `--max-ticks` | `15000` | Truncation limit. |
| `--score-scale` | `0.01` | Reward = score change x this ... |
| `--death-penalty` | `5.0` | ... minus this on a death ... |
| `--level-clear-bonus` | `10.0` | ... plus this on a level clear ... |
| `--step-penalty` | `0.01` | ... minus this every step. |
| `--n-steps`, `--batch-size`, `--learning-rate`, `--gamma`, `--ent-coef` | `256`, `128`, `3e-4`, `0.99`, `0.01` | PPO hyperparameters. |
| `--seed` | `42` | Training seed. |
| `--eval-freq`, `--eval-episodes`, `--eval-seed-start` | `25000`, `5`, `1000` | Evaluation schedule. |
| `--run-dir` | `runs/ppo-<timestamp>` | Output directory. |
| `--start-level`, `--ms`, `--art`, `--binary` | | Game and server settings. |

Output layout:

```text
runs/ppo-demo/
├── run.json                      # all settings, run_id, git commit
├── episodes.jsonl                # one line per training episode: reward, length, score, level, tick
├── tb/                           # TensorBoard logs (tensorboard --logdir runs/ppo-demo/tb)
├── checkpoints/ckpt-00025000.zip # a model every --eval-freq timesteps
├── eval/ckpt-00025000/
│   ├── seed-1000.pfr             # recorded evaluation games
│   ├── seed-1001.pfr
│   └── metrics.json
└── best/
    ├── model.zip                 # best checkpoint by evaluation mean score
    └── metrics.json
```

The evaluation replays carry `run_id`, `checkpoint` and `episode` in their `meta`, which the replay viewer shows. To watch the agent improve over training, play the whole `eval` directory as a playlist:

```sh
cd ..                                   # back to the repository root
go run . --replay runs/ppo-demo/eval --replay-speed 2
```

### 8.5 Evaluate a checkpoint

```sh
python eval.py ../runs/ppo-demo/best/model.zip --episodes 10 --record-dir ../runs/ppo-demo/eval-manual
../pacfan-headless --verify ../runs/ppo-demo/eval-manual/seed-1000.pfr
cd .. && go run . --replay runs/ppo-demo/eval-manual
```

`eval.py` plays `--episodes` games on seeds from `--seed-start` (default `1000`) and prints a summary. With `--record-dir` it writes `seed-<N>.pfr` for each game plus `metrics.json`. Use the same `--step-mode`, `--obs-mode` and `--frames-per-step` as the model was trained with. Other options: `--max-ticks` (default `20000`), `--start-level`, `--ms`, `--art`, `--binary`.

### 8.6 Using `PacFanEnv` directly

```python
from pacfan_env import PacFanEnv

with PacFanEnv(step_mode="intersection", obs_mode="bfs", record_dir="../runs/my-agent") as env:
    obs, info = env.reset(seed=1000)
    done = False
    while not done:
        mask = env.action_masks()                    # legal directions
        action = int(mask.nonzero()[0][0])           # replace with your policy
        obs, reward, terminated, truncated, info = env.step(action)
        done = terminated or truncated
    print("score", info["state"]["score"])
```

Constructor options:

| Option | Default | Description |
| --- | --- | --- |
| `binary` | found automatically | Path to `pacfan-headless`. |
| `art` | | Spritesheet PNG passed as `--art`. |
| `ms_pacman` | `False` | Ms. Pac-Man instead of classic Pac-Man (passes `--ms` to the server). |
| `start_level` | `1` | Starting level. |
| `step_mode` | `"tile"` | `"frames"`, `"tile"` or `"intersection"`. |
| `frames_per_step` | `4` | Ticks per step in `frames` mode. |
| `episodic_life`, `end_on_level_clear`, `max_ticks` | `False`, `False`, `0` | Passed to `reset`. |
| `obs_mode` | `"grid"` | See below. |
| `allow_noop` | `False` | Action space of 5 (with `noop`) instead of 4. |
| `record_dir` | | Record every episode to `ep-<NNNN>-seed-<seed>.pfr` in this directory. |
| `render_mode` | | `"ansi"` makes `env.render()` return a text picture of the maze. |

Observation modes:

| `obs_mode` | Shape | Contents |
| --- | --- | --- |
| `grid` | `{"grid": (12, 31, 28), "scalars": (11,)}` | Grid channels: walls, dots, power pellets, tunnels, player, one channel per normal ghost (4), frightened ghosts, eaten ghosts, fruit. Scalars: fright time, chase wave, dots left, lives, ghosts eaten, fruit active, player direction (one-hot of 4), level. |
| `grid_only` | `(12, 31, 28)` | Just the grid channels. |
| `bfs` | `(54,)` | Shortest-path distance and first-step direction to the nearest dot, power pellet, fruit and each ghost (plus whether each ghost is dangerous or frightened), the legal-move mask, fright time, dots left, chase wave and player direction. |

All values are scaled to `0..1`. The reward is the raw score change; wrap the environment in `RewardShapingWrapper` for the scaled and shaped reward used in training. `info` contains `tick`, `maze_index`, `events`, `eaten`, `action_mask` and the raw `state` (plus `seed`, `maze_hash` and `art` after `reset`). `reset(seed=N)` passes `N` to the game PRNG; `reset(options={"record": path, "meta": {...}})` records that one episode.

---

## 9. Benchmarks

Benchmarks measure each layer separately, so you can see where time goes before trying to make API mode faster, and check that a change really helped.

| Benchmark | Where | Measures |
| --- | --- | --- |
| `BenchmarkEnvTick` | [`internal/sim/env_test.go`](../internal/sim/env_test.go) | One raw simulation tick (`Env.TickOne`). |
| `BenchmarkEnvStep/{frames4,tile,intersection}` | [`internal/sim/env_test.go`](../internal/sim/env_test.go) | One `Env.Step` in each step mode, with a legal-move policy. Reports `ticks/step` and `ns/tick`. |
| `BenchmarkNDJSONStep` | [`internal/api/api_test.go`](../internal/api/api_test.go) | A 4-frame step through the NDJSON server in-process (JSON decode, step, JSON encode), over in-memory pipes. |
| `BenchmarkHeadlessProcess/{frames4,tile,intersection}` | [`cmd/pacfan-headless/main_test.go`](../cmd/pacfan-headless/main_test.go) | A step round trip against the real `pacfan-headless` binary over OS pipes, the way a client sees it. The binary is built into a temporary directory on first use. Reports `steps/s` and `ticks/step`. |
| `bench.py` | [`ml/bench.py`](../ml/bench.py) | From Python: the raw request round trip (`rpc`), observation building alone (`encode/<obs_mode>`), and the full `PacFanEnv.step()` (`env/<obs_mode>`), for each step mode. |
| `baselines.py --benchmark-workers` | [`ml/baselines.py`](../ml/baselines.py) | Total steps/s with several environments in parallel (`AsyncVectorEnv`). |

All of them play Ms. Pac-Man level 1 with the placeholder `assets/game.png` mazes and a fixed seed, so results are comparable between machines and over time. The Go step benchmarks and `bench.py` use the same deterministic legal-move policy, so their `ticks/step` match.

### Running

```sh
make bench                                 # all Go benchmarks once
make bench BENCH=EnvStep COUNT=6           # one benchmark, 6 runs
go test -run '^$' -bench . -benchmem ./internal/sim ./internal/api ./cmd/pacfan-headless

cd ml
python bench.py                            # all step modes, grid and bfs observations
python bench.py --steps 20000 --modes tile --obs-modes bfs
python baselines.py --episodes 1 --benchmark-workers 1 4 8 --benchmark-steps 5000
```

Run `make build-headless` before the Python benchmarks so they use the current Go code.

### Comparing before and after a change

Results vary by 10-20% between runs, so take several samples and compare them with [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
go install golang.org/x/perf/cmd/benchstat@latest
make bench BENCH='EnvStep|HeadlessProcess' COUNT=10 > old.txt
# ... make the change ...
make bench BENCH='EnvStep|HeadlessProcess' COUNT=10 > new.txt
benchstat old.txt new.txt
```

To find what to optimize, profile a benchmark:

```sh
go test -run '^$' -bench EnvStep/tile -cpuprofile cpu.out ./internal/sim
go tool pprof -top cpu.out                 # or: go tool pprof -http=:8080 cpu.out
```

Whatever you change, `go test ./...` (in particular the replay tests and `--verify` of existing `.pfr` files) must still pass: a speedup must not change how the game plays.

### Baseline (Apple M1)

| Layer | frames (4 ticks) | tile (about 8 ticks) | intersection (about 35 ticks) |
| --- | --- | --- | --- |
| `Env.Step` (simulation only) | 15 µs | 31 µs | 159 µs |
| `pacfan-headless` over pipes, from Go | 33 µs | 51 µs | 165 µs |
| Python `rpc` (round trip only) | 37 µs | 51 µs | 145 µs |
| Python `encode/grid` | 6 µs | 6 µs | 6 µs |
| Python `encode/bfs` | 320 µs | 320 µs | 320 µs |
| Python `env/grid` (full step) | 47 µs | 62 µs | 158 µs |
| Python `env/bfs` (full step) | 365 µs | 386 µs | 482 µs |

A single tick costs about 3.5-4.5 µs. What these numbers say:

- With `grid` observations, most of the time is the simulation itself; process I/O and JSON add roughly 15-20 µs per step. Faster ticks speed up training directly.
- With `bfs` observations, the Python breadth-first search (about 320 µs) dwarfs everything else. Speeding up the Go side barely matters there; the encoder is the thing to optimize.

---

## 10. Troubleshooting

| Problem | Cause and fix |
| --- | --- |
| `replay maze_hash mismatch: replay requires "game.png" ...` | The replay was recorded with different maze layouts. Pass `--art` with the PNG it names (for example `--art assets/game.png` for the bundled test replays when `assets/game-real.png` exists). |
| `warning: artwork SHA-256 differs ...` | The PNG is different but the mazes match. Harmless; the replay plays the same. |
| `replay hash mismatch at tick N` | The game played out differently from the recording, usually because gameplay code changed after the replay was recorded. Re-record, or find the change with `--verify` and `git bisect`. |
| `must call reset before step` | Send `reset` first. |
| `unknown step mode "..."` / `unknown action "..."` | Use `frames`/`tile`/`intersection` and `0..4` or `right`/`left`/`up`/`down`/`noop`. |
| Python: `pacfan-headless died (...)` | The server exited; the message includes its exit code and stderr. Check the binary is built (`make build-headless`) and that any `--art` path exists. |
| Client hangs waiting for a response | The request was not followed by a newline, or stdin was not flushed. |
| The Python AI ignores your Go changes | Rebuild with `make build-headless`. |
