# Assets

For copyright reasons, no sprite artwork, bitmap fonts, or sound effects are included in this repository (they are excluded via `.gitignore`).

At build time, `main.go` embeds the `assets/` directory with `//go:embed assets`, and `internal/render/art.go` loads `assets/game.png`, `assets/text.png`, and `assets/chroma_key.fs` on startup. To build and run the game locally, you must supply compatible `game.png` and `text.png` files.

> **Headless tests**: Running `go test ./...` does not require these PNG files. Tests that validate sprite coordinates or maze parsing against `assets/game.png` automatically skip when the file is absent.

---

## Tracked Assets

- **`chroma_key.fs`**: GLSL 330 fragment shader used by `internal/render` to discard near-black pixels (`keyColor = (0, 0, 0)`, `threshold = 0.05`) when drawing sprites and bitmap-font overlays.

---

## Required Local Assets (Not Included)

### `assets/game.png` (`680×1240` PNG)

Main maze atlas and spritesheet.

**Origin & Customization**:
- Originally based on the `648×1488` Ms. Pac-Man spritesheet from [The Spriters Resource](https://www.spriters-resource.com/) (circa November 2025).
- Customized to **`680×1240`** by:
  1. Removing the bottom two Ms. Pac-Man mazes (color-swapped duplicates of mazes 3 and 4).
  2. Appending the classic Pac-Man maze and sprites (also from The Spriters Resource) as maze index `4` (`y = 992`).
  3. Adjusting the classic Pac-Man directional sprite rows so each direction includes a closed-mouth frame.

**Expected Atlas Layout** (as read by `internal/component/maze.go`, `internal/render/renderer.go`, and `internal/scene/intermission.go`):

- **Mazes (`224×248` px = `28×31` tiles each, 5 rows at `y = index * 248`)**:
  - Indices `0..3` (`y = 0, 248, 496, 744`): Ms. Pac-Man mazes 1–4.
  - Index `4` (`y = 992`): Classic Pac-Man maze (`-pacman` mode).
  - Left column (`x = 0`): Mazes with dots, power pellets, and ghost-house doors baked in. `MazeData.LoadLevel` samples 8×8 pixel bitmasks here to classify each tile (`Empty`, `Dot`, `Power`, `Door`, `Wall`), and `Renderer.drawMaze` extracts individual 8×8 dot and power-pellet tiles from `x = 8`.
  - Middle column (`x = 228`): Matching empty maze backgrounds (no dots/pellets) rendered behind gameplay.
- **Sprites (`x >= 456`)**:
  - **Ms. Pac-Man**: `16×16`, 3 frames per direction at `x = 456`, `y = 0, 16, 32, 48` (Right, Left, Up, Down).
  - **Ghosts**: `16×16`, 2 frames per direction at `x = 456, 488, 520, 552` (Right, Left, Up, Down) on rows `y = 64` (Blinky), `80` (Pinky), `96` (Inky), and `112` (Clyde).
  - **Frightened & Eaten Ghosts**: Blue frightened at `x = 584, y = 64` (2 frames), white flashing at `x = 616, y = 64` (2 frames), and directional eyes at `x = 584, 600, 616, 632`, `y = 80` (Right, Left, Up, Down).
  - **Ghost-Eaten Bonus Scores**: `16×16` sprites for `200`, `400`, `800`, `1600` at `x = 456, 472, 488, 504`, `y = 128`.
  - **Intermissions**: Ms. Pac-Man cutscene sprites at `y = 144, 160` and 32×32 clapperboard frames at `x = 456, y = 208`; classic Pac-Man 32×32 giant Pac-Man frames at `x = 504, y = 1008`.
  - **Classic Pac-Man**: `16×16`, 3 frames per direction at `x = 456`, `y = 992, 1008, 1024, 1040` (Right, Left, Up, Down), plus the 11-frame collapse/starburst death animation at `x = 504..664, y = 992`.

### `assets/text.png` (`128×224` PNG)

8×8 arcade bitmap font used for the HUD, `"READY!"`, `"GAME OVER"`, and intermission titles (original source unknown). `internal/render/font.go` uppercases input strings and indexes 8×8 glyph cells as follows:

| Row (`y`) | Columns (`x = col * 8`) |
| --- | --- |
| `0` (`y = 0`) | `A`–`O` (cols `0..14`) |
| `1` (`y = 8`) | `P`–`Z` (cols `0..10`), `!` (col `11`) |
| `2` (`y = 16`) | `0`–`9` (cols `0..9`), `/` (col `10`), `-` (col `11`), `"` (col `12`) |

---

## Audio

`.gitignore` excludes `assets/*/*.wav` for local sound collections (`assets/pacman/`, `assets/ms-pacman/`). Audio playback is not currently wired into the engine.
