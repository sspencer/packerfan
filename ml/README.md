# pacfan-ml

Python Gymnasium environment, zero-ML baselines, and `MaskablePPO` training scripts for the headless `pacfan-headless` Pac-Man / Ms. Pac-Man simulator.

## 1. Build the headless Go server

From the repository root:

```bash
make build-headless
```

This builds `./pacfan-headless` (pure Go, zero Raylib/audio/scene dependencies).

## 2. Create a virtual environment

```bash
cd ml
python3 -m venv .venv
source .venv/bin/activate       # bash/zsh (or: source .venv/bin/activate.fish for fish)
pip install -e ".[dev]"
```

To train PPO policies with PyTorch + Stable-Baselines3:

```bash
pip install -e ".[dev,train]"
```

## 3. Sanity-check the Gymnasium environment

```bash
pytest
```

This runs Gymnasium's `check_env`, verifies seeded trajectory determinism, tests both `grid` and `bfs` observation modes, and exercises `AsyncVectorEnv` with multiple parallel workers.

## 4. Run zero-ML baselines

```bash
python baselines.py --episodes 5 --step-mode tile
python baselines.py --benchmark-workers 1 4 8
```

## 5. Train and evaluate MaskablePPO

```bash
python train_ppo.py \
  --total-timesteps 100000 \
  --n-envs 4 \
  --step-mode tile \
  --eval-freq 25000 \
  --run-dir ../runs/ppo-demo
```

Evaluate a saved checkpoint and optionally record `.pfr` replays:

```bash
python eval.py ../runs/ppo-demo/best/model.zip --episodes 10 --record-dir ../runs/ppo-demo/eval-manual
```

Verify any recorded `.pfr` replay headlessly:

```bash
../pacfan-headless -verify ../runs/ppo-demo/eval/ckpt-00025000/seed-1000.pfr
```
