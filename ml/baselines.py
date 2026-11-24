"""Zero-ML baselines (RandomLegalAgent and GreedyBFSAgent) and worker throughput benchmark."""

from __future__ import annotations

import argparse
from collections import deque
import time
from typing import Any

from gymnasium.vector import AsyncVectorEnv
import numpy as np

from cheddarhead_env import (
    DIR_DELTAS,
    MAZE_HEIGHT,
    MAZE_WIDTH,
    CheddarHeadEnv,
    _pixel_to_center_tile,
    default_binary_path,
)


class RandomLegalAgent:
    """Uniformly samples among legal actions from env.action_masks()."""

    def __init__(self, seed: int = 0) -> None:
        self.rng = np.random.default_rng(seed)

    def act(self, env: CheddarHeadEnv, info: dict[str, Any]) -> int:
        mask = env.action_masks()
        legal = np.flatnonzero(mask[:4])
        if len(legal) == 0:
            return 0
        return int(self.rng.choice(legal))


class GreedyBFSAgent:
    """Shortest-path BFS agent targeting frightened ghosts, fruit, and dots while avoiding danger."""

    def __init__(self, avoid_dist: int = 3) -> None:
        self.avoid_dist = avoid_dist

    def act(self, env: CheddarHeadEnv, info: dict[str, Any]) -> int:
        state = info["state"]
        tmpl = env._mazes[env._maze_index]
        passable = tmpl.passable
        dots = env._dots
        power = env._power

        px, py = _pixel_to_center_tile(state["player"]["x"], state["player"]["y"])

        dangers: list[tuple[int, int]] = []
        edible: set[tuple[int, int]] = set()
        for g in state["ghosts"]:
            gx, gy = _pixel_to_center_tile(g["x"], g["y"])
            if g["state"] in ("scatter", "chase"):
                dangers.append((gx, gy))
            elif g["state"] == "frightened":
                edible.add((gx, gy))

        fruit_pos: tuple[int, int] | None = None
        if state["fruit"].get("active", False):
            fruit_pos = _pixel_to_center_tile(state["fruit"]["x"], state["fruit"]["y"])

        def min_danger_dist(x: int, y: int) -> int:
            if not dangers:
                return 999
            return min(abs(gx - x) + abs(gy - y) for gx, gy in dangers)

        def is_danger(x: int, y: int) -> bool:
            return min_danger_dist(x, y) <= self.avoid_dist

        visited = np.zeros((MAZE_HEIGHT, MAZE_WIDTH), dtype=np.bool_)
        visited[py, px] = True
        q: deque[tuple[int, int, int]] = deque()

        best_escape_dir = 0
        best_escape_dist = -1

        for d, (dx, dy) in enumerate(DIR_DELTAS):
            nx = (px + dx) % MAZE_WIDTH
            ny = py + dy
            if not (0 <= ny < MAZE_HEIGHT and passable[ny, nx]):
                continue
            gd = min_danger_dist(nx, ny)
            if gd > best_escape_dist:
                best_escape_dist = gd
                best_escape_dir = d
            if is_danger(nx, ny):
                continue
            visited[ny, nx] = True
            q.append((nx, ny, d))

        while q:
            cx, cy, fdir = q.popleft()
            if (cx, cy) in edible:
                return fdir
            if fruit_pos is not None and (cx, cy) == fruit_pos:
                return fdir
            if dots[cy, cx] > 0.5 or power[cy, cx] > 0.5:
                return fdir
            for dx, dy in DIR_DELTAS:
                nx = (cx + dx) % MAZE_WIDTH
                ny = cy + dy
                if 0 <= ny < MAZE_HEIGHT and passable[ny, nx] and not visited[ny, nx] and not is_danger(nx, ny):
                    visited[ny, nx] = True
                    q.append((nx, ny, fdir))

        return best_escape_dir


def evaluate_agent(
    name: str,
    agent: Any,
    *,
    episodes: int,
    seeds: list[int],
    binary: str,
    art: str | None,
    ms_pacman: bool,
    step_mode: str,
    max_ticks: int,
) -> dict[str, float]:
    scores: list[int] = []
    levels_cleared: list[int] = []
    ticks_list: list[int] = []
    total_steps = 0

    t0 = time.perf_counter()
    with CheddarHeadEnv(
        binary=binary,
        art=art,
        ms_pacman=ms_pacman,
        step_mode=step_mode,
        max_ticks=max_ticks,
        obs_mode="grid",
    ) as env:
        for ep in range(episodes):
            seed = seeds[ep % len(seeds)]
            _, info = env.reset(seed=seed)
            cleared = 0
            while True:
                action = agent.act(env, info)
                _, _, term, trunc, info = env.step(action)
                total_steps += 1
                if "level_clear" in info["events"]:
                    cleared += 1
                if term or trunc:
                    break
            scores.append(int(info["state"]["score"]))
            levels_cleared.append(cleared)
            ticks_list.append(int(info["tick"]))
    elapsed = max(1e-9, time.perf_counter() - t0)

    stats = {
        "mean_score": float(np.mean(scores)),
        "max_score": float(np.max(scores)),
        "mean_levels_cleared": float(np.mean(levels_cleared)),
        "mean_ticks": float(np.mean(ticks_list)),
        "steps_per_sec": float(total_steps) / elapsed,
    }
    print(
        f"[{name}] episodes={episodes} "
        f"mean_score={stats['mean_score']:.1f} max_score={stats['max_score']:.0f} "
        f"mean_levels={stats['mean_levels_cleared']:.2f} mean_ticks={stats['mean_ticks']:.1f} "
        f"steps/s={stats['steps_per_sec']:.1f}"
    )
    return stats


def benchmark_workers(
    worker_counts: list[int],
    *,
    steps_per_worker: int,
    binary: str,
    art: str | None,
    ms_pacman: bool,
    step_mode: str,
    obs_mode: str,
) -> None:
    print(f"Benchmarking AsyncVectorEnv throughput (obs={obs_mode}, step_mode={step_mode}):")
    for n in worker_counts:
        if n == 1:
            t0 = time.perf_counter()
            with CheddarHeadEnv(binary=binary, art=art, ms_pacman=ms_pacman, step_mode=step_mode, obs_mode=obs_mode) as env:
                env.reset(seed=7)
                for s in range(steps_per_worker):
                    _, _, term, trunc, _ = env.step(s % 4)
                    if term or trunc:
                        env.reset()
            elapsed = max(1e-9, time.perf_counter() - t0)
            print(f"  workers=1  steps={steps_per_worker}  elapsed={elapsed:.3f}s  env_steps/s={steps_per_worker / elapsed:.1f}")
            continue

        def make_thunk():
            return CheddarHeadEnv(binary=binary, art=art, ms_pacman=ms_pacman, step_mode=step_mode, obs_mode=obs_mode)

        vec = AsyncVectorEnv([make_thunk for _ in range(n)])
        try:
            vec.reset(seed=7)
            t0 = time.perf_counter()
            for s in range(steps_per_worker):
                actions = np.full(n, s % 4, dtype=np.int64)
                vec.step(actions)
            elapsed = max(1e-9, time.perf_counter() - t0)
        finally:
            vec.close()
        total = n * steps_per_worker
        print(f"  workers={n:<2d} steps={total}  elapsed={elapsed:.3f}s  env_steps/s={total / elapsed:.1f}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Run zero-ML cheddarhead baselines and throughput benchmarks.")
    parser.add_argument("--binary", default=default_binary_path())
    parser.add_argument("--art", default=None)
    parser.add_argument("--ms", action="store_true", help="Ms. Pac-Man instead of classic Pac-Man")
    parser.add_argument("--episodes", type=int, default=5)
    parser.add_argument("--step-mode", default="tile", choices=["frames", "tile", "intersection"])
    parser.add_argument("--obs-mode", default="grid", choices=["grid", "grid_only", "bfs"])
    parser.add_argument("--max-ticks", type=int, default=12000)
    parser.add_argument("--avoid-dist", type=int, default=3)
    parser.add_argument("--benchmark-workers", type=int, nargs="*", default=None)
    parser.add_argument("--benchmark-steps", type=int, default=300)
    args = parser.parse_args()

    seeds = [1000 + i for i in range(max(1, args.episodes))]
    evaluate_agent(
        "RandomLegalAgent",
        RandomLegalAgent(seed=42),
        episodes=args.episodes,
        seeds=seeds,
        binary=args.binary,
        art=args.art,
        ms_pacman=args.ms,
        step_mode=args.step_mode,
        max_ticks=args.max_ticks,
    )
    evaluate_agent(
        "GreedyBFSAgent",
        GreedyBFSAgent(avoid_dist=args.avoid_dist),
        episodes=args.episodes,
        seeds=seeds,
        binary=args.binary,
        art=args.art,
        ms_pacman=args.ms,
        step_mode=args.step_mode,
        max_ticks=args.max_ticks,
    )

    if args.benchmark_workers:
        benchmark_workers(
            args.benchmark_workers,
            steps_per_worker=args.benchmark_steps,
            binary=args.binary,
            art=args.art,
            ms_pacman=args.ms,
            step_mode=args.step_mode,
            obs_mode=args.obs_mode,
        )


if __name__ == "__main__":
    main()
