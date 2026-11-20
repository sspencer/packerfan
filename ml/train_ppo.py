"""Train a MaskablePPO agent on pacfan-headless with a small 31x28 spatial CNN."""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
from typing import Any

from gymnasium import spaces
from sb3_contrib import MaskablePPO
from sb3_contrib.common.wrappers import ActionMasker
from stable_baselines3.common.callbacks import BaseCallback
from stable_baselines3.common.monitor import Monitor
from stable_baselines3.common.torch_layers import BaseFeaturesExtractor
from stable_baselines3.common.vec_env import DummyVecEnv, SubprocVecEnv
import torch
from torch import nn

from eval import evaluate_checkpoint
from pacfan_env import PacFanEnv, RewardShapingWrapper, default_binary_path


def _git_commit() -> str:
    try:
        out = subprocess.check_output(
            ["git", "rev-parse", "--short", "HEAD"],
            stderr=subprocess.DEVNULL,
            text=True,
        )
        return out.strip()
    except Exception:
        return "unknown"


class PacFanCNNExtractor(BaseFeaturesExtractor):
    """Compact CNN + scalar fusion extractor tailored to (C, 31, 28) Pac-Man grids."""

    def __init__(self, observation_space: spaces.Dict, features_dim: int = 256) -> None:
        super().__init__(observation_space, features_dim)
        grid_space = observation_space.spaces["grid"]
        scalar_space = observation_space.spaces["scalars"]
        in_channels = int(grid_space.shape[0])
        scalar_dim = int(scalar_space.shape[0])

        self.cnn = nn.Sequential(
            nn.Conv2d(in_channels, 32, kernel_size=3, stride=1, padding=1),
            nn.ReLU(),
            nn.Conv2d(32, 64, kernel_size=3, stride=2, padding=1),
            nn.ReLU(),
            nn.Conv2d(64, 64, kernel_size=3, stride=2, padding=1),
            nn.ReLU(),
            nn.Flatten(),
        )
        with torch.no_grad():
            sample = torch.zeros((1, *grid_space.shape), dtype=torch.float32)
            cnn_out_dim = int(self.cnn(sample).shape[1])

        self.head = nn.Sequential(
            nn.Linear(cnn_out_dim + scalar_dim, features_dim),
            nn.ReLU(),
        )

    def forward(self, observations: dict[str, torch.Tensor]) -> torch.Tensor:
        grid_feat = self.cnn(observations["grid"])
        return self.head(torch.cat([grid_feat, observations["scalars"]], dim=1))


def _mask_fn(env: Any) -> Any:
    return env.unwrapped.action_masks()


def make_train_env(
    *,
    binary: str,
    art: str | None,
    pacman: bool,
    start_level: int,
    step_mode: str,
    frames_per_step: int,
    episodic_life: bool,
    end_on_level_clear: bool,
    max_ticks: int,
    obs_mode: str,
    score_scale: float,
    death_penalty: float,
    level_clear_bonus: float,
    step_penalty: float,
):
    def _init():
        base = PacFanEnv(
            binary=binary,
            art=art,
            pacman=pacman,
            start_level=start_level,
            step_mode=step_mode,
            frames_per_step=frames_per_step,
            episodic_life=episodic_life,
            end_on_level_clear=end_on_level_clear,
            max_ticks=max_ticks,
            obs_mode=obs_mode,
        )
        shaped = RewardShapingWrapper(
            base,
            score_scale=score_scale,
            death_penalty=death_penalty,
            level_clear_bonus=level_clear_bonus,
            step_penalty=step_penalty,
        )
        monitored = Monitor(shaped)
        return ActionMasker(monitored, _mask_fn)

    return _init


class RunLoggingAndEvalCallback(BaseCallback):
    """Logs episode metrics to episodes.jsonl, saves checkpoints, and runs fixed-seed eval."""

    def __init__(
        self,
        *,
        run_dir: Path,
        run_id: str,
        git_commit: str,
        eval_freq: int,
        eval_seeds: list[int],
        eval_kwargs: dict[str, Any],
    ) -> None:
        super().__init__()
        self.run_dir = run_dir
        self.run_id = run_id
        self.git_commit = git_commit
        self.eval_freq = max(1, eval_freq)
        self.eval_seeds = eval_seeds
        self.eval_kwargs = eval_kwargs

        self.checkpoints_dir = run_dir / "checkpoints"
        self.eval_root = run_dir / "eval"
        self.best_dir = run_dir / "best"
        self.episodes_path = run_dir / "episodes.jsonl"

        self.checkpoints_dir.mkdir(parents=True, exist_ok=True)
        self.eval_root.mkdir(parents=True, exist_ok=True)
        self.best_dir.mkdir(parents=True, exist_ok=True)

        self.best_mean_score = float("-inf")
        self.next_eval_step = self.eval_freq
        self.last_eval_step = -1
        self.episode_counter = 0

    def _on_step(self) -> bool:
        infos = self.locals.get("infos", [])
        with self.episodes_path.open("a", encoding="utf-8") as fh:
            for info in infos:
                ep = info.get("episode")
                if ep is None:
                    continue
                self.episode_counter += 1
                state = info.get("state", {})
                record = {
                    "episode": self.episode_counter,
                    "timesteps": int(self.num_timesteps),
                    "reward": float(ep["r"]),
                    "length": int(ep["l"]),
                    "score": int(state.get("score", 0)),
                    "level": int(state.get("level", 1)),
                    "tick": int(info.get("tick", 0)),
                }
                fh.write(json.dumps(record) + "\n")

        if self.num_timesteps >= self.next_eval_step:
            self._checkpoint_and_eval()
            while self.next_eval_step <= self.num_timesteps:
                self.next_eval_step += self.eval_freq

        return True

    def _checkpoint_and_eval(self) -> None:
        if self.num_timesteps == self.last_eval_step:
            return
        self.last_eval_step = self.num_timesteps
        ckpt_name = f"ckpt-{self.num_timesteps:08d}"
        ckpt_path = self.checkpoints_dir / f"{ckpt_name}.zip"
        self.model.save(str(ckpt_path))

        eval_dir = self.eval_root / ckpt_name
        summary = evaluate_checkpoint(
            self.model,  # type: ignore[arg-type]
            seeds=self.eval_seeds,
            record_dir=eval_dir,
            meta_base={
                "run_id": self.run_id,
                "checkpoint": ckpt_name,
                "git_commit": self.git_commit,
            },
            **self.eval_kwargs,
        )
        mean_score = float(summary["mean_score"])
        print(
            f"[{ckpt_name}] eval mean_score={mean_score:.1f} "
            f"max={summary['max_score']} mean_levels={summary['mean_levels_cleared']:.2f}"
        )
        self.logger.record("eval/mean_score", mean_score)
        self.logger.record("eval/mean_levels_cleared", float(summary["mean_levels_cleared"]))
        self.logger.record("eval/mean_ticks", float(summary["mean_ticks"]))

        if mean_score > self.best_mean_score:
            self.best_mean_score = mean_score
            self.model.save(str(self.best_dir / "model.zip"))
            (self.best_dir / "metrics.json").write_text(
                json.dumps({"checkpoint": ckpt_name, **summary}, indent=2) + "\n",
                encoding="utf-8",
            )


def main() -> None:
    parser = argparse.ArgumentParser(description="Train MaskablePPO on headless pacfan.")
    parser.add_argument("--binary", default=default_binary_path())
    parser.add_argument("--art", default=None)
    parser.add_argument("--pacman", action="store_true")
    parser.add_argument("--start-level", type=int, default=1)
    parser.add_argument("--step-mode", default="tile", choices=["frames", "tile", "intersection"])
    parser.add_argument("--frames-per-step", type=int, default=4)
    parser.add_argument("--obs-mode", default="grid", choices=["grid", "bfs"])
    parser.add_argument("--episodic-life", action=argparse.BooleanOptionalAction, default=True)
    parser.add_argument("--end-on-level-clear", action="store_true")
    parser.add_argument("--max-ticks", type=int, default=15000)

    parser.add_argument("--score-scale", type=float, default=0.01)
    parser.add_argument("--death-penalty", type=float, default=5.0)
    parser.add_argument("--level-clear-bonus", type=float, default=10.0)
    parser.add_argument("--step-penalty", type=float, default=0.01)

    parser.add_argument("--total-timesteps", type=int, default=100_000)
    parser.add_argument("--n-envs", type=int, default=4)
    parser.add_argument("--n-steps", type=int, default=256)
    parser.add_argument("--batch-size", type=int, default=128)
    parser.add_argument("--learning-rate", type=float, default=3e-4)
    parser.add_argument("--gamma", type=float, default=0.99)
    parser.add_argument("--ent-coef", type=float, default=0.01)
    parser.add_argument("--seed", type=int, default=42)

    parser.add_argument("--eval-freq", type=int, default=25_000)
    parser.add_argument("--eval-episodes", type=int, default=5)
    parser.add_argument("--eval-seed-start", type=int, default=1000)
    parser.add_argument("--run-dir", type=Path, default=None)
    args = parser.parse_args()

    ts = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
    run_dir = args.run_dir or (Path(__file__).resolve().parent.parent / "runs" / f"ppo-{ts}")
    run_dir.mkdir(parents=True, exist_ok=True)
    run_id = run_dir.name
    commit = _git_commit()

    # Probe hello metadata once for run.json.
    with PacFanEnv(binary=args.binary, art=args.art, pacman=args.pacman, obs_mode=args.obs_mode) as probe:
        hello = probe.hello_info

    run_meta = {
        "run_id": run_id,
        "created_utc": datetime.now(timezone.utc).isoformat(),
        "git_commit": commit,
        "art": hello.get("art", {}),
        "maze_hash": hello.get("maze_hash", ""),
        "env": {
            "pacman": args.pacman,
            "start_level": args.start_level,
            "step_mode": args.step_mode,
            "frames_per_step": args.frames_per_step,
            "obs_mode": args.obs_mode,
            "episodic_life": args.episodic_life,
            "end_on_level_clear": args.end_on_level_clear,
            "max_ticks": args.max_ticks,
        },
        "reward": {
            "score_scale": args.score_scale,
            "death_penalty": args.death_penalty,
            "level_clear_bonus": args.level_clear_bonus,
            "step_penalty": args.step_penalty,
        },
        "hyperparameters": {
            "total_timesteps": args.total_timesteps,
            "n_envs": args.n_envs,
            "n_steps": args.n_steps,
            "batch_size": args.batch_size,
            "learning_rate": args.learning_rate,
            "gamma": args.gamma,
            "ent_coef": args.ent_coef,
            "seed": args.seed,
        },
    }
    (run_dir / "run.json").write_text(json.dumps(run_meta, indent=2) + "\n", encoding="utf-8")

    env_fns = [
        make_train_env(
            binary=args.binary,
            art=args.art,
            pacman=args.pacman,
            start_level=args.start_level,
            step_mode=args.step_mode,
            frames_per_step=args.frames_per_step,
            episodic_life=args.episodic_life,
            end_on_level_clear=args.end_on_level_clear,
            max_ticks=args.max_ticks,
            obs_mode=args.obs_mode,
            score_scale=args.score_scale,
            death_penalty=args.death_penalty,
            level_clear_bonus=args.level_clear_bonus,
            step_penalty=args.step_penalty,
        )
        for _ in range(args.n_envs)
    ]
    vec_env = SubprocVecEnv(env_fns) if args.n_envs > 1 else DummyVecEnv(env_fns)

    if args.obs_mode == "grid":
        policy = "MultiInputPolicy"
        policy_kwargs: dict[str, Any] = {
            "features_extractor_class": PacFanCNNExtractor,
            "features_extractor_kwargs": {"features_dim": 256},
        }
    else:
        policy = "MlpPolicy"
        policy_kwargs = {"net_arch": [128, 128]}

    eval_seeds = [args.eval_seed_start + i for i in range(args.eval_episodes)]
    callback = RunLoggingAndEvalCallback(
        run_dir=run_dir,
        run_id=run_id,
        git_commit=commit,
        eval_freq=args.eval_freq,
        eval_seeds=eval_seeds,
        eval_kwargs={
            "binary": args.binary,
            "art": args.art,
            "pacman": args.pacman,
            "start_level": args.start_level,
            "step_mode": args.step_mode,
            "frames_per_step": args.frames_per_step,
            "obs_mode": args.obs_mode,
            "max_ticks": args.max_ticks,
        },
    )

    try:
        model = MaskablePPO(
            policy,
            vec_env,
            n_steps=args.n_steps,
            batch_size=args.batch_size,
            learning_rate=args.learning_rate,
            gamma=args.gamma,
            ent_coef=args.ent_coef,
            policy_kwargs=policy_kwargs,
            tensorboard_log=str(run_dir / "tb"),
            seed=args.seed,
            verbose=1,
        )
        model.learn(total_timesteps=args.total_timesteps, callback=callback)
        callback._checkpoint_and_eval()
    finally:
        vec_env.close()


if __name__ == "__main__":
    main()
