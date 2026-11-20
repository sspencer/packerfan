# Packer Fan (`pacfan`)

Arcade dot-eating maze game with ghosts built in Go using [Raylib](https://github.com/gen2brain/raylib-go) and the [Donburi](https://github.com/yohamta/donburi) Entity Component System (ECS). Supports both **Ms. Pac-Man** (default) and classic **Pac-Man** (`-pacman`) mazes, sprites, death animations, and intermissions.

See [docs/ecs.md](docs/ecs.md) for the ECS, scene state machine, and rendering architecture overview.

## Prerequisites

- **Go 1.27+** (see [go.mod](go.mod))
- **Local artwork assets**: For copyright reasons, no artwork is included in this repository. To run the game, place compatible `assets/game.png` and `assets/text.png` files locally (originally sourced from [The Spriters Resource](https://www.spriters-resource.com/); see [assets/README.md](assets/README.md) for expected dimensions and layout).

> **Note**: `go test ./...` runs headlessly without Raylib or artwork, skipping only spritesheet-backed validation checks when `assets/game.png` is absent.

## Running the Game

```sh
go run .
```

Pass CLI flags directly when running with `go run`, for example:

```sh
go run . -pacman -zoom 4
```

### CLI Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-zoom` | `3` | Integer display scale factor (clamped to `1..10`) |
| `-pacman` | `false` | Use classic Pac-Man maze, sprites, death animation, and intermissions instead of Ms. Pac-Man |
| `-players` | `1` | Default player count on the attract screen (`1` or `2`) |

### Controls

- **1 / 2** (on Attract / Game Over screen): Start a 1-player or 2-player game
- **Enter / Space** (on Attract screen): Start with the selected player count
- **Arrow Keys** (`Up`, `Down`, `Left`, `Right`) or **WASD** (`W`, `A`, `S`, `D`): Steer Pac-Man / Ms. Pac-Man
- **Space**: Pause / resume gameplay
- **Any Key** (on the Game Over screen, after a 1-second delay): Start a new game from Level 1 (preserving High Score)

### Useful Make Targets

```sh
make build      # Build the ./pacfan binary
make run        # Build and launch ./pacfan (pass flags via ARGS="-pacman -zoom 4")
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

## Background

- [The Pac-Man Dossier](https://www.gamedeveloper.com/design/the-pac-man-dossier)
- [Donburi ECS for Go](https://github.com/yohamta/donburi)
