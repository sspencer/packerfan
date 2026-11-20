"""Validation tests for PacFanEnv (Gymnasium check_env, determinism, AsyncVectorEnv)."""

from __future__ import annotations

from pathlib import Path

from gymnasium.utils.env_checker import check_env
from gymnasium.vector import AsyncVectorEnv
import numpy as np
import pytest

from pacfan_env import PacFanEnv, RewardShapingWrapper, default_binary_path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
FALLBACK_ART = str(REPO_ROOT / "assets" / "game.png")


def _require_binary() -> str:
    binary = default_binary_path()
    if not Path(binary).is_file():
        pytest.skip(f"pacfan-headless binary not found at {binary}; run `make build-headless`")
    return binary


@pytest.mark.parametrize("obs_mode", ["grid", "grid_only", "bfs"])
def test_gymnasium_check_env(obs_mode: str) -> None:
    binary = _require_binary()
    with PacFanEnv(
        binary=binary,
        art=FALLBACK_ART,
        obs_mode=obs_mode,
        step_mode="tile",
        max_ticks=500,
    ) as env:
        check_env(env, skip_render_check=True)


def test_seeded_trajectory_determinism() -> None:
    binary = _require_binary()
    actions = [1, 1, 2, 1, 3, 0, 0, 2, 1, 1, 3, 3, 0, 2, 1, 0] * 3

    def run_once(seed: int) -> list[tuple[int, float, bool, int, int]]:
        traj: list[tuple[int, float, bool, int, int]] = []
        with PacFanEnv(binary=binary, art=FALLBACK_ART, obs_mode="grid", step_mode="tile") as env:
            obs, info = env.reset(seed=seed)
            assert info["tick"] == 0
            for a in actions:
                obs, reward, term, trunc, info = env.step(a)
                st = info["state"]
                traj.append((info["tick"], reward, term or trunc, st["player"]["x"], st["player"]["y"]))
                if term or trunc:
                    break
        return traj

    traj_a = run_once(123)
    traj_b = run_once(123)
    assert traj_a == traj_b
    assert len(traj_a) > 10


def test_reward_wrapper_and_action_masks(tmp_path: Path) -> None:
    binary = _require_binary()
    with PacFanEnv(
        binary=binary,
        art=FALLBACK_ART,
        obs_mode="grid",
        step_mode="tile",
        record_dir=tmp_path,
        max_ticks=400,
    ) as base_env:
        env = RewardShapingWrapper(
            base_env,
            score_scale=0.01,
            death_penalty=5.0,
            level_clear_bonus=10.0,
            step_penalty=0.01,
        )
        _, info = env.reset(seed=7)
        mask = env.action_masks()
        assert mask.shape == (4,)
        assert mask.dtype == np.bool_
        assert bool(np.any(mask))

        for _ in range(30):
            legal = np.flatnonzero(env.action_masks())
            _, reward, term, trunc, info = env.step(int(legal[0]))
            assert isinstance(reward, float)
            if term or trunc:
                break

    pfrs = list(tmp_path.glob("*.pfr"))
    assert len(pfrs) == 1


def test_async_vector_env() -> None:
    binary = _require_binary()
    n_workers = 4

    def make_env(idx: int):
        def _thunk():
            return PacFanEnv(
                binary=binary,
                art=FALLBACK_ART,
                obs_mode="bfs",
                step_mode="tile",
                max_ticks=300,
            )

        return _thunk

    vec = AsyncVectorEnv([make_env(i) for i in range(n_workers)])
    try:
        obs, infos = vec.reset(seed=42)
        assert obs.shape == (n_workers, 54)
        for step in range(25):
            actions = np.array([(step + i) % 4 for i in range(n_workers)], dtype=np.int64)
            obs, rewards, terms, truncs, infos = vec.step(actions)
            assert obs.shape == (n_workers, 54)
            assert rewards.shape == (n_workers,)
    finally:
        vec.close()
