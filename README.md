# CheddarHead (`cheddarhead`)

Arcade dot-eating maze game with ghosts built in Go using [Ebitengine](https://github.com/hajimehoshi/ebiten) and the [Donburi](https://github.com/yohamta/donburi) Entity Component System (ECS). Supports both classic **Pac-Man** (default) and **Ms. Pac-Man** (`--ms`) mazes, sprites, death animations, and intermissions.

![Ms. Pac-Man gameplay using the placeholder example artwork](docs/example-screenshot.png)

*Gameplay with the placeholder example artwork included in the repository.*

See [Documentation](#documentation) for the architecture overview, asset format reference, and scripting API design.

## Prerequisites

- **Go 1.27+** (see [go.mod](go.mod))
- **Artwork**: The repository includes placeholder spritesheets `assets/game.png` and `assets/text.png`, so the game runs out of the box. They use simple shapes and original maze layouts, drawn at the same positions as the real artwork. Regenerate them with `go run ./tools/genexample assets/game.png assets/text.png`.
- **Original artwork (optional)**: For copyright reasons, the original artwork is not included in this repository. To use it, place compatible `assets/game-real.png` and `assets/text-real.png` files locally (originally sourced from [The Spriters Resource](https://www.spriters-resource.com/); see [assets/README.md](assets/README.md) for expected dimensions and layout) and rebuild. Each `*-real.png` file that exists is loaded in place of its placeholder.

> **Note**: `go test ./...` runs headlessly without opening a window. Spritesheet-backed validation checks use `assets/game-real.png` when present, otherwise `assets/game.png`.

## Running the Game

```sh
go run .
```

Pass CLI flags directly when running with `go run`, for example:

```sh
go run . --ms --zoom 4     # or: go run . -m -z 4
```

Long flags use a double dash (`--zoom 4` or `--zoom=4`); short flags use a single dash and can be combined (`-mz 4`).

### macOS App

```sh
make install-app   # build bin/CheddarHead.app and copy it to /Applications
```

This builds a universal (Apple silicon + Intel), ad-hoc signed `CheddarHead.app` you can launch from Finder, Launchpad, Spotlight, or the Dock. It starts with the default flags (classic Pac-Man, fullscreen); press **3** on the attract screen to switch to Ms. Pac-Man. Assets are embedded at build time, so place any local artwork or audio before building. `make app` builds the bundle without installing it, and `make icon` regenerates the icon source `packaging/macos/icon.png` from `tools/genicon`.

### CLI Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--fullscreen` | `true` | Run in fullscreen mode with a black background, zoomed to the maximum fit and centered (`--fullscreen=false` for windowed mode) |
| `-z`, `--zoom` | `0` | Integer display scale factor (`1..10`; `0` auto-fits to maximum scale in fullscreen or `3x` in windowed mode) |
| `-m`, `--ms` | `false` | Use Ms. Pac-Man mazes, sprites, death animation, and intermissions instead of classic Pac-Man |
| `-p`, `--players` | `1` | Default player count on the attract screen (`1` or `2`) |
| `--mute` | `false` | Disable arcade audio playback |
| `-r`, `--replay` | | Play back a recorded `.pfr` replay file, or every `.pfr` file in a directory (recursively, sorted) as a playlist, instead of starting a game |
| `-s`, `--replay-speed` | `1` | Starting replay playback speed multiplier (`0.25..16`) |
| `-a`, `--art` | | Override the game spritesheet PNG used in `--replay` mode (needed when the replay's mazes differ from the embedded artwork) |

### Controls

- **1 / 2** (on Attract screen): Start a 1-player or 2-player game
- **Enter / Space** (on Attract screen): Start with the selected player count
- **3** (on Attract screen): Switch between Pac-Man and Ms. Pac-Man (each keeps its own high score)
- **Arrow Keys** (`Up`, `Down`, `Left`, `Right`) or **WASD** (`W`, `A`, `S`, `D`): Steer Pac-Man / Ms. Pac-Man
- **Space**: Pause / resume gameplay
- **M**: Mute / unmute audio
- **F11** or **Alt+Enter**: Toggle fullscreen / windowed mode
- **Any Key** (on the Game Over screen, after a 1-second delay): Return to the Attract screen to pick 1P or 2P (or returns automatically after 3 seconds)

### Replay Controls

Replays (`.pfr` files) are recorded by the headless simulator, for example by the Python AI's evaluations. Watch one with:

```sh
go run . --replay path/to/game.pfr        # or a directory of .pfr files
```

- **Space**: Pause / resume playback
- **Right Arrow** (while paused): Step forward one tick (hold to repeat)
- **= / +**: Double the playback speed (up to 16x)
- **-**: Halve the playback speed (down to 0.25x)
- **N / P**: Next / previous file when playing a directory (wraps around)
- **Esc** or close the window: Quit

See [docs/headless.md](docs/headless.md#7-watching-and-verifying-replays) for the status bar, artwork selection, and verifying replays without a window.

### Useful Make Targets

```sh
make build      # Build the ./cheddarhead binary
make all        # Cross-compile all binaries into bin/ (plus bin/CheddarHead.app on macOS)
make app        # Build the macOS app bundle bin/CheddarHead.app
make install-app # Build the macOS app and copy it to /Applications
make run        # Build and launch ./cheddarhead (pass flags via ARGS="--ms --zoom 4")
make test       # Run all unit, ECS simulation, and scene tests
make test-race  # Run all tests with the race detector enabled
make cover      # Run tests and print statement coverage
make check      # Run go fmt, go vet, and race-checked tests
make fix        # Run go fix modernizers
make generate   # Run go generate ./... (regenerate stringer enum helpers)
make clean      # Remove build and coverage artifacts
make help       # List all available targets
```

---

## Documentation

| Document | Description |
| --- | --- |
| [docs/ecs.md](docs/ecs.md) | Architecture overview: project layout, Donburi entities and components, the world-scoped event bus, the scene state machine, the game loop and simulation pipeline, and the audio and rendering pipeline. |
| [assets/README.md](assets/README.md) | Asset reference: the files the game embeds, the `game.png` maze and sprite atlas layout, the `text.png` font layout, the placeholder example artwork, and the optional WAV audio files. |
| [docs/headless.md](docs/headless.md) | Headless simulator guide: how `cheddarhead-headless` works, the NDJSON API with a client example, recording, verifying and watching `.pfr` replays, and running the Python Gymnasium environment, baselines and PPO training. |
| [ml/README.md](ml/README.md) | Python AI quick start: building the headless server, setting up the virtual environment, running the tests and zero-ML baselines, and training and evaluating MaskablePPO agents. |
| [docs/machine.md](docs/machine.md) | Plan for running the game headlessly so a machine-learning agent can play far faster than real time, with deterministic input-log replays (`.pfr` files) for watching the agent improve in the normal game window. |
| [docs/scriptable.md](docs/scriptable.md) | Design proposal (not yet implemented) for a scripting and machine-learning API: operating modes, a wire protocol for observing state and sending controls, observation and reward design, an implementation plan, and an example Python Gymnasium client. |

---

## Background

- [The Pac-Man Dossier](https://www.gamedeveloper.com/design/the-pac-man-dossier)
- [Donburi ECS for Go](https://github.com/yohamta/donburi)
