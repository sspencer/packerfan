"""Benchmark where time goes in a PacFanEnv step: server round trip vs observation encoding.

For each step mode it measures, with a single environment and a legal-move policy:

* rpc     -- send a step request to pacfan-headless and parse the JSON reply (no encoding)
* encode  -- build the observation from a captured state, per obs mode (no server)
* env     -- the full PacFanEnv.step() for each obs mode

Run from the ml/ directory after `make build-headless`:

    python bench.py
    python bench.py --steps 20000 --modes tile --obs-modes bfs
"""

from __future__ import annotations

import argparse
import time
from pathlib import Path
from typing import Any

from pacfan_env import PacFanEnv, default_binary_path

REPO_ROOT = Path(__file__).resolve().parent.parent
# Use the placeholder art so results are comparable across machines.
DEFAULT_ART = str(REPO_ROOT / "assets" / "game.png")
# Untimed steps before each measurement so process startup and caches settle.
WARMUP_STEPS = 500


def _policy(legal: list[bool], i: int) -> int:
    """Same deterministic legal-move policy as the Go benchmarks."""
    start = i // 8
    for k in range(4):
        d = (start + k) % 4
        if legal[d]:
            return d
    return 0


def _make_env(args: argparse.Namespace, step_mode: str, obs_mode: str) -> PacFanEnv:
    return PacFanEnv(
        binary=args.binary,
        art=args.art,
        step_mode=step_mode,
        frames_per_step=args.frames,
        obs_mode=obs_mode,
    )


def bench_rpc(args: argparse.Namespace, step_mode: str) -> tuple[float, float, list[dict[str, Any]]]:
    """Return (seconds, ticks, sampled states) for raw step round trips."""
    states: list[dict[str, Any]] = []
    with _make_env(args, step_mode, "grid") as env:
        _, info = env.reset(seed=42)
        state = info["state"]
        extra = {"frames": args.frames} if step_mode == "frames" else {}
        ticks = 0
        t0 = 0.0
        for i in range(-WARMUP_STEPS, args.steps):
            if i == 0:
                ticks = 0
                t0 = time.perf_counter()
            resp = env._rpc({"cmd": "step", "action": _policy(state["player"]["legal"], i), "mode": step_mode, **extra})
            state = resp["state"]
            ticks += resp["ticks"]
            if i >= 0 and i % 50 == 0:
                states.append(state)
            if resp["terminated"] or resp["truncated"]:
                state = env._rpc({"cmd": "reset", "seed": 42})["state"]
        elapsed = time.perf_counter() - t0
    return elapsed, ticks, states


def bench_encode(args: argparse.Namespace, obs_mode: str, states: list[dict[str, Any]]) -> float:
    """Return seconds per observation encode, cycling over sampled states."""
    with _make_env(args, "tile", obs_mode) as env:
        env.reset(seed=42)
        n = args.steps
        t0 = time.perf_counter()
        for i in range(n):
            st = states[i % len(states)]
            env._update_legal_mask(st)
            env._encode_obs(st)
        return (time.perf_counter() - t0) / n


def bench_env(args: argparse.Namespace, step_mode: str, obs_mode: str) -> float:
    """Return seconds per full PacFanEnv.step()."""
    with _make_env(args, step_mode, obs_mode) as env:
        _, info = env.reset(seed=42)
        t0 = 0.0
        for i in range(-WARMUP_STEPS, args.steps):
            if i == 0:
                t0 = time.perf_counter()
            _, _, term, trunc, info = env.step(_policy(info["state"]["player"]["legal"], i))
            if term or trunc:
                _, info = env.reset(seed=42)
        return (time.perf_counter() - t0) / args.steps


def main() -> None:
    parser = argparse.ArgumentParser(description="Benchmark PacFanEnv: server round trip vs observation encoding.")
    parser.add_argument("--binary", default=default_binary_path())
    parser.add_argument("--art", default=DEFAULT_ART)
    parser.add_argument("--steps", type=int, default=5000, help="steps per measurement")
    parser.add_argument("--frames", type=int, default=4, help="ticks per step in frames mode")
    parser.add_argument("--modes", nargs="+", default=["frames", "tile", "intersection"],
                        choices=["frames", "tile", "intersection"])
    parser.add_argument("--obs-modes", nargs="+", default=["grid", "bfs"],
                        choices=["grid", "grid_only", "bfs"])
    args = parser.parse_args()

    print(f"steps per measurement: {args.steps}")
    print(f"{'step_mode':<13} {'part':<12} {'us/step':>9} {'steps/s':>10} {'ticks/step':>11}")
    for mode in args.modes:
        secs, ticks, states = bench_rpc(args, mode)
        per = secs / args.steps
        tps = ticks / args.steps
        print(f"{mode:<13} {'rpc':<12} {per * 1e6:9.1f} {1 / per:10.0f} {tps:11.2f}")
        for obs in args.obs_modes:
            enc = bench_encode(args, obs, states)
            print(f"{mode:<13} {'encode/' + obs:<12} {enc * 1e6:9.1f} {1 / enc:10.0f}")
        for obs in args.obs_modes:
            full = bench_env(args, mode, obs)
            print(f"{mode:<13} {'env/' + obs:<12} {full * 1e6:9.1f} {1 / full:10.0f} {tps:11.2f}")


if __name__ == "__main__":
    main()
