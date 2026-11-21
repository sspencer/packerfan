# Scriptable & Machine Learning API Design (`scriptable.md`)

This document specifies how to expose `pacfan` to external programs (reinforcement learning agents, search/planning bots, or scripted test harnesses) so they can inspect complete game state and control Pac-Man / Ms. Pac-Man deterministically.

---

## 1. Why `pacfan` Fits ML/RL Cleanly

The codebase already separates simulation from presentation:

- **Zero mutable globals**: All runtime state lives inside a `donburi.World` instance, so multiple independent game instances can run concurrently in the same process or across worker processes.
- **Headless-ready simulation**: `internal/config`, `internal/component`, `internal/archetype`, `internal/event`, `internal/system`, and `internal/scene` have **zero Raylib imports**.
- **Injectable I/O boundaries**: Controls (`system.InputProvider`), rendering (`scene.Drawer`), audio (`audio.Player`), and maze pixel sampling (`isPixelLit`) are already decoupled interfaces/callbacks.
- **Deterministic RNG**: `component.PRNG` uses a self-contained simulated 8 KB arcade ROM table. Adding a configurable seed makes episodes 100% reproducible.

### Only Two Small Prerequisites in Existing Code

1. **Pure-Go PNG maze loader (no Raylib window)**: `internal/render/art.go` currently reads `assets/game.png` via Raylib's GPU texture API. For headless training (no display server / OpenGL context), add a tiny pure-Go `image/png` helper (identical to the one already used in `internal/system/system_test.go`) that reads `assets/game.png` from `assetsFS` and returns an `isPixelLit(x, y int) bool` callback.
2. **Configurable PRNG seed**: Extend `component.PRNGData` (`internal/component/prng.go`) with `InitWithSeed(seed int64)` (defaulting to `42`) so training runs can randomize seeds across episodes while keeping exact reproducibility for a given seed.

---

## 2. Operating Modes

Supporting two modes lets you **train fast headlessly** and **visually watch trained agents play**:

| Mode | CLI Example | Window / Audio | Pacing | Intended Use |
| --- | --- | --- | --- | --- |
| **Headless Lockstep (Training)** | `pacfan --headless --api stdio` | None (`NoopPlayer`, no Raylib) | Stepped strictly on demand (no 60.6 Hz sleep) | RL training (`PPO`, `DQN`, `MCTS`, genetic/evolutionary), vectorized parallel rollouts (10k–50k+ frames/sec per worker) |
| **Visual Lockstep / Spectate** | `pacfan --api tcp://127.0.0.1:7777 --zoom 3` | Full Raylib window + optional audio | 60.6 Hz (or `--speed 2.0` fast-forward) | Watching a trained policy live, recording demos, debugging failure cases |

In **Headless Lockstep** mode, non-interactive presentation scenes (`AttractScene`, `ReadyScene`, `GhostEatenFreezeScene`, `RoundClearedScene`, `IntermissionScene`, `PacmanDyingScene`) are bypassed (or collapsed to 0 frames) so 100% of CPU time goes into gameplay transitions and decisions.

---

## 3. Transport & Protocol Choice

### Recommended Primary Transport: Newline-Delimited JSON (`NDJSON`)

- **`stdio` (`--api stdio`)** — Default for ML training. The trainer (e.g., Python Gymnasium) spawns `pacfan --headless --api stdio` as a child process, writes one JSON command per line to `stdin`, and reads one JSON response per line from `stdout`.
  - Zero port collisions when spawning `N` parallel environments (`AsyncVectorEnv`).
  - Automatic cleanup on EOF if the Python trainer crashes or exits.
  - Zero third-party Go dependencies (`encoding/json` + `bufio.Scanner`).
- **Local Socket (`--api unix:///tmp/pacfan.sock` or `--api tcp://127.0.0.1:7777`)** — Best for attaching an external agent to a live visual window (bind TCP strictly to `127.0.0.1` by default).

> **Future Performance Path**: NDJSON over `stdio` typically sustains 10,000–30,000 steps/sec per worker (and scales linearly across CPU cores). If serialization ever becomes the bottleneck, the same `Session` Go API can be exposed over length-prefixed binary frames (or batched multi-world stepping inside one Go process) without changing any ECS code.

---

## 4. Wire Protocol Specification

All interactions follow a **lockstep request → response** cycle (matching the OpenAI Gym / Gymnasium contract). Every request and response is a single line of UTF-8 JSON terminated by `\n`.

### 4.1 Commands (Client → `pacfan`)

#### `reset` — Start a new episode

```json
{
  "cmd": "reset",
  "seed": 12345,
  "pacman": true,
  "level": 1,
  "episodic_life": false,
  "compact_maze": false,
  "max_steps": 10000
}
```

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `seed` | `int64` | `42` | Seed for `component.PRNG` (deterministic episode replay) |
| `pacman` | `bool` | `true` (`false` with CLI `--ms`) | `true` for classic Pac-Man, `false` for Ms. Pac-Man |
| `level` | `int` | `1` | Starting level (`1..255`, useful for curriculum learning) |
| `episodic_life` | `bool` | `false` | If `true`, sets `terminated: true` on losing a single life (standard Atari/Pac-Man RL convention) |
| `compact_maze` | `bool` | `false` | If `true`, omits the static wall layout on `step` responses (sending it only on `reset` and level advance) and sends only remaining dot/power coordinates |
| `max_steps` | `int` | `0` (unlimited) | Sets `truncated: true` when the episode step count reaches `max_steps` |

#### `step` — Apply an action and advance the simulation

```json
{
  "cmd": "step",
  "action": 1,
  "frames": 4,
  "step_mode": "frames"
}
```

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `action` | `int` or `string` | required | Direction to request: `0` / `"right"`, `1` / `"left"`, `2` / `"up"`, `3` / `"down"`, or `4` / `"noop"` (keep current direction) |
| `frames` | `int` | `1` | Number of simulation ticks to advance when `step_mode == "frames"` (stops early on death, level clear, or game over) |
| `step_mode` | `string` | `"frames"` | `"frames"`: advance `frames` ticks. `"tile"`: advance until Pac-Man enters a new centered tile (or stops at a wall / dies / clears level). `"intersection"`: advance until Pac-Man reaches a tile with ≥2 legal non-reverse moves (dramatically shortens RL horizons!) |

#### `observe` — Read current state without advancing

```json
{"cmd": "observe"}
```

#### `close` — Shut down cleanly

```json
{"cmd": "close"}
```

---

### 4.2 Response Schema (`pacfan` → Client)

Every `reset`, `step`, and `observe` command returns a single JSON object mapped directly from the ECS world:

```json
{
  "ok": true,
  "step": 42,
  "tick": 168,
  "reward": 10.0,
  "terminated": false,
  "truncated": false,
  "events": ["dot_eaten"],
  "state": {
    "level": 1,
    "pacman_mode": true,
    "score": 450,
    "high_score": 12400,
    "lives": 3,
    "dots_eaten": 41,
    "dots_remaining": 203,
    "wave_mode": "scatter",
    "wave_frames": 168,
    "is_fright": false,
    "fright_frames": 0,
    "num_ghosts_eaten": 0,
    "elroy_stage": 0,
    "player": {
      "x": 96,
      "y": 184,
      "tile_x": 12,
      "tile_y": 23,
      "dir": 1,
      "dot_pause": 1,
      "legal_actions": [true, true, false, false]
    },
    "ghosts": [
      {
        "id": "blinky",
        "x": 88,
        "y": 88,
        "tile_x": 11,
        "tile_y": 11,
        "dir": 1,
        "state": "scatter",
        "frightened_in_house": false,
        "is_eaten": false,
        "target_x": 200,
        "target_y": -24
      }
    ],
    "fruit": {
      "active": false,
      "moving": false,
      "symbol": "cherries",
      "points": 100,
      "x": 108,
      "y": 136,
      "tile_x": 13,
      "tile_y": 17,
      "timer": 0
    },
    "maze": {
      "width": 28,
      "height": 31,
      "rows": [
        "############################",
        "#............##............#",
        "..."
      ]
    }
  }
}
```

#### Field Reference

- **Top-level RL fields**:
  - `reward`: Default score delta accrued during this `step` (`score_after - score_before`). Agents that want custom reward shaping can ignore `reward` and compute their own from `events`, `dots_remaining`, `lives`, and `terminated`.
  - `events`: Ordered list of gameplay events that occurred during the ticks of this `step`: `"dot_eaten"`, `"powerup_eaten"`, `"ghost_eaten"`, `"fruit_eaten"`, `"extra_life"`, `"player_eaten"`, `"level_completed"`, `"game_over"`.
  - `terminated`: `true` on `GameOver` (or on `player_eaten` if `episodic_life: true`).
  - `truncated`: `true` if `step >= max_steps`.
- **`player.legal_actions`**: `[right, left, up, down]` booleans computed via `system.CanPlayerMove(maze, tileX, tileY, dir)`. Enables **action masking** (e.g., `MaskablePPO`), preventing wasted exploration into walls.
- **`ghosts`**: Always 4 entries in canonical order (`blinky`, `inky`, `pinky`, `clyde`). `state` is one of `"scatter"`, `"chase"`, `"frightened"`, `"eaten"`, `"entering_house"`, `"in_house"`, `"leaving_house"`.
- **`maze.rows`** (31 strings of 28 ASCII characters, ~900 bytes total — already produced directly by `MazeData.Rows()` and `MazeData.String()` in `internal/component/maze.go`, while `MazeData.PrettyString()` preserves the 3-character Unicode debug view):
  - `#` = `Wall`
  - `-` = `Door` (ghost house gate)
  - `.` = `Dot`
  - `O` = `Power` (energizer)
  - `@` = `Tunnel`
  - ` ` (space) = `Empty`
  - When `compact_maze: true`, `maze.rows` is sent on `reset` and whenever `level` changes; on normal `step`s, `maze` instead sends `dots: [[x,y], ...]` and `power: [[x,y], ...]` (or a 109-byte base64 bitmask of remaining edible tiles).

---

## 5. Designing Observations & Rewards for ML Agents

Because `pacfan` exposes structured semantic state instead of raw RGB pixels, training is dramatically faster and far more sample-efficient than pixel-based Atari RL.

### 5.1 Recommended Neural Network Representations

#### Option A: Multi-Channel Spatial Grid `(14, 31, 28)` (Best for CNN / ResNet PPO & DQN)

Stack `float32` channels aligned to the exact `31 × 28` maze grid:

| Channel | Encoding |
| --- | --- |
| `0` | Walls & ghost door (`1.0` where impassable to Pac-Man) |
| `1` | Remaining dots (`1.0` per dot) |
| `2` | Remaining power pellets (`1.0` per energizer) |
| `3` | Tunnel tiles (`1.0` where ghosts slow down) |
| `4` | Player position (`1.0` at `(tile_x, tile_y)`) |
| `5..8` | Dangerous ghosts (`Scatter`/`Chase`) — 1 channel per ghost, or combined + previous-step trail so the CNN sees ghost velocity/heading |
| `9` | Frightened ghosts (`1.0` where vulnerable, scaled by `fright_frames / 363.0`) |
| `10` | Eaten eyes returning to house (`1.0`) |
| `11` | Active bonus fruit (`1.0` at `(tile_x, tile_y)`) |
| `12` | Legal action mask / normalized global scalars broadcast or concatenated after the CNN flatten layer (`fright_frames`, `wave_mode`, `dots_remaining / 244`, `player.dir` one-hot) |

#### Option B: Feature Vector (Best for Lightweight MLPs, Imitation Learning, or Neuroevolution)

Compute graph/BFS features on the 28×31 maze graph:
- Shortest maze-path distance (via BFS) and first-step direction to: nearest dot, nearest power pellet, active fruit, and each of the 4 ghosts (separated into dangerous vs. frightened).
- Remaining `fright_frames` (normalized), `dots_remaining` ratio, `legal_actions` mask, and whether Pac-Man is near a dead-end / junction.

### 5.2 Action Space & Step Granularity

- **Action space**: `Discrete(4)` (`Right`, `Left`, `Up`, `Down`) — optionally `Discrete(5)` with `Noop`.
- **Choosing `step_mode`**:
  - `step_mode: "frames", frames: 4`: Closest to real-time control (~15 decisions/sec, preserves fine corner-cutting and mid-corridor reversals).
  - `step_mode: "tile"` or `"intersection"`: Reduces a full Level 1 clear from ~8,000 frames to ~600–1,200 decisions, making PPO/DQN converge an order of magnitude faster during initial experiments!

### 5.3 Reward Shaping Suggestions

Start simple, then tune only if needed:
- **Baseline**: Arcade score delta (`+10` dot, `+50` power, `+200..1600` ghost, `+100..5000` fruit), scaled by `0.01` so value targets stay well-conditioned.
- **Helpful shaping**:
  - Small death penalty (e.g., `-5.0` on `"player_eaten"`) so the agent values survival even when it has extra lives.
  - Level-clear bonus (e.g., `+10.0` on `"level_completed"`).
  - Tiny step penalty (e.g., `-0.01` per step) to discourage stalling in safe loops.
  - Enable `episodic_life: true` during training (standard DeepMind Atari practice), while evaluating full 3-life games with `episodic_life: false`.

---

## 6. Implementation Blueprint Inside `pacfan`

### 6.1 Proposed Package Layout

```text
internal/
├── component/
│   └── prng.go                  # Add NewPRNGDataWithSeed(seed int64) & InitWithSeed(seed int64)
├── api/                         # NEW package (zero Raylib imports)
│   ├── protocol.go              # Request/Response/State structs, Snapshot(w), ApplyAction
│   ├── maze_loader.go           # Pure-Go image/png pixel-lit loader for headless mode
│   ├── session.go               # Headless lockstep Session (Reset, Step, Observe, event collector)
│   ├── server.go                # NDJSON transport loop over io.Reader/io.Writer (stdio, Unix, TCP)
│   ├── remote_input.go          # system.InputProvider adapter for live visual spectating
│   └── api_test.go              # Determinism, action legality, event capture & throughput benchmarks
└── ...
main.go                          # Add --headless, --api, and --seed CLI flags
```

### 6.2 How `Session.Step` Executes Headlessly

```go
// Pseudocode for internal/api/session.go
func (s *Session) Step(req StepRequest) Response {
    scoreBefore := s.gameState().Score
    s.collector.Clear() // subscribes to DotEaten, GhostEaten, PlayerEaten, LevelCompleted, etc.

    ticks := 0
    for !s.stepDone(req, ticks) {
        s.input.SetAction(req.Action)
        s.pipeline.Update(s.world)
        ticks++

        gs := s.gameState()
        if gs.PlayerDying {
            s.collector.Add("player_eaten")
            alive := system.ConsumeLifeAndRespawn(s.world)
            if !alive {
                s.collector.Add("game_over")
            }
            break
        }
        if gs.LevelComplete {
            s.collector.Add("level_completed")
            system.AdvanceToNextLevel(s.world, s.isPixelLit)
            break
        }
        if gs.GhostEatenPause > 0 {
            // Skip visual freeze in headless mode; eyes immediately start returning.
            gs.GhostEatenPause = 0
            gs.GhostEatenEntity = donburi.Null
            gs.GhostEatenScore = 0
        }
    }

    s.stepCount++
    gs := s.gameState()
    terminated := gs.GameOver || (s.episodicLife && s.collector.Has("player_eaten"))
    truncated := s.maxSteps > 0 && s.stepCount >= s.maxSteps

    return Response{
        OK:         true,
        Step:       s.stepCount,
        Tick:       s.totalTicks,
        Reward:     float64(gs.Score - scoreBefore),
        Terminated: terminated,
        Truncated:  truncated,
        Events:     s.collector.Events(),
        State:      Snapshot(s.world, s.compactMaze),
    }
}
```

### 6.3 Live Visual Spectating Flow

When running with a Raylib window (`pacfan --api tcp://127.0.0.1:7777`):

1. `main.go` starts a goroutine listening on `127.0.0.1:7777` (accepting one agent connection at a time).
2. A thread-safe `api.RemoteInput` implements `system.InputProvider` and feeds the agent's latest requested direction into `Pipeline`.
3. On each simulation frame (or in lockstep with the agent's `step` messages), the engine snapshots `World` and writes the observation over the socket while Raylib continues rendering the window at 60.6 Hz!
4. Optionally, the human can still press `Space` to pause or `M` to mute while watching the AI play.

---

## 7. Example Python Gymnasium Client

Once `internal/api` is implemented, any Python RL library (Stable-Baselines3, CleanRL, RLlib, TorchRL, JAX) can train on `pacfan` with a thin wrapper like this:

```python
import json
import subprocess
import numpy as np
import gymnasium as gym
from gymnasium import spaces


class PacFanEnv(gym.Env):
    """Gymnasium wrapper communicating with `pacfan --headless --api stdio`."""

    metadata = {"render_modes": ["none"]}

    def __init__(self, binary="./pacfan", pacman=True, frames_per_step=4, episodic_life=True):
        super().__init__()
        self.pacman = pacman
        self.frames_per_step = frames_per_step
        self.episodic_life = episodic_life

        args = [binary, "--headless", "--api", "stdio"]
        if not pacman:
            args.append("--ms")
        self.proc = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            text=True,
            bufsize=1,
        )

        self.action_space = spaces.Discrete(4)  # 0=right, 1=left, 2=up, 3=down
        # Example: 12-channel spatial grid (31 rows x 28 cols)
        self.observation_space = spaces.Box(0.0, 1.0, shape=(12, 31, 28), dtype=np.float32)

    def _rpc(self, payload: dict) -> dict:
        self.proc.stdin.write(json.dumps(payload) + "\n")
        self.proc.stdin.flush()
        line = self.proc.stdout.readline()
        if not line:
            raise RuntimeError("pacfan subprocess exited unexpectedly")
        return json.loads(line)

    def _encode(self, state: dict) -> np.ndarray:
        obs = np.zeros((12, 31, 28), dtype=np.float32)
        for y, row in enumerate(state["maze"]["rows"]):
            for x, ch in enumerate(row):
                if ch in ("#", "-"):
                    obs[0, y, x] = 1.0
                elif ch == ".":
                    obs[1, y, x] = 1.0
                elif ch == "O":
                    obs[2, y, x] = 1.0
                elif ch == "@":
                    obs[3, y, x] = 1.0

        p = state["player"]
        if 0 <= p["tile_y"] < 31 and 0 <= p["tile_x"] < 28:
            obs[4, p["tile_y"], p["tile_x"]] = 1.0

        for idx, g in enumerate(state["ghosts"]):
            gx, gy = g["tile_x"], g["tile_y"]
            if not (0 <= gy < 31 and 0 <= gx < 28):
                continue
            if g["state"] in ("scatter", "chase"):
                obs[5 + idx, gy, gx] = 1.0
            elif g["state"] == "frightened":
                obs[9, gy, gx] = min(1.0, state["fright_frames"] / 363.0)
            elif g["state"] in ("eaten", "entering_house"):
                obs[10, gy, gx] = 1.0

        f = state["fruit"]
        if f["active"] and 0 <= f["tile_y"] < 31 and 0 <= f["tile_x"] < 28:
            obs[11, f["tile_y"], f["tile_x"]] = 1.0
        return obs

    def reset(self, *, seed=None, options=None):
        super().reset(seed=seed)
        resp = self._rpc({
            "cmd": "reset",
            "seed": int(seed) if seed is not None else 42,
            "pacman": self.pacman,
            "episodic_life": self.episodic_life,
        })
        return self._encode(resp["state"]), {"raw": resp["state"], "action_mask": resp["state"]["player"]["legal_actions"]}

    def step(self, action):
        resp = self._rpc({
            "cmd": "step",
            "action": int(action),
            "frames": self.frames_per_step,
        })
        info = {
            "events": resp["events"],
            "score": resp["state"]["score"],
            "level": resp["state"]["level"],
            "action_mask": resp["state"]["player"]["legal_actions"],
        }
        return self._encode(resp["state"]), float(resp["reward"]), bool(resp["terminated"]), bool(resp["truncated"]), info

    def close(self):
        if self.proc.poll() is None:
            try:
                self._rpc({"cmd": "close"})
            except Exception:
                pass
            self.proc.terminate()
            self.proc.wait(timeout=2)
```

---

## 8. Suggested Implementation Milestones

| Phase | Scope | Effort |
| --- | --- | :---: |
| **Phase 1: Core Headless API** | Seedable `PRNG`, pure-Go PNG maze loader, `internal/api` `Session` + `stdio` NDJSON protocol, `--headless` / `--api stdio` flags, determinism & benchmark tests | **M** |
| **Phase 2: Python Gymnasium Env & Baseline Agent** | `ml/pacfan_env.py` wrapper, random/heuristic sanity check, `MaskablePPO` or `DQN` training script with vectorized `AsyncVectorEnv` | **M** |
| **Phase 3: Live Visual Spectating & Replays** | Unix/TCP socket transport (`--api tcp://127.0.0.1:7777`), `RemoteInput` provider for rendered Raylib window, optional action-log replay recording/playback | **S** |
| **Phase 4: High-Throughput Batching (Optional)** | Compact binary framing or multi-world batched `step_batch` command if training exceeds NDJSON IPC throughput | **M** |
