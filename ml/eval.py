"""Evaluate a saved MaskablePPO checkpoint on fixed seeds and optionally record .pfr replays."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

import numpy as np
from sb3_contrib import MaskablePPO

from cheddarhead_env import CheddarHeadEnv, default_binary_path


def evaluate_checkpoint(
    model: MaskablePPO,
    *,
    seeds: list[int],
    binary: str,
    art: str | None = None,
    ms_pacman: bool = False,
    start_level: int = 1,
    step_mode: str = "tile",
    frames_per_step: int = 4,
    obs_mode: str = "grid",
    max_ticks: int = 20000,
    record_dir: Path | None = None,
    meta_base: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Run deterministic full-game episodes across fixed seeds and return summary metrics."""
    if record_dir is not None:
        record_dir.mkdir(parents=True, exist_ok=True)

    episodes: list[dict[str, Any]] = []
    with CheddarHeadEnv(
        binary=binary,
        art=art,
        ms_pacman=ms_pacman,
        start_level=start_level,
        step_mode=step_mode,
        frames_per_step=frames_per_step,
        episodic_life=False,
        end_on_level_clear=False,
        max_ticks=max_ticks,
        obs_mode=obs_mode,
    ) as env:
        for idx, seed in enumerate(seeds):
            reset_opts: dict[str, Any] = {}
            replay_path: str | None = None
            if record_dir is not None:
                replay_path = str(record_dir / f"seed-{seed}.pfr")
                meta = dict(meta_base or {})
                meta.update({"episode": idx, "seed": seed})
                reset_opts = {"record": replay_path, "meta": meta}

            obs, info = env.reset(seed=seed, options=reset_opts)
            levels_cleared = 0
            steps = 0
            while True:
                mask = env.action_masks()
                action, _ = model.predict(obs, action_masks=mask, deterministic=True)
                obs, _, terminated, truncated, info = env.step(int(action))
                steps += 1
                if "level_clear" in info["events"]:
                    levels_cleared += 1
                if terminated or truncated:
                    break

            st = info["state"]
            episodes.append(
                {
                    "seed": seed,
                    "score": int(st["score"]),
                    "level": int(st["level"]),
                    "levels_cleared": levels_cleared,
                    "ticks": int(info["tick"]),
                    "steps": steps,
                    "replay": replay_path,
                }
            )

    scores = [ep["score"] for ep in episodes]
    ticks = [ep["ticks"] for ep in episodes]
    levels = [ep["levels_cleared"] for ep in episodes]

    summary = {
        "n_episodes": len(episodes),
        "mean_score": float(np.mean(scores)) if scores else 0.0,
        "std_score": float(np.std(scores)) if scores else 0.0,
        "max_score": int(np.max(scores)) if scores else 0,
        "min_score": int(np.min(scores)) if scores else 0,
        "mean_levels_cleared": float(np.mean(levels)) if levels else 0.0,
        "mean_ticks": float(np.mean(ticks)) if ticks else 0.0,
        "episodes": episodes,
    }
    if record_dir is not None:
        (record_dir / "metrics.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    return summary


def main() -> None:
    parser = argparse.ArgumentParser(description="Evaluate a MaskablePPO checkpoint on cheddarhead-headless.")
    parser.add_argument("checkpoint", type=Path, help="Path to saved .zip checkpoint")
    parser.add_argument("--binary", default=default_binary_path())
    parser.add_argument("--art", default=None)
    parser.add_argument("--ms", action="store_true", help="Ms. Pac-Man instead of classic Pac-Man")
    parser.add_argument("--start-level", type=int, default=1)
    parser.add_argument("--step-mode", default="tile", choices=["frames", "tile", "intersection"])
    parser.add_argument("--frames-per-step", type=int, default=4)
    parser.add_argument("--obs-mode", default="grid", choices=["grid", "grid_only", "bfs"])
    parser.add_argument("--max-ticks", type=int, default=20000)
    parser.add_argument("--episodes", type=int, default=5)
    parser.add_argument("--seed-start", type=int, default=1000)
    parser.add_argument("--record-dir", type=Path, default=None)
    args = parser.parse_args()

    model = MaskablePPO.load(str(args.checkpoint))
    seeds = [args.seed_start + i for i in range(args.episodes)]
    summary = evaluate_checkpoint(
        model,
        seeds=seeds,
        binary=args.binary,
        art=args.art,
        ms_pacman=args.ms,
        start_level=args.start_level,
        step_mode=args.step_mode,
        frames_per_step=args.frames_per_step,
        obs_mode=args.obs_mode,
        max_ticks=args.max_ticks,
        record_dir=args.record_dir,
        meta_base={"checkpoint": args.checkpoint.name},
    )
    print(
        f"Eval {args.checkpoint.name} ({summary['n_episodes']} eps): "
        f"mean_score={summary['mean_score']:.1f} ± {summary['std_score']:.1f} "
        f"max={summary['max_score']} min={summary['min_score']} "
        f"mean_levels={summary['mean_levels_cleared']:.2f} "
        f"mean_ticks={summary['mean_ticks']:.1f}"
    )


if __name__ == "__main__":
    main()
