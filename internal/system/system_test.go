package system_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/features/events"

	"pacfan/internal/archetype"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/event"
	"pacfan/internal/system"
)

type stubInput struct {
	state system.InputState
}

func (s *stubInput) Sample() system.InputState {
	return s.state
}

func newTestWorld(t *testing.T) (donburi.World, *donburi.Entry, []*donburi.Entry) {
	t.Helper()
	w := donburi.NewWorld()
	archetype.CreateGameState(w, true, 3)
	archetype.CreateMaze(w, true)
	archetype.CreatePRNG(w)

	// Open floor by default with border walls so actors can move freely in tests.
	maze := component.MustFindMaze(w)
	for y := range config.GameHeight {
		for x := range config.GameWidth {
			if y == 0 || y == config.GameHeight-1 {
				maze.SetCell(x, y, component.Wall)
			} else {
				maze.SetCell(x, y, component.Empty)
			}
		}
	}
	component.MustFindGameState(w).DotsRemaining = 10

	player := archetype.SpawnPlayer(w)
	ghosts := archetype.SpawnAllGhosts(w)
	system.RegisterEventHandlers(w)
	return w, player, ghosts
}

func TestBitIsSet(t *testing.T) {
	tests := []struct {
		n    uint32
		bit  int
		want bool
	}{
		{0, 0, false},
		{1, 0, true},
		{1, 1, false},
		{2, 1, true},
		{2, 0, false},
		{0x80000000, 31, true},
		{0x80000000, 30, false},
		{0xFFFFFFFF, 0, true},
		{0xFFFFFFFF, 31, true},
	}

	for _, tt := range tests {
		if got := system.BitIsSet(tt.n, tt.bit); got != tt.want {
			t.Errorf("BitIsSet(%v, %v) = %v, want %v", tt.n, tt.bit, got, tt.want)
		}
	}
}

func TestPausePreventsSimulationTick(t *testing.T) {
	w, player, _ := newTestWorld(t)
	input := &stubInput{state: system.InputState{PausePressed: true}}
	pipeline := system.NewPipeline(input)

	startPos := component.Transform.Get(player).Pos
	pipeline.Update(w)

	gs := component.MustFindGameState(w)
	if !gs.Paused {
		t.Fatalf("expected game to be paused after Space press")
	}
	if gs.FrameCount != 0 {
		t.Fatalf("expected FrameCount=0 while paused, got %d", gs.FrameCount)
	}
	if got := component.Transform.Get(player).Pos; got != startPos {
		t.Fatalf("expected player not to move while paused; got %v, want %v", got, startPos)
	}

	// Unpause and verify next tick advances simulation.
	pipeline.Update(w)
	if gs.Paused {
		t.Fatalf("expected game to unpause on second Space press")
	}
	if gs.FrameCount != 1 {
		t.Fatalf("expected FrameCount=1 after unpausing, got %d", gs.FrameCount)
	}
}

func TestPlayerEatsDotAndPowerUpTriggersFrightenedMode(t *testing.T) {
	w, player, ghosts := newTestWorld(t)
	maze := component.MustFindMaze(w)
	gs := component.MustFindGameState(w)

	// Place a Dot at the player's current center tile and a Power pellet to the left.
	tr := component.Transform.Get(player)
	tr.Pos = config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	tr.Dir = config.Left
	component.PlayerInput.Get(player).Requested = config.Left

	maze.SetCell(10, 10, component.Dot)
	maze.SetCell(9, 10, component.Power)
	gs.DotsRemaining = 2

	pipeline := system.NewPipeline(&stubInput{})

	// First tick: player moves 1px left (still inside center tile 10,10) and eats the Dot.
	pipeline.Update(w)
	if gs.Score != config.PelletScore {
		t.Fatalf("after dot, Score = %d, want %d", gs.Score, config.PelletScore)
	}
	if gs.HighScore != config.PelletScore {
		t.Fatalf("after dot, HighScore = %d, want %d", gs.HighScore, config.PelletScore)
	}
	if gs.DotsEaten != 1 || gs.DotsRemaining != 1 {
		t.Fatalf("after dot, DotsEaten=%d DotsRemaining=%d, want 1,1", gs.DotsEaten, gs.DotsRemaining)
	}
	if maze.GetCell(10, 10) != component.Empty {
		t.Fatalf("expected tile (10,10) to be Empty after eating dot")
	}

	// Advance until player reaches tile (9,10) and eats Power pellet.
	for range 12 {
		pipeline.Update(w)
	}

	if !gs.IsFrightMode {
		t.Fatalf("expected IsFrightMode=true after eating power pellet")
	}
	if gs.Score != config.PelletScore+config.PowerScore {
		t.Fatalf("Score = %d, want %d", gs.Score, config.PelletScore+config.PowerScore)
	}

	// Blinky (ghosts[0]) starts in Scatter and must switch to Frightened.
	blinkyGhost := component.Ghost.Get(ghosts[0])
	if blinkyGhost.State != config.Frightened {
		t.Fatalf("expected Blinky state Frightened, got %v", blinkyGhost.State)
	}
}

func TestPlayerTunnelWrapAndWallBlocking(t *testing.T) {
	w, player, _ := newTestWorld(t)
	maze := component.MustFindMaze(w)

	// Wrap left edge -> right edge.
	tr := component.Transform.Get(player)
	tr.Pos = config.Vector{X: 0, Y: 14 * config.Tile}
	tr.Dir = config.Left
	component.PlayerInput.Get(player).Requested = config.Left

	playerSys := system.NewPlayerSystem()
	playerSys.Update(w)

	if tr.Pos.X != config.GameWidth*config.Tile-1 {
		t.Fatalf("expected tunnel wrap X=%d, got %d", config.GameWidth*config.Tile-1, tr.Pos.X)
	}

	// Blocked by a wall directly ahead.
	tr.Pos = config.Vector{X: 5 * config.Tile, Y: 5 * config.Tile}
	tr.Dir = config.Right
	component.PlayerInput.Get(player).Requested = config.Right
	maze.SetCell(6, 5, component.Wall)

	playerSys.Update(w)
	if tr.Pos.X != 5*config.Tile {
		t.Fatalf("expected player to stop at wall, moved to X=%d", tr.Pos.X)
	}
}

func TestGhostChaseTargetsMatchArcadeRules(t *testing.T) {
	w, player, ghosts := newTestWorld(t)

	// Place player at tile (10, 10) facing Up (tests Pinky & Inky Up-direction bug).
	playerTr := component.Transform.Get(player)
	playerTr.Pos = config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	playerTr.Dir = config.Up

	// Ghosts are spawned in order: Blinky, Inky, Pinky, Clyde.
	blinkyEntry, inkyEntry, pinkyEntry, clydeEntry := ghosts[0], ghosts[1], ghosts[2], ghosts[3]

	// Place all ghosts at centered tile coordinates in Chase mode.
	component.Transform.Get(blinkyEntry).Pos = config.Vector{X: 12 * config.Tile, Y: 12 * config.Tile}
	component.Ghost.Get(blinkyEntry).State = config.Chase

	component.Transform.Get(inkyEntry).Pos = config.Vector{X: 14 * config.Tile, Y: 14 * config.Tile}
	component.Ghost.Get(inkyEntry).State = config.Chase

	component.Transform.Get(pinkyEntry).Pos = config.Vector{X: 16 * config.Tile, Y: 16 * config.Tile}
	component.Ghost.Get(pinkyEntry).State = config.Chase

	// Place Clyde within 8 tiles of player so Clyde switches to ScatterTarget.
	component.Transform.Get(clydeEntry).Pos = config.Vector{X: 12 * config.Tile, Y: 10 * config.Tile}
	component.Ghost.Get(clydeEntry).State = config.Chase

	ghostSys := system.NewGhostSystem()
	ghostSys.Update(w)

	if got := component.Ghost.Get(blinkyEntry).Target; got != playerTr.Pos {
		t.Errorf("Blinky target = %v, want %v", got, playerTr.Pos)
	}

	// Pinky: player tile (10,10) + (-4,-4) = (6,6) -> (48,48)
	wantPinky := config.Vector{X: 6, Y: 6}.ToPixel()
	if got := component.Ghost.Get(pinkyEntry).Target; got != wantPinky {
		t.Errorf("Pinky target = %v, want %v", got, wantPinky)
	}

	// Inky: pivot = (10-2, 10-2) = (8,8); Blinky was at tile (12,12) before moving 1px ->
	// Note Blinky runs first and moves 1px, still tile (11 or 12) depending on dir!
	// Verify Inky computed a valid reflection relative to Blinky's current tile.
	blinkyTile := component.Transform.Get(blinkyEntry).Pos.ToTile()
	pivot := config.Vector{X: 8, Y: 8}
	wantInky := config.Vector{
		X: blinkyTile.X + 2*(pivot.X-blinkyTile.X),
		Y: blinkyTile.Y + 2*(pivot.Y-blinkyTile.Y),
	}.ToPixel()
	if got := component.Ghost.Get(inkyEntry).Target; got != wantInky {
		t.Errorf("Inky target = %v, want %v", got, wantInky)
	}

	// Clyde within 8 tiles -> ScatterTarget
	wantClyde := component.Ghost.Get(clydeEntry).ScatterTarget
	if got := component.Ghost.Get(clydeEntry).Target; got != wantClyde {
		t.Errorf("Clyde close target = %v, want ScatterTarget %v", got, wantClyde)
	}
}

func TestGhostLeavesHouseAndEntersActiveWaveState(t *testing.T) {
	w, _, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)
	gs.GhostState = config.Scatter

	pinkyEntry := ghosts[2]
	pinkyGhost := component.Ghost.Get(pinkyEntry)
	if pinkyGhost.State != config.InHouse {
		t.Fatalf("expected Pinky to start InHouse, got %v", pinkyGhost.State)
	}

	ghostSys := system.NewGhostSystem()

	// Pinky leaves immediately on first tick (InHouse -> LeavingHouse).
	ghostSys.Update(w)
	if pinkyGhost.State != config.LeavingHouse {
		t.Fatalf("expected Pinky to transition to LeavingHouse, got %v", pinkyGhost.State)
	}

	// Advance enough frames for Pinky to ascend from Y=112 to Y=88 at tunnel speed.
	for range 120 {
		ghostSys.Update(w)
		if pinkyGhost.State != config.LeavingHouse {
			break
		}
	}

	if pinkyGhost.State != config.Scatter {
		t.Fatalf("expected Pinky to exit house into Scatter state, got %v", pinkyGhost.State)
	}
	pos := component.Transform.Get(pinkyEntry).Pos
	if pos.Y != 88 {
		t.Fatalf("expected Pinky exit Y=88, got %v", pos)
	}
}

func TestFrightModeExpirationRestoresGhostWaveState(t *testing.T) {
	w, _, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)

	blinky := component.Ghost.Get(ghosts[0])
	blinky.State = config.Frightened
	gs.GhostState = config.Chase
	gs.IsFrightMode = true
	gs.FrightFrames = 1
	gs.FrightFlashes = 5

	gameStateSys := system.NewGameStateSystem()
	gameStateSys.Update(w)
	events.ProcessAllEvents(w)

	if gs.IsFrightMode {
		t.Fatalf("expected IsFrightMode=false after FrightFrames expired")
	}
	if blinky.State != gs.GhostState {
		t.Fatalf("expected Blinky to revert to %v, got %v", gs.GhostState, blinky.State)
	}
}

func TestPlayerCollisionWithFrightenedGhostMarksGhostEaten(t *testing.T) {
	w, player, ghosts := newTestWorld(t)

	blinkyEntry := ghosts[0]
	blinkyGhost := component.Ghost.Get(blinkyEntry)
	blinkyGhost.State = config.Frightened

	pos := config.Vector{X: 80, Y: 80}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(blinkyEntry).Pos = pos

	collisionSys := system.NewCollisionSystem()
	collisionSys.Update(w)
	events.ProcessAllEvents(w)

	if !component.Actor.Get(blinkyEntry).IsEaten {
		t.Fatalf("expected Blinky IsEaten=true after collision while frightened")
	}
	if blinkyGhost.State != config.Eaten {
		t.Fatalf("expected Blinky State=Eaten, got %v", blinkyGhost.State)
	}
}

func TestGhostWaveChangeReversesActiveGhosts(t *testing.T) {
	w, _, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)

	blinky := component.Ghost.Get(ghosts[0])
	blinky.State = config.Scatter
	blinky.Reverse = false

	// Jump right before Scatter->Chase boundary on level 1.
	gs.FrameCount = config.SecondsToFrames(7)
	gs.GhostState = config.Scatter

	gameStateSys := system.NewGameStateSystem()
	gameStateSys.Update(w)
	events.ProcessAllEvents(w)

	if gs.GhostState != config.Chase {
		t.Fatalf("expected GameState.GhostState=Chase, got %v", gs.GhostState)
	}
	if blinky.State != config.Chase || !blinky.Reverse {
		t.Fatalf("expected Blinky state=Chase with Reverse=true, got state=%v reverse=%v", blinky.State, blinky.Reverse)
	}
}

func TestLevelCompletedMarksLevelComplete(t *testing.T) {
	w, _, _ := newTestWorld(t)
	gs := component.MustFindGameState(w)
	gs.DotsRemaining = 0

	gameStateSys := system.NewGameStateSystem()
	gameStateSys.Update(w)
	events.ProcessAllEvents(w)

	if !gs.LevelComplete {
		t.Fatalf("expected LevelComplete=true when DotsRemaining reaches 0")
	}
	if gs.Paused {
		t.Fatalf("expected user Paused flag to remain false on LevelCompleted")
	}
}

func TestWorldIsolationNoGlobalStateLeakage(t *testing.T) {
	w1, _, _ := newTestWorld(t)
	w2, _, _ := newTestWorld(t)

	event.DotEaten.Publish(w1, event.DotEatenEvent{X: 5, Y: 5})
	events.ProcessAllEvents(w1)
	events.ProcessAllEvents(w2)

	if component.MustFindGameState(w1).Score != config.PelletScore {
		t.Fatalf("w1 score = %d, want %d", component.MustFindGameState(w1).Score, config.PelletScore)
	}
	if component.MustFindGameState(w2).Score != 0 {
		t.Fatalf("w2 score leaked from w1: got %d, want 0", component.MustFindGameState(w2).Score)
	}
}

func TestMazeLoadLevelFromSpritesheetIfPresent(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "game.png")
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode spritesheet: %v", err)
	}

	isLit := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		return r > 0 || g > 0 || b > 0
	}

	// Classic Pac-Man maze has 244 edible tiles (240 dots + 4 power pellets).
	pacMaze := component.NewMazeData(true)
	pacDots := pacMaze.LoadLevel(1, isLit)
	if pacDots != 244 {
		t.Errorf("Pac-Man level 1 dots = %d, want 244", pacDots)
	}

	// Ms. Pac-Man level 1 maze has 224 edible tiles (220 dots + 4 power pellets).
	msMaze := component.NewMazeData(false)
	msDots := msMaze.LoadLevel(1, isLit)
	if msDots == 0 {
		t.Errorf("Ms. Pac-Man level 1 dots = 0, expected > 0")
	}
}

func TestEatenGhostEntersHouseRegeneratesAndLeaves(t *testing.T) {
	w, _, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)
	gs.GhostState = config.Chase

	// Build ghost house walls and doorway matching the real arcade mazes.
	maze := component.MustFindMaze(w)
	for x := 10; x <= 17; x++ {
		maze.SetCell(x, 10, component.Wall)
		maze.SetCell(x, 12, component.Wall)
		maze.SetCell(x, 16, component.Wall)
	}
	maze.SetCell(13, 12, component.Door)
	maze.SetCell(14, 12, component.Door)
	for y := 13; y <= 15; y++ {
		maze.SetCell(10, y, component.Wall)
		maze.SetCell(17, y, component.Wall)
	}

	ghostSys := system.NewGhostSystem()

	for _, entry := range ghosts {
		actor := component.Actor.Get(entry)
		tr := component.Transform.Get(entry)
		g := component.Ghost.Get(entry)

		// Simulate being eaten mid-tile along row 11 (not centered on X!).
		actor.IsEaten = true
		g.State = config.Eaten
		g.PixelsMoved = config.Tile
		tr.Pos = config.Vector{X: 8*config.Tile + 3, Y: 11 * config.Tile}
		tr.Dir = config.Right

		sawEntering := false
		sawLeaving := false
		returnedToPlay := false

		for range 400 {
			ghostSys.Update(w)

			switch g.State {
			case config.Eaten:
				if tr.Pos.X%config.Tile != 0 && tr.Pos.Y%config.Tile != 0 {
					t.Fatalf("%s eyes drifted off hallway grid at %v", actor.ID, tr.Pos)
				}
			case config.EnteringHouse:
				sawEntering = true
			case config.LeavingHouse:
				if actor.IsEaten {
					t.Fatalf("%s entered LeavingHouse while IsEaten is still true", actor.ID)
				}
				sawLeaving = true
			case config.Chase:
				if sawLeaving {
					returnedToPlay = true
				}
			}

			if returnedToPlay {
				break
			}
		}

		if !sawEntering || !sawLeaving || !returnedToPlay {
			t.Fatalf("%s lifecycle incomplete: entering=%v leaving=%v returned=%v (final state=%v pos=%v)",
				actor.ID, sawEntering, sawLeaving, returnedToPlay, g.State, tr.Pos)
		}
	}
}

func TestEatenGhostNavigationOnRealMazesIfPresent(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "game.png")
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("spritesheet %s not present: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode spritesheet: %v", err)
	}
	isLit := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		return r > 0 || g > 0 || b > 0
	}

	cases := []struct {
		name       string
		pacmanMode bool
		startPos   config.Vector
		startDir   config.Direction
	}{
		{"Pacman bottom-left mid-tile", true, config.Vector{X: 1*config.Tile + 3, Y: 29 * config.Tile}, config.Right},
		{"Pacman top-right mid-tile", true, config.Vector{X: 26*config.Tile - 2, Y: 1 * config.Tile}, config.Left},
		{"MsPacman bottom-left mid-tile", false, config.Vector{X: 1*config.Tile + 5, Y: 29 * config.Tile}, config.Right},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := donburi.NewWorld()
			archetype.CreateGameState(w, tc.pacmanMode, 3)
			archetype.CreateMaze(w, tc.pacmanMode)
			archetype.CreatePRNG(w)

			maze := component.MustFindMaze(w)
			gs := component.MustFindGameState(w)
			gs.DotsRemaining = maze.LoadLevel(1, isLit)
			gs.GhostState = config.Scatter

			archetype.SpawnPlayer(w)
			blinky := archetype.SpawnBlinky(w)
			system.RegisterEventHandlers(w)

			component.Actor.Get(blinky).IsEaten = true
			tr := component.Transform.Get(blinky)
			tr.Pos = tc.startPos
			tr.Dir = tc.startDir
			g := component.Ghost.Get(blinky)
			g.State = config.Eaten
			g.PixelsMoved = config.Tile

			ghostSys := system.NewGhostSystem()
			sawEntering := false
			sawLeaving := false
			returned := false

			for frame := range 1000 {
				ghostSys.Update(w)

				switch g.State {
				case config.Eaten:
					if tr.Dir == config.Left || tr.Dir == config.Right {
						if tr.Pos.Y%config.Tile != 0 {
							t.Fatalf("frame %d: horizontal eaten eyes off-center Y at %v", frame, tr.Pos)
						}
					} else {
						if tr.Pos.X%config.Tile != 0 {
							t.Fatalf("frame %d: vertical eaten eyes off-center X at %v", frame, tr.Pos)
						}
					}
					tile := tr.Pos.ToTile()
					if maze.InBounds(tile.X, tile.Y) && maze.GetCell(tile.X, tile.Y) == component.Wall {
						t.Fatalf("frame %d: eaten eyes clipped wall at tile %v (pos %v)", frame, tile, tr.Pos)
					}
				case config.EnteringHouse:
					sawEntering = true
				case config.LeavingHouse:
					sawLeaving = true
				case config.Scatter:
					if sawLeaving {
						returned = true
					}
				}

				if returned {
					break
				}
			}

			if !sawEntering || !sawLeaving || !returned {
				t.Fatalf("eaten ghost did not complete return cycle: entering=%v leaving=%v returned=%v (state=%v pos=%v)",
					sawEntering, sawLeaving, returned, g.State, tr.Pos)
			}
			if component.Actor.Get(blinky).IsEaten {
				t.Fatalf("expected IsEaten=false after regenerating")
			}
		})
	}
}

func TestSuccessiveGhostsEatenScoringAndPowerPelletReset(t *testing.T) {
	w, player, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)
	maze := component.MustFindMaze(w)

	// Place all ghosts outside the house in Scatter so eating a power pellet turns all 4 frightened.
	for _, entry := range ghosts {
		component.Ghost.Get(entry).State = config.Scatter
		component.Transform.Get(entry).Pos = config.Vector{X: 16, Y: 16}
	}

	// Eat a power pellet to start frightened mode.
	maze.SetCell(5, 5, component.Power)
	event.PowerUpEaten.Publish(w, event.PowerUpEatenEvent{X: 5, Y: 5})
	events.ProcessAllEvents(w)

	wantPoints := []int{200, 400, 800, 1600}
	wantTotal := config.PowerScore
	collisionSys := system.NewCollisionSystem()

	for i, entry := range ghosts {
		pos := config.Vector{X: (6 + i*2) * config.Tile, Y: 10 * config.Tile}
		component.Transform.Get(player).Pos = pos
		component.Transform.Get(entry).Pos = pos

		collisionSys.Update(w)
		events.ProcessAllEvents(w)

		wantTotal += wantPoints[i]
		if gs.NumGhostsEaten != i+1 {
			t.Fatalf("ghost %d: NumGhostsEaten = %d, want %d", i+1, gs.NumGhostsEaten, i+1)
		}
		if gs.Score != wantTotal {
			t.Fatalf("ghost %d: Score = %d, want %d", i+1, gs.Score, wantTotal)
		}
		if gs.HighScore != wantTotal {
			t.Fatalf("ghost %d: HighScore = %d, want %d", i+1, gs.HighScore, wantTotal)
		}
		if gs.GhostEatenScore != wantPoints[i] {
			t.Fatalf("ghost %d: GhostEatenScore = %d, want %d", i+1, gs.GhostEatenScore, wantPoints[i])
		}
		if gs.GhostEatenPause != config.GhostEatenPause {
			t.Fatalf("ghost %d: GhostEatenPause = %d, want %d", i+1, gs.GhostEatenPause, config.GhostEatenPause)
		}
		if gs.GhostEatenX != pos.X || gs.GhostEatenY != pos.Y {
			t.Fatalf("ghost %d: GhostEaten pos = (%d,%d), want (%d,%d)", i+1, gs.GhostEatenX, gs.GhostEatenY, pos.X, pos.Y)
		}
		if gs.GhostEatenEntity != entry.Entity() {
			t.Fatalf("ghost %d: GhostEatenEntity = %v, want %v", i+1, gs.GhostEatenEntity, entry.Entity())
		}
	}

	// Duplicate GhostEaten for an already-eaten ghost must not double-score.
	event.GhostEaten.Publish(w, event.GhostEatenEvent{
		Entity: ghosts[0].Entity(),
		ID:     component.Actor.Get(ghosts[0]).ID,
		X:      80,
		Y:      80,
	})
	events.ProcessAllEvents(w)
	if gs.Score != wantTotal || gs.NumGhostsEaten != 4 {
		t.Fatalf("duplicate GhostEaten altered state: Score=%d NumGhostsEaten=%d", gs.Score, gs.NumGhostsEaten)
	}

	// Eating a new power pellet resets the ghost-eaten streak back to 200.
	component.Actor.Get(ghosts[0]).IsEaten = false
	component.Ghost.Get(ghosts[0]).State = config.Scatter
	maze.SetCell(6, 5, component.Power)
	event.PowerUpEaten.Publish(w, event.PowerUpEatenEvent{X: 6, Y: 5})
	events.ProcessAllEvents(w)
	wantTotal += config.PowerScore

	if gs.NumGhostsEaten != 0 {
		t.Fatalf("after second power pellet: NumGhostsEaten = %d, want 0", gs.NumGhostsEaten)
	}

	pos := config.Vector{X: 120, Y: 120}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(ghosts[0]).Pos = pos
	collisionSys.Update(w)
	events.ProcessAllEvents(w)
	wantTotal += 200

	if gs.Score != wantTotal || gs.GhostEatenScore != 200 || gs.NumGhostsEaten != 1 {
		t.Fatalf("after second power pellet eat: Score=%d (want %d) GhostEatenScore=%d NumGhostsEaten=%d",
			gs.Score, wantTotal, gs.GhostEatenScore, gs.NumGhostsEaten)
	}
}

func TestGhostEatenPausesSimulationForHalfSecond(t *testing.T) {
	w, player, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)

	blinky := ghosts[0]
	component.Ghost.Get(blinky).State = config.Frightened
	gs.IsFrightMode = true
	gs.FrightFrames = 300

	pos := config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(player).Dir = config.Left
	component.PlayerInput.Get(player).Requested = config.Left
	component.Transform.Get(blinky).Pos = pos

	input := &stubInput{}
	pipeline := system.NewPipeline(input)

	// Frame 0: collision triggers GhostEatenPause = 30 (~0.5s); actors must not move this frame.
	pipeline.Update(w)
	if gs.GhostEatenPause != config.GhostEatenPause {
		t.Fatalf("after collision frame: GhostEatenPause = %d, want %d", gs.GhostEatenPause, config.GhostEatenPause)
	}
	if gs.GhostEatenScore != 200 {
		t.Fatalf("after collision frame: GhostEatenScore = %d, want 200", gs.GhostEatenScore)
	}
	if got := component.Transform.Get(player).Pos; got != pos {
		t.Fatalf("player moved during collision frame: got %v, want %v", got, pos)
	}
	if got := component.Transform.Get(blinky).Pos; got != pos {
		t.Fatalf("eaten ghost moved during collision frame: got %v, want %v", got, pos)
	}

	frozenFrameCount := gs.FrameCount
	frozenFrightFrames := gs.FrightFrames

	// User pause during ghost-eaten freeze must not decrement GhostEatenPause.
	input.state = system.InputState{PausePressed: true}
	pipeline.Update(w)
	if !gs.Paused || gs.GhostEatenPause != config.GhostEatenPause {
		t.Fatalf("while user-paused: Paused=%v GhostEatenPause=%d, want true and %d",
			gs.Paused, gs.GhostEatenPause, config.GhostEatenPause)
	}
	// Unpause (Space pressed again) -> decrements GhostEatenPause by 1.
	pipeline.Update(w)
	input.state = system.InputState{}
	if gs.Paused || gs.GhostEatenPause != config.GhostEatenPause-1 {
		t.Fatalf("after unpause: Paused=%v GhostEatenPause=%d, want false and %d",
			gs.Paused, gs.GhostEatenPause, config.GhostEatenPause-1)
	}

	// Remaining frozen frames keep popup active and gameplay frozen (total 30 drawn frames = 0.5 seconds).
	for i := range config.GhostEatenPause - 2 {
		pipeline.Update(w)
		if gs.GhostEatenPause <= 0 {
			t.Fatalf("popup ended early on frozen frame %d", i+2)
		}
		if gs.FrameCount != frozenFrameCount {
			t.Fatalf("FrameCount advanced during ghost-eaten freeze: got %d, want %d", gs.FrameCount, frozenFrameCount)
		}
		if gs.FrightFrames != frozenFrightFrames {
			t.Fatalf("FrightFrames ticked down during ghost-eaten freeze: got %d, want %d", gs.FrightFrames, frozenFrightFrames)
		}
		if got := component.Transform.Get(player).Pos; got != pos {
			t.Fatalf("player moved during ghost-eaten freeze: got %v, want %v", got, pos)
		}
		if got := component.Transform.Get(blinky).Pos; got != pos {
			t.Fatalf("eaten ghost moved during ghost-eaten freeze: got %v, want %v", got, pos)
		}
	}

	if gs.GhostEatenPause != 1 {
		t.Fatalf("before final tick: GhostEatenPause = %d, want 1", gs.GhostEatenPause)
	}

	// Next tick ends the 0.5-second freeze and immediately resumes simulation.
	pipeline.Update(w)
	if gs.GhostEatenPause != 0 {
		t.Fatalf("after 0.5s freeze: GhostEatenPause = %d, want 0", gs.GhostEatenPause)
	}
	if gs.GhostEatenEntity != donburi.Null || gs.GhostEatenScore != 0 {
		t.Fatalf("after 0.5s freeze: expected cleared popup state, got entity=%v score=%d",
			gs.GhostEatenEntity, gs.GhostEatenScore)
	}
	if gs.FrameCount != frozenFrameCount+1 {
		t.Fatalf("FrameCount did not resume after freeze: got %d, want %d", gs.FrameCount, frozenFrameCount+1)
	}
	if gs.FrightFrames != frozenFrightFrames-1 {
		t.Fatalf("FrightFrames did not resume after freeze: got %d, want %d", gs.FrightFrames, frozenFrightFrames-1)
	}
	if got := component.Transform.Get(player).Pos; got == pos {
		t.Fatalf("expected player to resume moving after freeze, stayed at %v", got)
	}
	if got := component.Transform.Get(blinky).Pos; got == pos {
		t.Fatalf("expected eaten ghost eyes to resume moving after freeze, stayed at %v", got)
	}
}

func TestAdvanceToNextLevelResetsRoundAndActors(t *testing.T) {
	w, player, ghosts := newTestWorld(t)
	gs := component.MustFindGameState(w)
	prng := component.MustFindPRNG(w)

	// Dirty all round state and actor positions.
	gs.Score = 12340
	gs.HighScore = 15000
	gs.Lives = 2
	gs.FrameCount = 500
	gs.DotsEaten = 244
	gs.DotsRemaining = 0
	gs.IsFrightMode = true
	gs.FrightFrames = 120
	gs.FrightFlashes = 5
	gs.NumGhostsEaten = 3
	gs.GhostEatenPause = 30
	gs.GhostEatenEntity = ghosts[0].Entity()
	gs.GhostEatenScore = 800
	gs.GhostState = config.Chase
	gs.LevelComplete = true
	prng.NextByte()

	component.Transform.Get(player).Pos = config.Vector{X: 40, Y: 40}
	component.Transform.Get(player).Dir = config.Down
	component.PlayerInput.Get(player).Requested = config.Right
	component.Motion.Get(player).DotPause = 3

	blinky := ghosts[0]
	component.Actor.Get(blinky).IsEaten = true
	component.Ghost.Get(blinky).State = config.Eaten
	component.Transform.Get(blinky).Pos = config.Vector{X: 24, Y: 24}

	next := system.AdvanceToNextLevel(w, nil)
	if next != 2 || gs.Level != 2 {
		t.Fatalf("AdvanceToNextLevel returned %d (gs.Level=%d), want 2", next, gs.Level)
	}
	if gs.Spec != config.GetLevelSpec(2) {
		t.Fatalf("Spec did not update to level 2")
	}
	if gs.Score != 12340 || gs.HighScore != 15000 || gs.Lives != 2 {
		t.Fatalf("persistent metrics altered: Score=%d HighScore=%d Lives=%d", gs.Score, gs.HighScore, gs.Lives)
	}
	if gs.FrameCount != 0 || gs.DotsEaten != 0 || gs.IsFrightMode || gs.NumGhostsEaten != 0 || gs.GhostEatenPause != 0 || gs.LevelComplete {
		t.Fatalf("round state not reset: %+v", gs)
	}
	if gs.GhostState != config.Scatter {
		t.Fatalf("GhostState = %v, want Scatter", gs.GhostState)
	}
	if prng.Index != 0 {
		t.Fatalf("PRNG index = %d, want 0", prng.Index)
	}
	if got, want := component.Transform.Get(player).Pos, component.Spawn.Get(player).StartPos; got != want {
		t.Fatalf("player Pos = %v, want spawn %v", got, want)
	}
	if got, want := component.PlayerInput.Get(player).Requested, component.Spawn.Get(player).StartDir; got != want {
		t.Fatalf("player Requested dir = %v, want %v", got, want)
	}
	if component.Actor.Get(blinky).IsEaten {
		t.Fatalf("expected Blinky IsEaten=false after level advance")
	}
	if got, want := component.Ghost.Get(blinky).State, component.Ghost.Get(blinky).StartState; got != want {
		t.Fatalf("Blinky State = %v, want %v", got, want)
	}
	if got, want := component.Transform.Get(blinky).Pos, component.Spawn.Get(blinky).StartPos; got != want {
		t.Fatalf("Blinky Pos = %v, want spawn %v", got, want)
	}

	// Verify level clamps cleanly at 255.
	gs.SetLevel(255)
	if got := system.AdvanceToNextLevel(w, nil); got != 255 {
		t.Fatalf("expected level to clamp at 255, got %d", got)
	}
}
