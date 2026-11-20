package main

import (
	"fmt"
	"strings"
)

// Original example maze layouts. Each maze is described by its left half
// (columns 0-13, 31 rows) and mirrored to produce the right half, as in the
// arcade mazes.
//
//	#  wall block (tiles not touching a corridor become hollow Empty tiles)
//	.  corridor with a dot
//	o  corridor with a power pellet
//	   corridor without a dot (tunnels, the ring around the ghost house)
//	-  ghost-house door
//
// Every maze shares the same ghost house (door at row 12, columns 13-14),
// the clear corridor ring around it (rows 11 and 17, columns 9 and 18), and
// the dot-free Pac-Man start tiles (row 23, columns 13-14). Tunnel rows match
// the per-maze rows hardcoded in internal/component/maze.go.
var mazeHalves = [5][31]string{
	// Maze 0 (Ms. Pac-Man levels 1-2): tunnels on rows 8 and 17.
	{
		"##############",
		"#............#",
		"#o####.#####.#",
		"#.####.#####.#",
		"#.............",
		"######.##.####",
		"######.##.####",
		"######.##.####",
		"  .....##.....",
		"######.##.####",
		"######.##.####",
		"######...     ",
		"######.## ###-",
		"######.## #   ",
		"#......   #   ",
		"#.####.## #   ",
		"#.####.## ####",
		"  .......     ",
		"######.#####.#",
		"######.#####.#",
		"#............#",
		"#.#######.####",
		"#.#######.####",
		"#............ ",
		"#.####.#####.#",
		"#.####.#####.#",
		"#o.....##....#",
		"#.#######.####",
		"#.#######.####",
		"#.............",
		"##############",
	},
	// Maze 1 (levels 3-5): tunnels on rows 1 and 23.
	{
		"##############",
		"     ........#",
		"######.#####.#",
		"######.#####.#",
		"#o.....##....#",
		"#.####.##.####",
		"#.####.##.####",
		"#.####.##.####",
		"#.............",
		"#.#######.####",
		"#.#######.####",
		"#......##     ",
		"######.## ###-",
		"######.## #   ",
		"#......   #   ",
		"#.####### #   ",
		"#.####### ####",
		"#........     ",
		"######.#####.#",
		"######.#####.#",
		"######........",
		"######.##.####",
		"######.##.####",
		"  ........... ",
		"######.#####.#",
		"######.#####.#",
		"#o...........#",
		"#.#######.####",
		"#.#######.####",
		"#.............",
		"##############",
	},
	// Maze 2 (levels 6-9, alternating after 13): tunnel on row 9.
	{
		"##############",
		"#......##.....",
		"#.####.##.####",
		"#o####.##.####",
		"#.####.##.####",
		"#.............",
		"####.####.##.#",
		"####.####.##.#",
		"####.####.##.#",
		" ....####.##.#",
		"#.##.####.##.#",
		"#.##.....     ",
		"#.####.## ###-",
		"#.####.## #   ",
		"#......   #   ",
		"#.####### #   ",
		"#.####### ####",
		"#........     ",
		"#.####.#####.#",
		"#.####.#####.#",
		"#............#",
		"####.####.####",
		"####.####.####",
		"#o........... ",
		"#.####.#####.#",
		"#.####.#####.#",
		"#.........##..",
		"#.#######.####",
		"#.#######.####",
		"#.............",
		"##############",
	},
	// Maze 3 (levels 10-13, alternating after 13): tunnels on rows 13 and 16.
	{
		"##############",
		"#.............",
		"#.#####.##.###",
		"#o#####.##.###",
		"#.......##....",
		"#.##.#####.###",
		"#.##.#####.###",
		"#.##.#####.###",
		"#..........###",
		"####.#####.###",
		"####.#####.###",
		"####.....     ",
		"####.#### ###-",
		"  ...#### #   ",
		"####.#### #   ",
		"####.#### #   ",
		"  ...#### ####",
		"####.....     ",
		"####.##.##.###",
		"####.##.##.###",
		"#..........###",
		"#.#####.######",
		"#.#####.######",
		"#............ ",
		"#.####.#####.#",
		"#.####.#####.#",
		"#o.....##.....",
		"#.####.##.####",
		"#.####.##.####",
		"#.............",
		"##############",
	},
	// Maze 4 (Pac-Man): tunnel on row 14; 240 dots + 4 power pellets.
	{
		"##############",
		"#............#",
		"#.####.#####.#",
		"#o####.#####.#",
		"#.####.#####.#",
		"#.............",
		"#.#######.####",
		"#.#######.####",
		"#............#",
		"######.#####.#",
		"###### #####.#",
		"###### ..     ",
		"###### ## ###-",
		"###### ## #   ",
		"          #   ",
		"###### ## #   ",
		"###### ## ####",
		"###### ..     ",
		"###### #####.#",
		"######.#####.#",
		"#............#",
		"#.####.#####.#",
		"#.####.#####.#",
		"#o........... ",
		"######.##.####",
		"######.##.####",
		"#......##.....",
		"#.##########.#",
		"#.##########.#",
		"#.............",
		"##############",
	},
}

// tunnelSpans mirrors the setTunnels calls in MazeData.LoadLevel: for each
// maze, {row, last left tunnel column}. Right-side spans are mirrored.
var tunnelSpans = [5][][2]int{
	{{8, 1}, {17, 1}},
	{{1, 4}, {23, 1}},
	{{9, 0}},
	{{13, 1}, {16, 1}},
	{{14, 4}},
}

// rendererDotRows and rendererPowerRows are the maze-local rows of the column-1
// tiles Renderer.drawMaze copies dot and power-pellet graphics from.
var (
	rendererDotRows   = [5]int{1, 5, 1, 1, 1}
	rendererPowerRows = [5]int{2, 4, 3, 3, 3}
)

// expandMaze mirrors a left-half layout into a full 28x31 character grid.
func expandMaze(half [31]string) ([31][28]byte, error) {
	var g [31][28]byte
	for y, row := range half {
		if len(row) != 14 {
			return g, fmt.Errorf("row %d has %d columns, want 14", y, len(row))
		}
		for x := range 14 {
			g[y][x] = row[x]
			g[y][27-x] = row[x]
		}
	}
	return g, nil
}

// tileGrid converts a character grid to tile types. Wall-block tiles that do
// not touch a corridor (8-neighbourhood) become Empty, leaving thick blocks
// hollow like the arcade artwork.
func tileGrid(c [31][28]byte) [31][28]int {
	var g [31][28]int
	for y := range 31 {
		for x := range 28 {
			switch c[y][x] {
			case '.':
				g[y][x] = Dot
			case 'o':
				g[y][x] = Power
			case '-':
				g[y][x] = Door
			case ' ':
				g[y][x] = Empty
			case '#':
				g[y][x] = Empty
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= 0 && ny >= 0 && nx < 28 && ny < 31 && c[ny][nx] != '#' {
							g[y][x] = Wall
						}
					}
				}
			}
		}
	}
	return g
}

func inHouse(x, y int) bool { return y >= 13 && y <= 15 && x >= 11 && x <= 16 }

// houseWall reports whether (x, y) is part of the ghost house's 1-tile-thick
// walls, which are exempt from the wall thickness rule.
func houseWall(x, y int) bool { return y >= 12 && y <= 16 && x >= 10 && x <= 17 }

// houseOrDoor reports whether the 2x2 block at (x, y) touches the ghost-house
// interior or door, where open 2x2 areas are expected.
func houseOrDoor(c [31][28]byte, x, y int) bool {
	for dy := range 2 {
		for dx := range 2 {
			if inHouse(x+dx, y+dy) || c[y+dy][x+dx] == '-' {
				return true
			}
		}
	}
	return false
}

// validateMaze checks a character grid against the layout rules the game
// relies on, returning every problem found.
func validateMaze(idx int, c [31][28]byte) []string {
	var errs []string
	fail := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	open := func(x, y int) bool { return c[y][x] != '#' }

	tunnel := map[[2]int]bool{}
	tunnelRow := map[int]bool{}
	for _, s := range tunnelSpans[idx] {
		tunnelRow[s[0]] = true
		for x := 0; x <= s[1]; x++ {
			tunnel[[2]int{x, s[0]}] = true
			tunnel[[2]int{27 - x, s[0]}] = true
		}
	}
	for t := range tunnel {
		if c[t[1]][t[0]] != ' ' {
			fail("tunnel tile (%d,%d) must be a dot-free corridor, got %q", t[0], t[1], c[t[1]][t[0]])
		}
	}
	for y := range 31 {
		for _, x := range []int{0, 27} {
			if open(x, y) && !tunnelRow[y] {
				fail("edge tile (%d,%d) is open on a non-tunnel row", x, y)
			}
		}
	}
	for x := range 28 {
		if open(x, 0) || open(x, 30) {
			fail("top/bottom border open at column %d", x)
		}
	}

	// Fixed ghost house, ring and start position.
	house := []string{
		"          ", // row 11, columns 9-18
		" ###--### ",
		" #      # ",
		" #      # ",
		" #      # ",
		" ######## ",
		"          ", // row 17
	}
	for j, row := range house {
		for i := range row {
			if got := c[11+j][9+i]; got != row[i] {
				fail("ghost house region (%d,%d) = %q, want %q", 9+i, 11+j, got, row[i])
			}
		}
	}
	for _, x := range []int{13, 14} {
		if c[23][x] != ' ' {
			fail("start tile (%d,23) must be a dot-free corridor", x)
		}
	}
	if c[rendererDotRows[idx]][1] != '.' {
		fail("renderer dot source (1,%d) is not a dot", rendererDotRows[idx])
	}
	if c[rendererPowerRows[idx]][1] != 'o' {
		fail("renderer power source (1,%d) is not a power pellet", rendererPowerRows[idx])
	}

	neighbors := func(x, y int) [][2]int {
		var n [][2]int
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if tunnelRow[y] && (nx < 0 || nx > 27) {
				nx = (nx + 28) % 28
			}
			if nx >= 0 && ny >= 0 && nx < 28 && ny < 31 && open(nx, ny) {
				n = append(n, [2]int{nx, ny})
			}
		}
		return n
	}

	edibles, start := 0, [2]int{}
	total := 0
	for y := range 31 {
		for x := range 28 {
			if !open(x, y) {
				if x > 0 && x < 27 && y > 0 && y < 30 && !houseWall(x, y) &&
					((open(x-1, y) && open(x+1, y)) || (open(x, y-1) && open(x, y+1))) {
					fail("1-tile-thick wall at (%d,%d)", x, y)
				}
				continue
			}
			total++
			start = [2]int{x, y}
			if c[y][x] == '.' || c[y][x] == 'o' {
				edibles++
			}
			if !inHouse(x, y) && c[y][x] != '-' && len(neighbors(x, y)) < 2 {
				fail("dead end at (%d,%d)", x, y)
			}
			if x < 27 && y < 30 && open(x+1, y) && open(x, y+1) && open(x+1, y+1) &&
				!houseOrDoor(c, x, y) {
				fail("2x2 open area at (%d,%d)", x, y)
			}
		}
	}

	seen := map[[2]int]bool{start: true}
	stack := [][2]int{start}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, n := range neighbors(p[0], p[1]) {
			if !seen[n] {
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	if len(seen) != total {
		fail("maze is disconnected: reached %d of %d open tiles", len(seen), total)
	}

	if idx == 4 {
		if edibles != 244 {
			fail("Pac-Man maze has %d dots+pellets, want 244", edibles)
		}
		for _, t := range [][2]int{{12, 10}, {15, 10}, {12, 22}, {15, 22}} {
			if !open(t[0], t[1]) {
				fail("Pac-Man ghost no-up tile (%d,%d) should be a corridor", t[0], t[1])
			}
		}
	}
	for _, t := range [][2]int{{1, 29}, {26, 29}, {1, 1}, {26, 1}} {
		if idx == 4 && !open(t[0], t[1]) {
			fail("corner corridor tile (%d,%d) is closed", t[0], t[1])
		}
	}
	if !open(1, 29) {
		fail("bottom-left corridor tile (1,29) is closed")
	}
	return errs
}

// buildMazes expands, validates and converts all example mazes.
func buildMazes() ([5][31][28]int, error) {
	var grids [5][31][28]int
	var problems []string
	for idx, half := range mazeHalves {
		c, err := expandMaze(half)
		if err != nil {
			problems = append(problems, fmt.Sprintf("maze %d: %v", idx, err))
			continue
		}
		for _, e := range validateMaze(idx, c) {
			problems = append(problems, fmt.Sprintf("maze %d: %s", idx, e))
		}
		grids[idx] = tileGrid(c)
	}
	if len(problems) > 0 {
		return grids, fmt.Errorf("invalid maze layouts:\n  %s", strings.Join(problems, "\n  "))
	}
	return grids, nil
}
