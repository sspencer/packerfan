package main

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Cell byte

type Cells [][]Cell

type Maze struct {
	grid   Cells
	level  int
	pacman bool
}

const (
	Empty Cell = iota
	Wall
	Door
	Dot
	Power
	Tunnel
)

const (
	EmptyMask = 0
	DotMask   = 103481868288
	PowerMask = 4359202964317896252
	DoorMask  = 16776960
)

func (c Cell) String() string {
	switch c {
	case Empty:
		return "   "
	case Wall:
		return "███"
	case Door:
		return "───"
	case Dot:
		return " · "
	case Power:
		return " ○ "
	case Tunnel:
		return "@@@"
	default:
		panic("unhandled default case")
	}
}

func (m *Maze) String() string {
	sb := strings.Builder{}
	sb.WriteString("  │")
	for x := 0; x < len(m.grid[0]); x++ {
		sb.WriteString(fmt.Sprintf("%2d│", x))
	}
	sb.WriteString("\n")

	for y := range m.grid {
		sb.WriteString(fmt.Sprintf("%2d│", y))
		for x := 0; x < len(m.grid[y]); x++ {
			sb.WriteString(m.grid[y][x].String())
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func NewMaze(pacman bool) *Maze {
	cells := make(Cells, GameHeight)
	for i := range GameHeight {
		cells[i] = make([]Cell, GameWidth)
	}

	return &Maze{
		grid:   cells,
		pacman: pacman,
		level:  1,
	}
}

func (m *Maze) computeMask(startX, startY int32) uint64 {
	var result uint64

	// Read each pixel in the specified area
	for y := startY; y < startY+Tile; y++ {
		for x := startX; x < startX+Tile; x++ {
			result <<= 1
			color := rl.GetImageColor(*art.image, x, y)
			if color.R > 0 || color.G > 0 || color.B > 0 {
				result |= 1
			}
		}
	}

	return result
}

func (m *Maze) getIndex() int {
	if m.pacman {
		return 4
	}

	if m.level <= 2 {
		return 0 // Levels 1-2: Maze 1 (index 0)
	}

	if m.level <= 5 {
		return 1 // Levels 3-5: Maze 2 (index 1)
	}

	if m.level <= 9 {
		return 2 // Levels 6-9: Maze 3 (index 2)
	}

	if m.level <= 13 {
		return 3 // Levels 10-13: Maze 4 (index 3)
	}

	// After level 13, alternate between mazes 3 and 4 every four levels.
	// Level 14-17: Maze 3 (index 2)
	// Level 18-21: Maze 4 (index 3)
	// Level 22-25: Maze 3 (index 2)
	// ...
	cycle := (m.level - 14) / 4
	if cycle%2 == 0 {
		return 2
	}

	return 3
}

func (m *Maze) setTunnels(x1, x2, y int) {
	for x := 0; x <= x1; x++ {
		m.grid[y][x] = Tunnel
	}

	for x := x2; x < GameWidth; x++ {
		m.grid[y][x] = Tunnel
	}
}

func (m *Maze) setCell(x, y int, c Cell) {
	m.grid[y][x] = c
}

func (m *Maze) getCell(x, y int) Cell {
	return m.grid[y][x]
}

func (m *Maze) setLevel(level int) {
	m.level = level
	mazeIndex := m.getIndex()
	offset := mazeIndex * GameHeight * Tile
	game.dotsEaten = 0
	game.dotsRemaining = 0

	for y := range GameHeight {
		for x := range GameWidth {
			mask := m.computeMask(int32(x*Tile), int32(y*Tile+offset))
			switch mask {
			case EmptyMask:
				m.grid[y][x] = Empty
			case DotMask:
				m.grid[y][x] = Dot
				game.dotsRemaining++
			case PowerMask:
				game.dotsRemaining++
				m.grid[y][x] = Power
			case DoorMask:
				m.grid[y][x] = Door
			default:
				m.grid[y][x] = Wall
			}
		}
	}

	switch mazeIndex {
	case 0:
		m.setTunnels(1, 26, 8)
		m.setTunnels(1, 26, 17)
	case 1:
		m.setTunnels(4, 23, 1)
		m.setTunnels(1, 26, 23)
	case 2:
		m.setTunnels(0, 27, 9)
	case 3:
		m.setTunnels(1, 26, 13)
		m.setTunnels(1, 26, 16)
	default:
		m.setTunnels(4, 23, 14)
	}
}

func (m *Maze) isGhostWall(x, y int) bool {
	if m.getIndex() == 4 {
		// only pacman has ghost walls
		return (y == 10 || y == 22) && (x == 12 || x == 15)
	}

	return false
}

func (m *Maze) draw() {
	const mazeW = float32(GameWidth) * Tile
	const mazeH = float32(GameHeight) * Tile
	index := m.getIndex()
	src := rl.Rectangle{
		X:      GameWidth*Tile + Half,
		Y:      float32(index) * GameHeight * Tile,
		Width:  mazeW,
		Height: mazeH,
	}
	dst := rl.NewRectangle(0, 0, mazeW, mazeH)
	rl.DrawTexturePro(art.texture, src, dst, rl.Vector2{}, 0, rl.White)

	dots := []float32{1, 36, 63, 94, 125}
	powerUps := []float32{2, 35, 65, 96, 127}

	dotV := rl.Rectangle{X: 1, Y: dots[index]}
	powerV := rl.Rectangle{X: 1, Y: powerUps[index]}

	dotSrc := rl.Rectangle{X: dotV.X * Tile, Y: dotV.Y * Tile, Width: Tile, Height: Tile}
	powerSrc := rl.Rectangle{X: powerV.X * Tile, Y: powerV.Y * Tile, Width: Tile, Height: Tile}

	for y := range GameHeight {
		for x := range GameWidth {
			tile := m.grid[y][x]
			if tile == Dot {
				rec := rl.Rectangle{X: float32(x) * Tile, Y: float32(y) * Tile, Width: Tile, Height: Tile}
				rl.DrawTexturePro(art.texture, dotSrc, rec, rl.Vector2{}, 0, rl.White)
			} else if tile == Power {
				rec := rl.Rectangle{X: float32(x) * Tile, Y: float32(y) * Tile, Width: Tile, Height: Tile}
				rl.DrawTexturePro(art.texture, powerSrc, rec, rl.Vector2{}, 0, rl.White)
			}
		}
	}
}
