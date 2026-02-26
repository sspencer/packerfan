package main

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Cell byte

type Maze [][]Cell

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

func (m Maze) String() string {
	sb := strings.Builder{}
	sb.WriteString("  │")
	for x := 0; x < len(m[0]); x++ {
		sb.WriteString(fmt.Sprintf("%2d│", x))
	}
	sb.WriteString("\n")

	for y := range m {
		sb.WriteString(fmt.Sprintf("%2d│", y))
		for x := 0; x < len(m[y]); x++ {
			sb.WriteString(m[y][x].String())
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (g *Game) computeMask(startX, startY int32) uint64 {
	var result uint64

	// Read each pixel in the specified area
	for y := startY; y < startY+Tile; y++ {
		for x := startX; x < startX+Tile; x++ {
			result <<= 1
			color := rl.GetImageColor(*g.image, x, y)
			if color.R > 0 || color.G > 0 || color.B > 0 {
				result |= 1
			}
		}
	}

	return result
}

func (g *Game) getMazeIndex(level int) int {
	if g.pacman {
		return 4
	}

	if level <= 2 {
		return 0 // Levels 1-2: Maze 1 (index 0)
	}

	if level <= 5 {
		return 1 // Levels 3-5: Maze 2 (index 1)
	}

	if level <= 9 {
		return 2 // Levels 6-9: Maze 3 (index 2)
	}

	if level <= 13 {
		return 3 // Levels 10-13: Maze 4 (index 3)
	}

	// After level 13, alternate between mazes 3 and 4 every four levels.
	// Level 14-17: Maze 3 (index 2)
	// Level 18-21: Maze 4 (index 3)
	// Level 22-25: Maze 3 (index 2)
	// ...
	cycle := (level - 14) / 4
	if cycle%2 == 0 {
		return 2
	}

	return 3
}

func (g *Game) setTunnels(x1, x2, y int) {
	for x := 0; x <= x1; x++ {
		g.maze[y][x] = Tunnel
	}

	for x := x2; x < GameWidth; x++ {
		g.maze[y][x] = Tunnel
	}
}

func (g *Game) initMaze() {
	maze := g.getMazeIndex(g.level)
	offset := maze * GameHeight * Tile
	g.dotsEaten = 0
	g.dotsRemaining = 0

	for y := range GameHeight {
		for x := range GameWidth {
			mask := g.computeMask(int32(x*Tile), int32(y*Tile+offset))
			switch mask {
			case EmptyMask:
				g.maze[y][x] = Empty
			case DotMask:
				g.maze[y][x] = Dot
				g.dotsRemaining++
			case PowerMask:
				g.dotsRemaining++
				g.maze[y][x] = Power
			case DoorMask:
				g.maze[y][x] = Door
			default:
				g.maze[y][x] = Wall
			}
		}
	}

	switch maze {
	case 0:
		g.setTunnels(1, 26, 8)
		g.setTunnels(1, 26, 17)
	case 1:
		g.setTunnels(4, 23, 1)
		g.setTunnels(1, 26, 23)
	case 2:
		g.setTunnels(0, 27, 9)
	case 3:
		g.setTunnels(1, 26, 13)
		g.setTunnels(1, 26, 16)
	default:
		g.setTunnels(4, 23, 14)
	}
}

func (g *Game) isGhostWall(x, y int) bool {
	if g.getMazeIndex(g.level) == 4 {
		// only pacman has ghost walls
		return (y == 10 || y == 22) && (x == 12 || x == 15)
	}

	return false
}

func (g *Game) drawMaze() {
	const mazeW = float32(GameWidth) * Tile
	const mazeH = float32(GameHeight) * Tile
	mazeNum := g.getMazeIndex(g.level)
	src := rl.Rectangle{
		X:      GameWidth*Tile + Half,
		Y:      float32(mazeNum) * GameHeight * Tile,
		Width:  mazeW,
		Height: mazeH,
	}
	dst := rl.NewRectangle(0, 0, mazeW, mazeH)
	rl.DrawTexturePro(g.texture, src, dst, rl.Vector2{}, 0, rl.White)

	dots := []float32{1, 36, 63, 94, 125}
	powerUps := []float32{2, 35, 65, 96, 127}

	dotV := rl.Rectangle{X: 1, Y: dots[mazeNum]}
	powerV := rl.Rectangle{X: 1, Y: powerUps[mazeNum]}

	dotSrc := rl.Rectangle{X: dotV.X * Tile, Y: dotV.Y * Tile, Width: Tile, Height: Tile}
	powerSrc := rl.Rectangle{X: powerV.X * Tile, Y: powerV.Y * Tile, Width: Tile, Height: Tile}

	for y := range GameHeight {
		for x := range GameWidth {
			tile := g.maze[y][x]
			if tile == Dot {
				rec := rl.Rectangle{X: float32(x) * Tile, Y: float32(y) * Tile, Width: Tile, Height: Tile}
				rl.DrawTexturePro(g.texture, dotSrc, rec, rl.Vector2{}, 0, rl.White)
			} else if tile == Power {
				rec := rl.Rectangle{X: float32(x) * Tile, Y: float32(y) * Tile, Width: Tile, Height: Tile}
				rl.DrawTexturePro(g.texture, powerSrc, rec, rl.Vector2{}, 0, rl.White)
			}
		}
	}
}
