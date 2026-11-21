"""Gymnasium environment and wrappers for the headless pacfan simulator."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from collections import deque
from pathlib import Path
from typing import Any

import gymnasium as gym
from gymnasium import spaces
import numpy as np

TILE_SIZE = 8
MAZE_WIDTH = 28
MAZE_HEIGHT = 31
MAX_FRIGHT_FRAMES = 363.0
MAX_BFS_DIST = 60.0

GRID_CHANNELS = 12
SCALAR_DIM = 11
BFS_OBS_DIM = 54

# Canonical action order matching Go's config.Direction:
# 0=right, 1=left, 2=up, 3=down, 4=noop
DIR_DELTAS = (
    (1, 0),   # 0: right
    (-1, 0),  # 1: left
    (0, -1),  # 2: up
    (0, 1),   # 3: down
)


def default_binary_path() -> str:
    """Locate the pacfan-headless binary relative to the repo root or on PATH."""
    repo_candidate = Path(__file__).resolve().parent.parent / "pacfan-headless"
    if repo_candidate.is_file() and os.access(repo_candidate, os.X_OK):
        return str(repo_candidate)
    found = shutil.which("pacfan-headless")
    if found:
        return found
    return str(repo_candidate)


def _pixel_to_center_tile(x: int, y: int) -> tuple[int, int]:
    tx = ((x + TILE_SIZE // 2) // TILE_SIZE) % MAZE_WIDTH
    ty = max(0, min(MAZE_HEIGHT - 1, (y + TILE_SIZE // 2) // TILE_SIZE))
    return tx, ty


class _MazeCache:
    """Pre-parsed static arrays for one of the five arcade mazes."""

    __slots__ = ("index", "dots", "walls", "tunnels", "init_dots", "init_power", "passable")

    def __init__(self, index: int, dots: int, rows: list[str]) -> None:
        self.index = index
        self.dots = max(1, dots)
        self.walls = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)
        self.tunnels = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)
        self.init_dots = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)
        self.init_power = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)

        for y, row in enumerate(rows):
            for x, ch in enumerate(row):
                if ch == "#" or ch == "-":
                    self.walls[y, x] = 1.0
                elif ch == ".":
                    self.init_dots[y, x] = 1.0
                elif ch == "O":
                    self.init_power[y, x] = 1.0
                elif ch == "@":
                    self.tunnels[y, x] = 1.0

        self.passable = self.walls == 0.0


class PacFanEnv(gym.Env[Any, int]):
    """Gymnasium environment driving `pacfan-headless` over piped NDJSON."""

    metadata = {"render_modes": ["ansi"]}

    def __init__(
        self,
        *,
        binary: str | None = None,
        art: str | None = None,
        ms_pacman: bool = False,
        start_level: int = 1,
        step_mode: str = "tile",
        frames_per_step: int = 4,
        episodic_life: bool = False,
        end_on_level_clear: bool = False,
        max_ticks: int = 0,
        obs_mode: str = "grid",
        allow_noop: bool = False,
        record_dir: str | Path | None = None,
        render_mode: str | None = None,
    ) -> None:
        super().__init__()

        if step_mode not in ("frames", "tile", "intersection"):
            raise ValueError(f"unsupported step_mode {step_mode!r}")
        if obs_mode not in ("grid", "grid_only", "bfs"):
            raise ValueError(f"unsupported obs_mode {obs_mode!r}")

        self.binary = binary or default_binary_path()
        self.art = art
        self.ms_pacman = ms_pacman
        self.start_level = max(1, min(255, int(start_level)))
        self.step_mode = step_mode
        self.frames_per_step = max(1, int(frames_per_step))
        self.episodic_life = episodic_life
        self.end_on_level_clear = end_on_level_clear
        self.max_ticks = max(0, int(max_ticks))
        self.obs_mode = obs_mode
        self.allow_noop = allow_noop
        self.record_dir = Path(record_dir) if record_dir is not None else None
        self.render_mode = render_mode

        self.action_space = spaces.Discrete(5 if allow_noop else 4)
        if obs_mode == "grid":
            self.observation_space = spaces.Dict(
                {
                    "grid": spaces.Box(
                        low=0.0,
                        high=1.0,
                        shape=(GRID_CHANNELS, MAZE_HEIGHT, MAZE_WIDTH),
                        dtype=np.float32,
                    ),
                    "scalars": spaces.Box(
                        low=0.0,
                        high=1.0,
                        shape=(SCALAR_DIM,),
                        dtype=np.float32,
                    ),
                }
            )
        elif obs_mode == "grid_only":
            self.observation_space = spaces.Box(
                low=0.0,
                high=1.0,
                shape=(GRID_CHANNELS, MAZE_HEIGHT, MAZE_WIDTH),
                dtype=np.float32,
            )
        else:
            self.observation_space = spaces.Box(
                low=0.0,
                high=1.0,
                shape=(BFS_OBS_DIM,),
                dtype=np.float32,
            )

        self._proc: subprocess.Popen[str] | None = None
        self._mazes: dict[int, _MazeCache] = {}
        self.hello_info: dict[str, Any] = {}
        self._maze_index: int = 0
        self._dots = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)
        self._power = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)
        self._last_state: dict[str, Any] = {}
        self._last_legal = np.ones(4, dtype=np.bool_)
        self._episode_count = 0
        self._last_seed = 0

        self._start_process()

    def _start_process(self) -> None:
        cmd = [self.binary]
        if self.art:
            cmd.extend(["--art", str(self.art)])
        if self.ms_pacman:
            cmd.append("--ms")
        if self.start_level != 1:
            cmd.extend(["--level", str(self.start_level)])

        self._proc = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
        )

        hello = self._rpc({"cmd": "hello"})
        self.hello_info = hello
        self._mazes = {
            int(m["index"]): _MazeCache(int(m["index"]), int(m["dots"]), list(m["rows"]))
            for m in hello["mazes"]
        }

    def _rpc(self, payload: dict[str, Any]) -> dict[str, Any]:
        if self._proc is None or self._proc.stdin is None or self._proc.stdout is None:
            raise RuntimeError("pacfan-headless process is not running")

        try:
            self._proc.stdin.write(json.dumps(payload, separators=(",", ":")) + "\n")
            self._proc.stdin.flush()
            line = self._proc.stdout.readline()
        except BrokenPipeError as exc:
            raise RuntimeError(self._format_crash("broken pipe")) from exc

        if not line:
            raise RuntimeError(self._format_crash("unexpected EOF"))

        resp = json.loads(line)
        if not resp.get("ok", False):
            raise RuntimeError(f"pacfan-headless error: {resp.get('error', 'unknown error')}")
        return resp

    def _format_crash(self, reason: str) -> str:
        rc = None
        stderr_text = ""
        if self._proc is not None:
            rc = self._proc.poll()
            if rc is None:
                try:
                    rc = self._proc.wait(timeout=0.5)
                except subprocess.TimeoutExpired:
                    pass
            if self._proc.stderr is not None:
                try:
                    stderr_text = self._proc.stderr.read().strip()
                except Exception:
                    stderr_text = ""
        msg = f"pacfan-headless died ({reason}, exit_code={rc})"
        if stderr_text:
            msg += f": {stderr_text}"
        return msg

    def _select_maze(self, maze_index: int) -> None:
        self._maze_index = maze_index
        tmpl = self._mazes[maze_index]
        np.copyto(self._dots, tmpl.init_dots)
        np.copyto(self._power, tmpl.init_power)

    def reset(
        self,
        *,
        seed: int | None = None,
        options: dict[str, Any] | None = None,
    ) -> tuple[Any, dict[str, Any]]:
        super().reset(seed=seed)

        if seed is not None:
            game_seed = int(seed)
        else:
            game_seed = int(self.np_random.integers(0, 2**31 - 1))
        self._last_seed = game_seed
        self._episode_count += 1

        req: dict[str, Any] = {
            "cmd": "reset",
            "pacman": not self.ms_pacman,
            "level": self.start_level,
            "seed": game_seed,
            "episodic_life": self.episodic_life,
            "end_on_level_clear": self.end_on_level_clear,
            "max_ticks": self.max_ticks,
        }

        record_path: str | None = None
        meta: dict[str, Any] | None = None
        if options:
            if "record" in options and options["record"]:
                record_path = str(options["record"])
            if "meta" in options and options["meta"] is not None:
                meta = dict(options["meta"])
        if record_path is None and self.record_dir is not None:
            self.record_dir.mkdir(parents=True, exist_ok=True)
            record_path = str(self.record_dir / f"ep-{self._episode_count:04d}-seed-{game_seed}.pfr")
        if record_path:
            req["record"] = record_path
        if meta is not None:
            req["meta"] = meta

        resp = self._rpc(req)
        self._select_maze(int(resp["maze_index"]))
        self._last_state = resp["state"]
        self._update_legal_mask(self._last_state)

        obs = self._encode_obs(self._last_state)
        info = {
            "tick": int(resp["tick"]),
            "maze_index": self._maze_index,
            "seed": game_seed,
            "state": self._last_state,
            "action_mask": self.action_masks(),
            "maze_hash": self.hello_info.get("maze_hash", ""),
            "art": dict(self.hello_info.get("art", {})),
        }
        return obs, info

    def step(self, action: int) -> tuple[Any, float, bool, bool, dict[str, Any]]:
        act = int(action)
        req: dict[str, Any] = {
            "cmd": "step",
            "action": act,
            "mode": self.step_mode,
        }
        if self.step_mode == "frames":
            req["frames"] = self.frames_per_step

        resp = self._rpc(req)

        if "maze_index" in resp and resp["maze_index"] is not None:
            self._select_maze(int(resp["maze_index"]))
        else:
            for xy in resp.get("eaten", ()):
                x, y = int(xy[0]), int(xy[1])
                self._dots[y, x] = 0.0
                self._power[y, x] = 0.0

        self._last_state = resp["state"]
        self._update_legal_mask(self._last_state)

        obs = self._encode_obs(self._last_state)
        reward = float(resp["reward"])
        terminated = bool(resp["terminated"])
        truncated = bool(resp["truncated"])
        info = {
            "tick": int(resp["tick"]),
            "ticks": int(resp["ticks"]),
            "score_delta": int(resp["reward"]),
            "events": list(resp.get("events", ())),
            "eaten": resp.get("eaten", []),
            "maze_index": self._maze_index,
            "state": self._last_state,
            "action_mask": self.action_masks(),
        }
        return obs, reward, terminated, truncated, info

    def _update_legal_mask(self, state: dict[str, Any]) -> None:
        legal = state["player"]["legal"]
        self._last_legal[:] = legal[:4]
        if not np.any(self._last_legal):
            self._last_legal[:] = True

    def action_masks(self) -> np.ndarray:
        """Return boolean mask of legal actions for MaskablePPO."""
        if self.allow_noop:
            mask = np.empty(5, dtype=np.bool_)
            mask[:4] = self._last_legal
            mask[4] = True
            return mask
        return self._last_legal.copy()

    def _encode_obs(self, state: dict[str, Any]) -> Any:
        if self.obs_mode == "bfs":
            return self._encode_bfs(state)
        grid = self._encode_grid(state)
        if self.obs_mode == "grid_only":
            return grid
        scalars = self._encode_scalars(state)
        return {"grid": grid, "scalars": scalars}

    def _encode_grid(self, state: dict[str, Any]) -> np.ndarray:
        tmpl = self._mazes[self._maze_index]
        grid = np.zeros((GRID_CHANNELS, MAZE_HEIGHT, MAZE_WIDTH), dtype=np.float32)
        grid[0] = tmpl.walls
        grid[1] = self._dots
        grid[2] = self._power
        grid[3] = tmpl.tunnels

        px, py = _pixel_to_center_tile(state["player"]["x"], state["player"]["y"])
        grid[4, py, px] = 1.0

        fright_val = min(1.0, float(state["fright_frames"]) / MAX_FRIGHT_FRAMES)
        if fright_val <= 0.0:
            fright_val = 1.0

        for idx, g in enumerate(state["ghosts"]):
            gx, gy = _pixel_to_center_tile(g["x"], g["y"])
            gstate = g["state"]
            if gstate in ("scatter", "chase"):
                grid[5 + idx, gy, gx] = 1.0
            elif gstate == "frightened":
                grid[9, gy, gx] = fright_val
            elif gstate in ("eaten", "entering_house"):
                grid[10, gy, gx] = 1.0

        fruit = state["fruit"]
        if fruit.get("active", False):
            fx, fy = _pixel_to_center_tile(fruit["x"], fruit["y"])
            grid[11, fy, fx] = 1.0

        return grid

    def _encode_scalars(self, state: dict[str, Any]) -> np.ndarray:
        tmpl = self._mazes[self._maze_index]
        scalars = np.zeros(SCALAR_DIM, dtype=np.float32)
        scalars[0] = min(1.0, float(state["fright_frames"]) / MAX_FRIGHT_FRAMES)
        scalars[1] = 1.0 if state["wave"] == "chase" else 0.0
        scalars[2] = min(1.0, float(state["dots_remaining"]) / float(tmpl.dots))
        scalars[3] = min(1.0, max(0.0, float(state["lives"]) / 5.0))
        scalars[4] = min(1.0, float(state["ghosts_eaten"]) / 4.0)
        scalars[5] = 1.0 if state["fruit"].get("active", False) else 0.0
        pdir = int(state["player"]["dir"])
        if 0 <= pdir < 4:
            scalars[6 + pdir] = 1.0
        scalars[10] = min(1.0, float(state["level"]) / 21.0)
        return scalars

    def _bfs_from_player(self, px: int, py: int) -> tuple[np.ndarray, np.ndarray]:
        """Compute shortest-path distances and first-step directions from (px, py)."""
        passable = self._mazes[self._maze_index].passable
        dist = np.full((MAZE_HEIGHT, MAZE_WIDTH), -1, dtype=np.int16)
        first_dir = np.full((MAZE_HEIGHT, MAZE_WIDTH), -1, dtype=np.int8)
        dist[py, px] = 0

        q: deque[tuple[int, int, int]] = deque()
        for d, (dx, dy) in enumerate(DIR_DELTAS):
            nx = (px + dx) % MAZE_WIDTH
            ny = py + dy
            if 0 <= ny < MAZE_HEIGHT and passable[ny, nx]:
                dist[ny, nx] = 1
                first_dir[ny, nx] = d
                q.append((nx, ny, d))

        while q:
            cx, cy, fdir = q.popleft()
            nd = dist[cy, cx] + 1
            for dx, dy in DIR_DELTAS:
                nx = (cx + dx) % MAZE_WIDTH
                ny = cy + dy
                if 0 <= ny < MAZE_HEIGHT and passable[ny, nx] and dist[ny, nx] < 0:
                    dist[ny, nx] = nd
                    first_dir[ny, nx] = fdir
                    q.append((nx, ny, fdir))

        return dist, first_dir

    def _encode_bfs(self, state: dict[str, Any]) -> np.ndarray:
        tmpl = self._mazes[self._maze_index]
        px, py = _pixel_to_center_tile(state["player"]["x"], state["player"]["y"])
        dist, first_dir = self._bfs_from_player(px, py)

        feats = np.zeros(BFS_OBS_DIM, dtype=np.float32)

        def write_target(offset: int, d_val: int, fdir: int) -> None:
            if d_val < 0:
                feats[offset] = 1.0
                return
            feats[offset] = min(1.0, float(d_val) / MAX_BFS_DIST)
            if 0 <= fdir < 4:
                feats[offset + 1 + fdir] = 1.0

        def nearest_in_mask(mask: np.ndarray) -> tuple[int, int]:
            ys, xs = np.nonzero(mask)
            if len(ys) == 0:
                return -1, -1
            ds = dist[ys, xs]
            valid = ds >= 0
            if not np.any(valid):
                return -1, -1
            ys, xs, ds = ys[valid], xs[valid], ds[valid]
            best = int(np.argmin(ds))
            return int(ds[best]), int(first_dir[ys[best], xs[best]])

        # 0..4: nearest dot
        write_target(0, *nearest_in_mask(self._dots > 0.5))
        # 5..9: nearest power pellet
        write_target(5, *nearest_in_mask(self._power > 0.5))
        # 10..14: active fruit
        fruit = state["fruit"]
        if fruit.get("active", False):
            fx, fy = _pixel_to_center_tile(fruit["x"], fruit["y"])
            write_target(10, int(dist[fy, fx]), int(first_dir[fy, fx]))
        else:
            feats[10] = 1.0

        # 15..42: 4 ghosts x 7 features (dist, 4 first-dir one-hot, is_dangerous, is_frightened)
        for idx, g in enumerate(state["ghosts"]):
            off = 15 + idx * 7
            gx, gy = _pixel_to_center_tile(g["x"], g["y"])
            write_target(off, int(dist[gy, gx]), int(first_dir[gy, gx]))
            gstate = g["state"]
            feats[off + 5] = 1.0 if gstate in ("scatter", "chase") else 0.0
            feats[off + 6] = 1.0 if gstate == "frightened" else 0.0

        # 43..53: global scalars (legal[4], fright_norm, dots_ratio, wave_chase, player_dir[4])
        feats[43:47] = self._last_legal.astype(np.float32)
        feats[47] = min(1.0, float(state["fright_frames"]) / MAX_FRIGHT_FRAMES)
        feats[48] = min(1.0, float(state["dots_remaining"]) / float(tmpl.dots))
        feats[49] = 1.0 if state["wave"] == "chase" else 0.0
        pdir = int(state["player"]["dir"])
        if 0 <= pdir < 4:
            feats[50 + pdir] = 1.0

        return feats

    def render(self) -> str | None:
        if self.render_mode != "ansi" or not self._last_state:
            return None
        tmpl = self._mazes[self._maze_index]
        chars = []
        px, py = _pixel_to_center_tile(self._last_state["player"]["x"], self._last_state["player"]["y"])
        ghost_pos = {
            _pixel_to_center_tile(g["x"], g["y"]): g["state"]
            for g in self._last_state["ghosts"]
        }
        for y in range(MAZE_HEIGHT):
            row = []
            for x in range(MAZE_WIDTH):
                if (x, y) == (px, py):
                    row.append("P")
                elif (x, y) in ghost_pos:
                    row.append("f" if ghost_pos[(x, y)] == "frightened" else "G")
                elif tmpl.walls[y, x] > 0.5:
                    row.append("#")
                elif self._power[y, x] > 0.5:
                    row.append("O")
                elif self._dots[y, x] > 0.5:
                    row.append(".")
                elif tmpl.tunnels[y, x] > 0.5:
                    row.append("@")
                else:
                    row.append(" ")
            chars.append("".join(row))
        return "\n".join(chars)

    def close(self) -> None:
        if self._proc is None:
            return
        proc = self._proc
        self._proc = None
        try:
            if proc.poll() is None and proc.stdin is not None:
                proc.stdin.write('{"cmd":"close"}\n')
                proc.stdin.flush()
        except Exception:
            pass
        for stream in (proc.stdin, proc.stdout, proc.stderr):
            if stream is not None:
                try:
                    stream.close()
                except Exception:
                    pass
        try:
            proc.wait(timeout=2.0)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=2.0)

    def __enter__(self) -> PacFanEnv:
        return self

    def __exit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        self.close()

    def __del__(self) -> None:
        try:
            self.close()
        except Exception:
            pass


class RewardShapingWrapper(gym.Wrapper[Any, int, Any, int]):
    """Applies score scaling and optional death/level-clear/step shaping."""

    def __init__(
        self,
        env: gym.Env[Any, int],
        *,
        score_scale: float = 0.01,
        death_penalty: float = 0.0,
        level_clear_bonus: float = 0.0,
        step_penalty: float = 0.0,
    ) -> None:
        super().__init__(env)
        self.score_scale = float(score_scale)
        self.death_penalty = float(death_penalty)
        self.level_clear_bonus = float(level_clear_bonus)
        self.step_penalty = float(step_penalty)

    def step(self, action: int) -> tuple[Any, float, bool, bool, dict[str, Any]]:
        obs, raw_reward, terminated, truncated, info = self.env.step(action)
        events = info.get("events", ())
        shaped = float(raw_reward) * self.score_scale - self.step_penalty
        if "death" in events:
            shaped -= self.death_penalty
        if "level_clear" in events:
            shaped += self.level_clear_bonus
        return obs, shaped, terminated, truncated, info

    def action_masks(self) -> np.ndarray:
        return self.env.action_masks()  # type: ignore[attr-defined]
