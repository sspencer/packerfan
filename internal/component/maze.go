package component

import (
	"fmt"
	"strings"

	"github.com/yohamta/donburi"

	"pacfan/internal/config"
)

type Cell byte

type Cells [][]Cell

const (
	Empty Cell = iota
	Wall
	Door
	Dot
	Power
	Tunnel
)

const (
	EmptyMask uint64 = 0
	DotMask   uint64 = 103481868288
	PowerMask uint64 = 4359202964317896252
	DoorMask  uint64 = 16776960
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

// MazeData is the singleton world resource holding the 28x31 tile grid and
// maze metadata for the active level.
type MazeData struct {
	Grid       Cells
	Level      int
	PacmanMode bool
}

// NewMazeData allocates an empty GameWidth x GameHeight maze grid.
func NewMazeData(pacmanMode bool) MazeData {
	cells := make(Cells, config.GameHeight)
	for i := range config.GameHeight {
		cells[i] = make([]Cell, config.GameWidth)
	}

	return MazeData{
		Grid:       cells,
		PacmanMode: pacmanMode,
		Level:      1,
	}
}

func (m *MazeData) String() string {
	sb := strings.Builder{}
	sb.WriteString("  │")
	for x := 0; x < len(m.Grid[0]); x++ {
		sb.WriteString(fmt.Sprintf("%2d│", x))
	}
	sb.WriteString("\n")

	for y := range m.Grid {
		sb.WriteString(fmt.Sprintf("%2d│", y))
		for x := 0; x < len(m.Grid[y]); x++ {
			sb.WriteString(m.Grid[y][x].String())
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetIndex returns the spritesheet maze index (4 for Pac-Man; 0..3 for Ms. Pac-Man).
func (m *MazeData) GetIndex() int {
	if m.PacmanMode {
		return 4
	}

	if m.Level <= 2 {
		return 0 // Levels 1-2: Maze 1 (index 0)
	}
	if m.Level <= 5 {
		return 1 // Levels 3-5: Maze 2 (index 1)
	}
	if m.Level <= 9 {
		return 2 // Levels 6-9: Maze 3 (index 2)
	}
	if m.Level <= 13 {
		return 3 // Levels 10-13: Maze 4 (index 3)
	}

	// After level 13, alternate between mazes 3 and 4 every four levels.
	cycle := (m.Level - 14) / 4
	if cycle%2 == 0 {
		return 2
	}
	return 3
}

func (m *MazeData) InBounds(x, y int) bool {
	return x >= 0 && x < config.GameWidth && y >= 0 && y < config.GameHeight
}

func (m *MazeData) SetCell(x, y int, c Cell) {
	m.Grid[y][x] = c
}

func (m *MazeData) GetCell(x, y int) Cell {
	return m.Grid[y][x]
}

// IsGhostWall returns true for the special upward-restricted tiles in the
// classic Pac-Man maze where non-frightened ghosts may not turn up.
func (m *MazeData) IsGhostWall(x, y int) bool {
	if m.GetIndex() == 4 {
		return (y == 10 || y == 22) && (x == 12 || x == 15)
	}
	return false
}

func (m *MazeData) setTunnels(x1, x2, y int) {
	for x := 0; x <= x1; x++ {
		m.Grid[y][x] = Tunnel
	}
	for x := x2; x < config.GameWidth; x++ {
		m.Grid[y][x] = Tunnel
	}
}

func computeTileMask(startX, startY int, isPixelLit func(x, y int) bool) uint64 {
	var result uint64
	for y := startY; y < startY+config.Tile; y++ {
		for x := startX; x < startX+config.Tile; x++ {
			result <<= 1
			if isPixelLit(x, y) {
				result |= 1
			}
		}
	}
	return result
}

// LoadLevel populates the tile grid by sampling 8x8 pixel masks from the
// spritesheet image via isPixelLit, marks tunnels, and returns the total count
// of edible dots and power pellets in the maze.
func (m *MazeData) LoadLevel(level int, isPixelLit func(x, y int) bool) int {
	m.Level = level
	mazeIndex := m.GetIndex()
	offset := mazeIndex * config.GameHeight * config.Tile
	dotsRemaining := 0

	for y := range config.GameHeight {
		for x := range config.GameWidth {
			mask := computeTileMask(x*config.Tile, y*config.Tile+offset, isPixelLit)
			switch mask {
			case EmptyMask:
				m.Grid[y][x] = Empty
			case DotMask:
				m.Grid[y][x] = Dot
				dotsRemaining++
			case PowerMask:
				m.Grid[y][x] = Power
				dotsRemaining++
			case DoorMask:
				m.Grid[y][x] = Door
			default:
				m.Grid[y][x] = Wall
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

	return dotsRemaining
}

var Maze = donburi.NewComponentType[MazeData]()

// MustFindMaze returns the singleton Maze component in the world.
func MustFindMaze(w donburi.World) *MazeData {
	return Maze.Get(Maze.MustFirst(w))
}
