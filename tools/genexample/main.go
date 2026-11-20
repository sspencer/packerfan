// Command genexample generates placeholder spritesheets for local development:
// assets/game.png and assets/text.png. Every sprite, maze, and
// glyph is drawn as a simple shape at the coordinates the game reads (see
// assets/README.md).
//
// The five maze layouts are original designs (see mazes.go) that share the
// arcade mazes' ghost house, Pac-Man start position, tunnel rows, and the
// tiles the game samples for dot and power-pellet graphics.
//
// Usage:
//
//	go run ./tools/genexample assets/game.png assets/text.png
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type C = color.RGBA

func rgb(r, g, b uint8) C { return C{R: r, G: g, B: b, A: 255} }

var (
	black  = rgb(0, 0, 0)
	white  = rgb(255, 255, 255)
	yellow = rgb(255, 255, 0)
	red    = rgb(255, 0, 0)
	pink   = rgb(255, 184, 255)
	cyan   = rgb(0, 255, 255)
	orange = rgb(255, 184, 82)
	blue   = rgb(33, 33, 255)
	peach  = rgb(255, 184, 174)
	green  = rgb(0, 200, 0)
	brown  = rgb(200, 120, 40)
	grey   = rgb(170, 170, 170)
	purple = rgb(180, 80, 255)
)

var img *image.RGBA

func set(x, y int, c C) {
	if image.Pt(x, y).In(img.Rect) {
		img.SetRGBA(x, y, c)
	}
}
func rect(x, y, w, h int, c C) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			set(i, j, c)
		}
	}
}

// disc fills pixels whose centre lies within r of (cx,cy), relative to origin (ox,oy).
func disc(ox, oy int, cx, cy, r float64, c C) {
	for j := int(cy - r - 1); j <= int(cy+r+1); j++ {
		for i := int(cx - r - 1); i <= int(cx+r+1); i++ {
			dx, dy := float64(i)+0.5-cx, float64(j)+0.5-cy
			if dx*dx+dy*dy <= r*r {
				set(ox+i, oy+j, c)
			}
		}
	}
}

// ---------- tiny fonts ----------

var digits3x5 = map[rune][5]string{
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", "###"},
	'2': {"###", "..#", "###", "#..", "###"},
	'3': {"###", "..#", ".##", "..#", "###"},
	'4': {"#.#", "#.#", "###", "..#", "..#"},
	'5': {"###", "#..", "###", "..#", "###"},
	'6': {"###", "#..", "###", "#.#", "###"},
	'7': {"###", "..#", ".#.", ".#.", ".#."},
	'8': {"###", "#.#", "###", "#.#", "###"},
	'9': {"###", "#.#", "###", "..#", "###"},
}

// number draws s with the 3x5 font centred horizontally in a box of width w.
func number(x, y, w int, s string, c C) {
	tw := len(s)*4 - 1
	x += (w - tw) / 2
	for _, ch := range s {
		g := digits3x5[ch]
		for j, row := range g {
			for i, p := range row {
				if p == '#' {
					set(x+i, y+j, c)
				}
			}
		}
		x += 4
	}
}

var font5x7 = map[rune][7]string{
	'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'B': {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'C': {".###.", "#...#", "#....", "#....", "#....", "#...#", ".###."},
	'D': {"####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'F': {"#####", "#....", "#....", "####.", "#....", "#....", "#...."},
	'G': {".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".####"},
	'H': {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'I': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "#####"},
	'J': {"..###", "...#.", "...#.", "...#.", "...#.", "#..#.", ".##.."},
	'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
	'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'P': {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'Q': {".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'S': {".####", "#....", "#....", ".###.", "....#", "....#", "####."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'V': {"#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
	'X': {"#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"},
	'Y': {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
	'Z': {"#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"},
	'0': {".###.", "#..##", "#.#.#", "#.#.#", "#.#.#", "##..#", ".###."},
	'1': {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "....#", "..##.", ".#...", "#....", "#####"},
	'3': {"#####", "...#.", "..#..", "...#.", "....#", "#...#", ".###."},
	'4': {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5': {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6': {"..##.", ".#...", "#....", "####.", "#...#", "#...#", ".###."},
	'7': {"#####", "....#", "...#.", "..#..", ".#...", ".#...", ".#..."},
	'8': {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9': {".###.", "#...#", "#...#", ".####", "....#", "...#.", ".##.."},
	'!': {"..#..", "..#..", "..#..", "..#..", "..#..", ".....", "..#.."},
	'/': {"....#", "....#", "...#.", "..#..", ".#...", "#....", "#...."},
	'-': {".....", ".....", ".....", "#####", ".....", ".....", "....."},
	'"': {".#.#.", ".#.#.", ".#.#.", ".....", ".....", ".....", "....."},
}

func glyph(x, y int, ch rune, c C) {
	for j, row := range font5x7[ch] {
		for i, p := range row {
			if p == '#' {
				set(x+1+i, y+j, c)
			}
		}
	}
}

// ---------- sprites ----------

var dirAngle = [4]float64{0, math.Pi, -math.Pi / 2, math.Pi / 2} // R, L, U, D (screen y down)

// pac draws a size x size rounded square facing dir with a wedge mouth of
// half-angle mouth (radians) cut out of the facing side.
func pac(x, y, size int, dir float64, mouth float64, c C) {
	m := max(size/16, 0)  // margin around the square
	lo, hi := m, size-1-m // inclusive pixel bounds
	cx, cy := float64(size)/2, float64(size)/2
	for j := lo; j <= hi; j++ {
		for i := lo; i <= hi; i++ {
			if (i == lo || i == hi) && (j == lo || j == hi) {
				continue // round off the corners
			}
			dx, dy := float64(i)+0.5-cx, float64(j)+0.5-cy
			a := math.Atan2(dy, dx) - dir
			for a > math.Pi {
				a -= 2 * math.Pi
			}
			for a < -math.Pi {
				a += 2 * math.Pi
			}
			if math.Abs(a) < mouth {
				continue
			}
			set(x+i, y+j, c)
		}
	}
}

var mouths = [3]float64{0.8, 0.4, 0} // frame 0 open, 1 half, 2 closed

func msBow(x, y int) {
	rect(x+2, y+1, 2, 3, red)
	rect(x+5, y+1, 2, 3, red)
	rect(x+4, y+2, 1, 1, blue)
}

// ghostBody draws an upside-down triangle (flat top at row 1, point at row 15)
// whose tip sways left or right between the two animation frames.
func ghostBody(x, y, frame int, body C) {
	sway := []float64{-1.5, 1.5}[frame%2]
	for j := 1; j <= 15; j++ {
		t := float64(j-1) / 14
		hw := 7.5*(1-t) + 0.5
		center := 8 + sway*t
		for i := range 16 {
			if math.Abs(float64(i)+0.5-center) <= hw {
				set(x+i, y+j, body)
			}
		}
	}
}

func ghostEyes(x, y, dir int) {
	dx := [4]int{1, -1, 0, 0}[dir]
	dy := [4]int{0, 0, -1, 1}[dir]
	for _, ex := range []int{4, 9} {
		rect(x+ex+dx, y+4+dy, 4, 5, white)
		rect(x+ex+1+dx*2, y+5+dy*2+1, 2, 2, blue)
	}
}

func frightFace(x, y int, face C) {
	rect(x+5, y+4, 2, 2, face)
	rect(x+10, y+4, 2, 2, face)
	for i := range 7 {
		set(x+5+i, y+7+i%2, face)
	}
}

func fruit(x, y int, c C) {
	disc(x, y, 8, 9, 5, c)
	rect(x+8, y+2, 1, 3, green)
	rect(x+9, y+2, 3, 1, green)
}

func drawSprites() {
	// Ms. Pac-Man: 3 frames x 4 dirs at x=456, y=0..48.
	for d := range 4 {
		for f := range 3 {
			x, y := 456+f*16, d*16
			pac(x, y, 16, dirAngle[d], mouths[f], yellow)
			msBow(x, y)
		}
	}
	// Ms. Pac-Man bonus fruit (7) at y=0, scores at y=16.
	msFruit := []C{red, rgb(255, 60, 120), orange, brown, rgb(255, 40, 40), rgb(160, 255, 80), yellow}
	for i, c := range msFruit {
		fruit(504+i*16, 0, c)
	}
	for i, s := range []string{"100", "200", "500", "700", "1000", "2000", "5000"} {
		number(504+i*16, 16+5, 16, s, pink)
	}
	// Ghosts: rows 64,80,96,112; x=456,488,520,552 (R,L,U,D), 2 frames each.
	for g, c := range []C{red, pink, cyan, orange} {
		for d := range 4 {
			for f := range 2 {
				x, y := 456+d*32+f*16, 64+g*16
				ghostBody(x, y, f, c)
				ghostEyes(x, y, d)
			}
		}
	}
	// Frightened blue (584,64) and white (616,64), 2 frames each.
	for f := range 2 {
		ghostBody(584+f*16, 64, f, blue)
		frightFace(584+f*16, 64, peach)
		ghostBody(616+f*16, 64, f, white)
		frightFace(616+f*16, 64, red)
	}
	// Eyes (584..632, 80): R, L, U, D.
	for d := range 4 {
		ghostEyes(584+d*16, 80, d)
	}
	// Ghost-eaten scores at y=128.
	for i, s := range []string{"200", "400", "800", "1600"} {
		number(456+i*16, 128+5, 16, s, cyan)
	}
	// Ms. intermission: Pac-Man right (y=144) and left (y=160), 3 frames.
	for f := range 3 {
		pac(456+f*16, 144, 16, dirAngle[0], mouths[f], yellow)
		pac(456+f*16, 160, 16, dirAngle[1], mouths[f], yellow)
	}
	// Heart (488,160) -- note: shares a cell with Pac-Man left frame 2.
	rect(488, 160, 16, 16, black)
	disc(488, 160, 5, 6, 3.5, red)
	disc(488, 160, 11, 6, 3.5, red)
	for j := range 7 {
		rect(488+2+j, 160+7+j, 12-2*j, 1, red)
	}
	// Stork (2 frames, 32 wide: head half at +0, body half at +16) y=176.
	for f := range 2 {
		x := 488 + f*32
		rect(x+2, 176+7, 6, 1, orange)  // beak
		disc(x, 176, 10, 7, 2.5, white) // head
		rect(x+12, 176+7, 4, 2, white)  // neck
		rect(x+16, 176+6, 12, 5, white) // body
		if f == 0 {
			rect(x+18, 176+1, 8, 5, grey) // wing up
		} else {
			rect(x+18, 176+11, 8, 4, grey) // wing down
		}
	}
	// Bundle (488,192) and Junior (504,192).
	disc(488, 192, 8, 9, 5, rgb(80, 120, 255))
	rect(488+6, 192+2, 4, 3, rgb(80, 120, 255))
	pac(504+4, 192+4, 8, dirAngle[0], 0.5, yellow)
	rect(504+5, 192+3, 2, 2, red)
	// Clapperboard 32x32 x 3 frames at (456,208): arm closes over frames.
	for f := range 3 {
		x, y := 456+f*32, 208
		rect(x+3, y+12, 26, 17, white)
		rect(x+5, y+14, 22, 13, black)
		rect(x+7, y+17, 18, 1, white)
		rect(x+7, y+21, 12, 1, white)
		ang := []float64{-0.6, -0.3, 0}[f]
		for t := 0.0; t < 26; t += 0.3 {
			for w := 0.0; w < 3; w++ {
				px := float64(x+3) + t*math.Cos(ang)
				py := float64(y+10) - w + t*math.Sin(ang)
				set(int(px), int(py), white)
			}
		}
	}

	// ---- Classic Pac-Man (y >= 992) ----
	for d := range 4 {
		for f := range 3 {
			pac(456+f*16, 992+d*16, 16, dirAngle[d], mouths[f], yellow)
		}
	}
	// Death: 10 collapse frames + starburst at (504..664, 992).
	for f := range 10 {
		pac(504+f*16, 992, 16, dirAngle[2], 0.3+float64(f)*0.28, yellow)
	}
	for k := range 8 {
		a := float64(k) * math.Pi / 4
		for t := 3.0; t < 7; t += 0.5 {
			set(664+8+int(t*math.Cos(a)), 992+8+int(t*math.Sin(a)), yellow)
		}
	}
	// Giant Pac-Man 32x32 x 3 frames at (504,1008), facing right.
	for f := range 3 {
		pac(504+f*32, 1008, 32, dirAngle[0], mouths[f], yellow)
	}
	// Pac-Man bonus fruit (8) at (504.., 1040).
	pmFruit := []C{red, rgb(255, 60, 120), orange, rgb(255, 40, 40), green, purple, yellow, cyan}
	for i, c := range pmFruit {
		fruit(504+i*16, 1040, c)
	}
	// Act 2 nail stages (584..632, 1088).
	for i := range 4 {
		x := 584 + i*16
		rect(x+7, 1088+6, 2, 8, grey)
		rect(x+6, 1088+5, 4, 1, grey)
		switch i {
		case 1:
			rect(x+1, 1088+8, 6, 4, red)
		case 2:
			rect(x+3, 1088+9, 4, 3, red)
		case 3:
			rect(x+5, 1088+10, 2, 2, red)
		}
	}
	// Act 2 torn-sheet Blinky looking (584,1104) 2 frames; Act 3 patched Blinky (616,1104) 2 frames.
	for f := range 2 {
		x := 584 + f*16
		ghostBody(x, 1104, 0, red)
		rect(x+8, 1104+11, 5, 5, black) // torn-off tip
		rect(x+8, 1104+13, 2, 3, peach)
		ghostEyes(x, 1104, []int{2, 0}[f])

		x = 616 + f*16
		ghostBody(x, 1104, f, red)
		rect(x+6, 1104+10, 4, 2, white) // patch
		ghostEyes(x, 1104, 1)
	}
	// Act 3 dragged sheet (584,1136) 2 frames; bare Blinky (616,1136) 2 frames.
	for f := range 2 {
		x := 584 + f*16
		rect(x+1, 1136+8+f, 14, 5, red)
		rect(x+12, 1136+6, 3, 3, red)
		x = 616 + f*16
		ghostEyes(x, 1136, 0)
		rect(x+5, 1136+11, 6, 2, peach)
		rect(x+5+f, 1136+13, 2, 3, peach)
		rect(x+9-f, 1136+13, 2, 3, peach)
	}
	// Bonus fruit scores: 16-wide at y=1136, 20-wide 1000, 24-wide rows below.
	for i, s := range []string{"100", "300", "500", "700"} {
		number(456+i*16, 1136+5, 16, s, pink)
	}
	number(520, 1136+5, 20, "1000", pink)
	for i, s := range []string{"2000", "3000", "5000"} {
		number(516, 1152+i*16+5, 24, s, pink)
	}
}

// ---------- mazes ----------

const (
	Empty = iota
	Wall
	Door
	Dot
	Power
)

const (
	dotMask   uint64 = 103481868288
	powerMask uint64 = 4359202964317896252
	doorMask  uint64 = 16776960
)

func classify(m image.Image, ox, oy int) int {
	var mask uint64
	for y := oy; y < oy+8; y++ {
		for x := ox; x < ox+8; x++ {
			mask <<= 1
			r, g, b, _ := m.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				mask |= 1
			}
		}
	}
	switch mask {
	case 0:
		return Empty
	case dotMask:
		return Dot
	case powerMask:
		return Power
	case doorMask:
		return Door
	}
	return Wall
}

func drawMask(x, y int, mask uint64, c C) {
	for b := range 64 {
		if mask&(1<<(63-b)) != 0 {
			set(x+b%8, y+b/8, c)
		}
	}
}

func scale(c C, f float64) C {
	return rgb(uint8(float64(c.R)*f), uint8(float64(c.G)*f), uint8(float64(c.B)*f))
}

// openTiles flood-fills from every dot and power pellet through non-wall tiles
// (including the ghost-house door) to find the corridors actors can occupy.
// Empty tiles that are not reached are the hollow insides of wall blocks.
func openTiles(g [31][28]int) [31][28]bool {
	var open [31][28]bool
	var stack [][2]int
	for ty := range 31 {
		for tx := range 28 {
			if g[ty][tx] == Dot || g[ty][tx] == Power {
				open[ty][tx] = true
				stack = append(stack, [2]int{tx, ty})
			}
		}
	}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := p[0]+d[0], p[1]+d[1]
			if nx < 0 || ny < 0 || nx >= 28 || ny >= 31 || open[ny][nx] || g[ny][nx] == Wall {
				continue
			}
			open[ny][nx] = true
			stack = append(stack, [2]int{nx, ny})
		}
	}
	return open
}

// wallInset is how far (in pixels) wall outlines sit inside the wall tiles
// that border a corridor. Corridors are one 8px tile wide in the tile grid, so
// insetting both sides by 4px draws them 16px wide, like the arcade mazes.
const wallInset = 4.0

// wallPixels returns, for one maze, the pixels of wall tiles to light: a thin
// outline wallInset pixels (Euclidean, so corners come out rounded) away from
// the nearest corridor pixel. Wall tiles too far from any corridor to receive
// an outline are filled instead, so they still classify as walls.
func wallPixels(g [31][28]int) (line, fill [248][224]bool) {
	open := openTiles(g)
	isOpenPx := func(px, py int) bool {
		if px < 0 || py < 0 || px >= 224 || py >= 248 {
			return false
		}
		return open[py/8][px/8]
	}
	r := int(wallInset) + 1
	for ty := range 31 {
		for tx := range 28 {
			if g[ty][tx] != Wall {
				continue
			}
			any := false
			for py := ty * 8; py < ty*8+8; py++ {
				for px := tx * 8; px < tx*8+8; px++ {
					best := math.Inf(1)
					for oy := -r; oy <= r; oy++ {
						for ox := -r; ox <= r; ox++ {
							if isOpenPx(px+ox, py+oy) {
								best = min(best, math.Hypot(float64(ox), float64(oy)))
							}
						}
					}
					if best >= wallInset-0.5 && best < wallInset+0.5 {
						line[py][px] = true
						any = true
					}
				}
			}
			if !any {
				for py := ty * 8; py < ty*8+8; py++ {
					for px := tx * 8; px < tx*8+8; px++ {
						fill[py][px] = true
					}
				}
			}
		}
	}
	return line, fill
}

func drawMazes(grids [5][31][28]int) {
	wallColors := []C{rgb(255, 120, 120), rgb(110, 190, 255), rgb(255, 160, 60), rgb(190, 120, 255), blue}
	dotColor := []C{white, yellow, rgb(255, 80, 80), white, peach}
	for idx := range 5 {
		g := grids[idx]
		wc := wallColors[idx]
		line, fill := wallPixels(g)
		for col, xoff := range []int{0, 228} {
			oy := idx * 248
			for py := range 248 {
				for px := range 224 {
					if line[py][px] {
						set(xoff+px, oy+py, wc)
					} else if fill[py][px] {
						set(xoff+px, oy+py, scale(wc, 0.35))
					}
				}
			}
			for ty := range 31 {
				for tx := range 28 {
					x, y := xoff+tx*8, oy+ty*8
					switch g[ty][tx] {
					case Door:
						drawMask(x, y, doorMask, pink)
					case Dot:
						if col == 0 {
							drawMask(x, y, dotMask, dotColor[idx])
						}
					case Power:
						if col == 0 {
							drawMask(x, y, powerMask, dotColor[idx])
						}
					}
				}
			}
		}
	}
}

func savePNG(path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: genexample <out game.png> <out text.png>")
		os.Exit(2)
	}
	gameOut, textOut := os.Args[1], os.Args[2]

	grids, err := buildMazes()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	img = image.NewRGBA(image.Rect(0, 0, 680, 1240))
	rect(0, 0, 680, 1240, black)
	drawMazes(grids)
	drawSprites()
	savePNG(gameOut)

	// Verify the drawn mazes classify back to the designed tiles.
	mismatch := 0
	for idx := range 5 {
		dots := 0
		for ty := range 31 {
			for tx := range 28 {
				if classify(img, tx*8, idx*248+ty*8) != grids[idx][ty][tx] {
					mismatch++
				}
				if t := grids[idx][ty][tx]; t == Dot || t == Power {
					dots++
				}
			}
		}
		dotRows := []int{1, 36, 63, 94, 125}
		powRows := []int{2, 35, 65, 96, 127}
		fmt.Printf("maze %d: %d dots+pellets, renderer dot src=%v power src=%v\n", idx, dots,
			classify(img, 8, dotRows[idx]*8) == Dot, classify(img, 8, powRows[idx]*8) == Power)
	}
	fmt.Println("tile mismatches:", mismatch)

	// text.png
	img = image.NewRGBA(image.Rect(0, 0, 128, 224))
	rect(0, 0, 128, 224, black)
	for i := range 15 {
		glyph(i*8, 0, rune('A'+i), white)
	}
	for i := range 11 {
		glyph(i*8, 8, rune('P'+i), white)
	}
	glyph(11*8, 8, '!', white)
	for i := range 10 {
		glyph(i*8, 16, rune('0'+i), white)
	}
	glyph(80, 16, '/', white)
	glyph(88, 16, '-', white)
	glyph(96, 16, '"', white)
	savePNG(textOut)
}
