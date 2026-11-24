package component

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/yohamta/donburi"

	"cheddarhead/internal/config"
)

const NumMazes = 5

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

// Char returns the single-character ASCII encoding for a tile, suitable for
// compact serialization and ML/scripting APIs.
func (c Cell) Char() byte {
	switch c {
	case Empty:
		return ' '
	case Wall:
		return '#'
	case Door:
		return '-'
	case Dot:
		return '.'
	case Power:
		return 'O'
	case Tunnel:
		return '@'
	default:
		panic("unhandled default case")
	}
}

// CellFromChar converts a single-character ASCII encoding back into a Cell.
func CellFromChar(ch byte) (Cell, error) {
	switch ch {
	case ' ':
		return Empty, nil
	case '#':
		return Wall, nil
	case '-':
		return Door, nil
	case '.':
		return Dot, nil
	case 'O':
		return Power, nil
	case '@':
		return Tunnel, nil
	default:
		return Empty, fmt.Errorf("unknown maze tile character %q", ch)
	}
}

// String returns the single-character ASCII representation of a tile.
func (c Cell) String() string {
	return string(c.Char())
}

// PrettyString returns a 3-character wide Unicode representation of a tile for
// human-readable terminal debugging.
func (c Cell) PrettyString() string {
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

// GhostHouse describes the pixel geometry of the ghost pen derived from the
// maze's Door tiles (or standard arcade coordinates when no Door tiles exist).
type GhostHouse struct {
	Entrance     config.Vector // Corridor point directly above the house door
	Center       config.Vector // Center slot inside the house
	LeftSlot     config.Vector // Left home slot inside the house (Inky)
	RightSlot    config.Vector // Right home slot inside the house (Clyde)
	BounceTop    int           // Upper vertical bounce limit inside the house
	BounceBottom int           // Lower vertical bounce limit inside the house
}

const (
	defaultDoorLeftX  = 13
	defaultDoorRightX = 14
	defaultDoorRow    = 12
)

func ghostHouseFromDoor(minDoorX, maxDoorX, doorRow int) GhostHouse {
	centerX := (minDoorX + maxDoorX) * config.Tile / 2
	entranceY := (doorRow - 1) * config.Tile
	centerY := (doorRow + 2) * config.Tile
	return GhostHouse{
		Entrance:     config.Vector{X: centerX, Y: entranceY},
		Center:       config.Vector{X: centerX, Y: centerY},
		LeftSlot:     config.Vector{X: centerX - 2*config.Tile, Y: centerY},
		RightSlot:    config.Vector{X: centerX + 2*config.Tile, Y: centerY},
		BounceTop:    centerY - config.Half,
		BounceBottom: centerY + config.Half,
	}
}

// DefaultGhostHouse returns the canonical arcade ghost-house geometry.
func DefaultGhostHouse() GhostHouse {
	return ghostHouseFromDoor(defaultDoorLeftX, defaultDoorRightX, defaultDoorRow)
}

// MazeTemplate holds a pre-parsed 28x31 maze layout (grid, edible dot count,
// and ghost-house geometry) ready to be copied into MazeData without touching
// the spritesheet pixels.
type MazeTemplate struct {
	Index int
	Grid  [config.GameHeight][config.GameWidth]Cell
	Dots  int
	House GhostHouse
}

// Rows returns the template as 31 strings of 28 single-character ASCII tiles.
func (t *MazeTemplate) Rows() []string {
	rows := make([]string, config.GameHeight)
	var buf [config.GameWidth]byte
	for y := range config.GameHeight {
		for x := range config.GameWidth {
			buf[x] = t.Grid[y][x].Char()
		}
		rows[y] = string(buf[:])
	}
	return rows
}

// String returns the template as newline-joined ASCII rows.
func (t *MazeTemplate) String() string {
	return strings.Join(t.Rows(), "\n")
}

// ParseMazeTemplateFromRows parses 31 rows of 28 ASCII characters into a MazeTemplate.
func ParseMazeTemplateFromRows(index int, rows []string) (MazeTemplate, error) {
	if len(rows) != config.GameHeight {
		return MazeTemplate{}, fmt.Errorf("maze %d: got %d rows, want %d", index, len(rows), config.GameHeight)
	}
	tmpl := MazeTemplate{Index: index}
	for y, row := range rows {
		if len(row) != config.GameWidth {
			return MazeTemplate{}, fmt.Errorf("maze %d row %d: got length %d, want %d", index, y, len(row), config.GameWidth)
		}
		for x := range config.GameWidth {
			cell, err := CellFromChar(row[x])
			if err != nil {
				return MazeTemplate{}, fmt.Errorf("maze %d (%d,%d): %w", index, x, y, err)
			}
			tmpl.Grid[y][x] = cell
			if cell == Dot || cell == Power {
				tmpl.Dots++
			}
		}
	}
	tmpl.House = computeGhostHouseFromTemplate(&tmpl.Grid)
	return tmpl, nil
}

// ParseMazeTemplate classifies the 28x31 tile grid for mazeIndex (0..3 Ms. Pac-Man,
// 4 classic Pac-Man) by sampling 8x8 pixel masks via isPixelLit, marks tunnels,
// counts edible dots/power pellets, and derives ghost-house geometry.
func ParseMazeTemplate(mazeIndex int, isPixelLit func(x, y int) bool) MazeTemplate {
	tmpl := MazeTemplate{Index: mazeIndex}
	offset := mazeIndex * config.GameHeight * config.Tile

	for y := range config.GameHeight {
		for x := range config.GameWidth {
			mask := computeTileMask(x*config.Tile, y*config.Tile+offset, isPixelLit)
			switch mask {
			case EmptyMask:
				tmpl.Grid[y][x] = Empty
			case DotMask:
				tmpl.Grid[y][x] = Dot
				tmpl.Dots++
			case PowerMask:
				tmpl.Grid[y][x] = Power
				tmpl.Dots++
			case DoorMask:
				tmpl.Grid[y][x] = Door
			default:
				tmpl.Grid[y][x] = Wall
			}
		}
	}

	applyTunnelsToTemplate(&tmpl.Grid, mazeIndex)
	tmpl.House = computeGhostHouseFromTemplate(&tmpl.Grid)
	return tmpl
}

// MazeSet holds all five pre-parsed arcade maze templates (0..3 Ms. Pac-Man,
// 4 classic Pac-Man) and their combined layout hash.
type MazeSet struct {
	Templates [NumMazes]MazeTemplate
	hash      string
}

// ParseMazeSet parses all five maze layouts from isPixelLit into a MazeSet.
func ParseMazeSet(isPixelLit func(x, y int) bool) *MazeSet {
	set := &MazeSet{}
	for i := range NumMazes {
		set.Templates[i] = ParseMazeTemplate(i, isPixelLit)
	}
	set.hash = computeMazeSetHash(&set.Templates)
	return set
}

// NewMazeSetFromTemplates constructs a MazeSet from pre-built templates and computes its hash.
func NewMazeSetFromTemplates(templates [NumMazes]MazeTemplate) *MazeSet {
	set := &MazeSet{Templates: templates}
	set.hash = computeMazeSetHash(&set.Templates)
	return set
}

// Template returns the template for mazeIndex (0..4).
func (s *MazeSet) Template(mazeIndex int) *MazeTemplate {
	return &s.Templates[mazeIndex]
}

// Hash returns the lowercase hex-encoded SHA-256 hash of all five parsed maze texts.
func (s *MazeSet) Hash() string {
	return s.hash
}

func computeMazeSetHash(templates *[NumMazes]MazeTemplate) string {
	h := sha256.New()
	for i := range NumMazes {
		fmt.Fprintf(h, "maze %d\n", i)
		for _, row := range templates[i].Rows() {
			h.Write([]byte(row))
			h.Write([]byte{'\n'})
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func computeGhostHouseFromTemplate(grid *[config.GameHeight][config.GameWidth]Cell) GhostHouse {
	found := false
	minX, maxX, doorRow := config.GameWidth, -1, 0
	for y := range config.GameHeight {
		for x := range config.GameWidth {
			if grid[y][x] != Door {
				continue
			}
			if !found {
				found = true
				doorRow = y
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
		}
	}
	if !found {
		return DefaultGhostHouse()
	}
	return ghostHouseFromDoor(minX, maxX, doorRow)
}

func setTemplateTunnels(grid *[config.GameHeight][config.GameWidth]Cell, x1, x2, y int) {
	for x := 0; x <= x1; x++ {
		grid[y][x] = Tunnel
	}
	for x := x2; x < config.GameWidth; x++ {
		grid[y][x] = Tunnel
	}
}

func applyTunnelsToTemplate(grid *[config.GameHeight][config.GameWidth]Cell, mazeIndex int) {
	switch mazeIndex {
	case 0:
		setTemplateTunnels(grid, 1, 26, 8)
		setTemplateTunnels(grid, 1, 26, 17)
	case 1:
		setTemplateTunnels(grid, 4, 23, 1)
		setTemplateTunnels(grid, 1, 26, 23)
	case 2:
		setTemplateTunnels(grid, 0, 27, 9)
	case 3:
		setTemplateTunnels(grid, 1, 26, 13)
		setTemplateTunnels(grid, 1, 26, 16)
	default:
		setTemplateTunnels(grid, 4, 23, 14)
	}
}

// MazeData is the singleton world resource holding the 28x31 tile grid and
// maze metadata for the active level.
type MazeData struct {
	Grid       Cells
	Level      int
	PacmanMode bool
	house      GhostHouse
	houseDirty bool
	templates  *MazeSet
}

// UseTemplates configures MazeData to copy pre-parsed templates on LoadLevel
// instead of sampling spritesheet pixels via isPixelLit.
func (m *MazeData) UseTemplates(set *MazeSet) {
	m.templates = set
}

// HasTemplates reports whether MazeData has a pre-parsed MazeSet configured.
func (m *MazeData) HasTemplates() bool {
	return m.templates != nil
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
		house:      DefaultGhostHouse(),
	}
}

// Rows returns the maze as a slice of 1-character-per-tile ASCII row strings
// (31 rows of 28 characters) suitable for JSON/scripting serialization.
func (m *MazeData) Rows() []string {
	rows := make([]string, len(m.Grid))
	for y := range m.Grid {
		buf := make([]byte, len(m.Grid[y]))
		for x, cell := range m.Grid[y] {
			buf[x] = cell.Char()
		}
		rows[y] = string(buf)
	}
	return rows
}

// String returns a compact 1-character-per-tile ASCII representation of the maze
// (rows separated by newlines, no coordinate borders).
func (m *MazeData) String() string {
	return strings.Join(m.Rows(), "\n")
}

// PrettyString returns a 3-character-per-tile Unicode grid with row/column
// indices for human-readable terminal debugging.
func (m *MazeData) PrettyString() string {
	sb := strings.Builder{}
	sb.WriteString("  │")
	for x := 0; x < len(m.Grid[0]); x++ {
		sb.WriteString(fmt.Sprintf("%2d│", x))
	}
	sb.WriteString("\n")

	for y := range m.Grid {
		sb.WriteString(fmt.Sprintf("%2d│", y))
		for x := 0; x < len(m.Grid[y]); x++ {
			sb.WriteString(m.Grid[y][x].PrettyString())
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
	if m.Grid[y][x] == Door || c == Door {
		m.houseDirty = true
	}
	m.Grid[y][x] = c
}

func (m *MazeData) GetCell(x, y int) Cell {
	return m.Grid[y][x]
}

// InvalidateHouseCache forces GhostHouse() to recompute from the current Grid.
func (m *MazeData) InvalidateHouseCache() {
	m.houseDirty = true
}

// GhostHouse returns the ghost-house geometry derived from the maze's Door
// tiles, falling back to DefaultGhostHouse when the grid contains no Door cells.
func (m *MazeData) GhostHouse() GhostHouse {
	if !m.houseDirty && m.house.Entrance != (config.Vector{}) {
		return m.house
	}
	m.house = m.computeGhostHouse()
	m.houseDirty = false
	return m.house
}

func (m *MazeData) computeGhostHouse() GhostHouse {
	found := false
	minX, maxX, doorRow := config.GameWidth, -1, 0
	for y := range len(m.Grid) {
		for x := range len(m.Grid[y]) {
			if m.Grid[y][x] != Door {
				continue
			}
			if !found {
				found = true
				doorRow = y
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
		}
	}
	if !found {
		return DefaultGhostHouse()
	}
	return ghostHouseFromDoor(minX, maxX, doorRow)
}

// TunnelRows returns the maze tile Y rows containing left/right wrap tunnels
// for the active maze layout.
func (m *MazeData) TunnelRows() []int {
	switch m.GetIndex() {
	case 0:
		return []int{8, 17}
	case 1:
		return []int{1, 23}
	case 2:
		return []int{9}
	case 3:
		return []int{13, 16}
	default:
		return []int{14}
	}
}

// IsGhostWall returns true for the special upward-restricted tiles in the
// classic Pac-Man maze where non-frightened ghosts may not turn up.
func (m *MazeData) IsGhostWall(x, y int) bool {
	if m.GetIndex() == 4 {
		return (y == 10 || y == 22) && (x == 12 || x == 15)
	}
	return false
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

// LoadLevel populates the tile grid for the given level, marks tunnels, and
// returns the total count of edible dots and power pellets in the maze.
// When UseTemplates has been configured, LoadLevel copies the pre-parsed
// template and ignores isPixelLit; otherwise it samples 8x8 pixel masks via
// isPixelLit.
func (m *MazeData) LoadLevel(level int, isPixelLit func(x, y int) bool) int {
	m.Level = level
	mazeIndex := m.GetIndex()

	var tmpl *MazeTemplate
	if m.templates != nil && mazeIndex >= 0 && mazeIndex < NumMazes {
		tmpl = &m.templates.Templates[mazeIndex]
	} else {
		parsed := ParseMazeTemplate(mazeIndex, isPixelLit)
		tmpl = &parsed
	}

	for y := range config.GameHeight {
		copy(m.Grid[y], tmpl.Grid[y][:])
	}
	m.house = tmpl.House
	m.houseDirty = false

	return tmpl.Dots
}

var Maze = donburi.NewComponentType[MazeData]()

// MustFindMaze returns the singleton Maze component in the world.
func MustFindMaze(w donburi.World) *MazeData {
	return Maze.Get(Maze.MustFirst(w))
}
