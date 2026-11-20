package scene_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/scene"
	"pacfan/internal/system"
)

type stubInput struct {
	state system.InputState
}

func (s *stubInput) Sample() system.InputState {
	return s.state
}

type recordingDrawer struct {
	worldCalls        []scene.DrawOptions
	intermissionCalls []scene.IntermissionView
}

func (d *recordingDrawer) DrawWorld(_ donburi.World, opts scene.DrawOptions) {
	d.worldCalls = append(d.worldCalls, opts)
}

func (d *recordingDrawer) DrawIntermission(_ donburi.World, view scene.IntermissionView) {
	d.intermissionCalls = append(d.intermissionCalls, view)
}

type spyScene struct {
	name    string
	log     *[]string
	onEnter func()
}

func (s *spyScene) Enter() {
	*s.log = append(*s.log, "enter:"+s.name)
	if s.onEnter != nil {
		s.onEnter()
	}
}
func (s *spyScene) Update() { *s.log = append(*s.log, "update:"+s.name) }
func (s *spyScene) Draw()   { *s.log = append(*s.log, "draw:"+s.name) }
func (s *spyScene) Exit()   { *s.log = append(*s.log, "exit:"+s.name) }

func newSceneHarness(t *testing.T, pacmanMode bool) (*scene.Context, *stubInput, *recordingDrawer, *donburi.Entry, []*donburi.Entry) {
	t.Helper()
	w := donburi.NewWorld()
	archetype.CreateGameState(w, pacmanMode, 3)
	archetype.CreateMaze(w, pacmanMode)
	archetype.CreatePRNG(w)

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

	input := &stubInput{}
	pipeline := system.NewPipeline(input)
	drawer := &recordingDrawer{}

	ctx := &scene.Context{
		World:                  w,
		Input:                  pipeline.Input(),
		Pipeline:               pipeline,
		Drawer:                 drawer,
		ReadyDuration:          5,
		RoundClearedHoldFrames: 4,
		RoundClearedFlashes:    2,
		PlayerDyingHoldFrames:  3,
		PlayerDyingFrameTicks:  2,
		PlayerDyingPostFrames:  2,
	}
	mgr := scene.NewManager(nil)
	ctx.Manager = mgr

	return ctx, input, drawer, player, ghosts
}

func TestManagerLifecycleEnterExitAndChainedTransitions(t *testing.T) {
	var log []string
	var mgr *scene.Manager

	s3 := &spyScene{name: "s3", log: &log}
	s2 := &spyScene{name: "s2", log: &log, onEnter: func() {
		mgr.Transition(s3)
	}}
	s1 := &spyScene{name: "s1", log: &log}

	mgr = scene.NewManager(s1)
	mgr.Update()
	mgr.Draw()

	mgr.Transition(s2)
	mgr.Update()
	mgr.Draw()

	want := []string{
		"enter:s1",
		"update:s1",
		"draw:s1",
		"exit:s1",
		"enter:s2",
		"exit:s2",
		"enter:s3",
		"update:s3",
		"draw:s3",
	}
	if len(log) != len(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("log[%d] = %q, want %q (full: %v)", i, log[i], want[i], log)
		}
	}
}

func TestReadyToPlayingTransition(t *testing.T) {
	ctx, input, drawer, player, _ := newSceneHarness(t, true)
	ctx.Manager.Start(scene.NewReadyScene(ctx))

	startPos := component.Transform.Get(player).Pos

	// Pause during ReadyScene holds the countdown.
	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update()
	ctx.Manager.Draw()
	if !component.MustFindGameState(ctx.World).Paused {
		t.Fatalf("expected Paused=true after Space in ReadyScene")
	}
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected to remain in ReadyScene while paused")
	}
	if len(drawer.worldCalls) == 0 || !drawer.worldCalls[len(drawer.worldCalls)-1].ShowReady {
		t.Fatalf("expected ReadyScene to draw with ShowReady=true")
	}

	// Unpause and run 4 ticks (total 4 of 5 frames): still in ReadyScene, player unmoved.
	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update()
	input.state = system.InputState{}
	for range 3 {
		ctx.Manager.Update()
	}
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected to still be in ReadyScene before duration elapses")
	}
	if got := component.Transform.Get(player).Pos; got != startPos {
		t.Fatalf("player moved during ReadyScene: got %v, want %v", got, startPos)
	}

	// 5th unpaused tick transitions to PlayingScene.
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.PlayingScene); !ok {
		t.Fatalf("expected transition to PlayingScene, got %T", ctx.Manager.Current())
	}

	// Next tick advances simulation.
	ctx.Manager.Update()
	if component.MustFindGameState(ctx.World).FrameCount != 1 {
		t.Fatalf("expected FrameCount=1 after first PlayingScene tick, got %d", component.MustFindGameState(ctx.World).FrameCount)
	}
}

func TestEatingLastDotAdvancesToNextLevelWithoutIntermission(t *testing.T) {
	ctx, _, drawer, player, _ := newSceneHarness(t, true)
	gs := component.MustFindGameState(ctx.World)
	maze := component.MustFindMaze(ctx.World)

	// Place a single final dot right at the player's position on Level 1.
	tr := component.Transform.Get(player)
	tr.Pos = config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	tr.Dir = config.Left
	component.PlayerInput.Get(player).Requested = config.Left
	maze.SetCell(10, 10, component.Dot)
	gs.DotsRemaining = 1

	ctx.Manager.Start(scene.NewPlayingScene(ctx))

	// Tick 1: player eats the last dot -> immediate transition to RoundClearedScene.
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.RoundClearedScene); !ok {
		t.Fatalf("expected transition to RoundClearedScene on last dot, got %T", ctx.Manager.Current())
	}
	if gs.Score != config.PelletScore {
		t.Fatalf("Score = %d, want %d", gs.Score, config.PelletScore)
	}

	// Hold phase draws ghosts visible and no maze flash; flash phase hides ghosts and toggles FlashMaze.
	sawHold := false
	sawFlashWhite := false
	sawFlashNormal := false
	totalRoundCleared := ctx.RoundClearedHoldFrames + 2*ctx.RoundClearedFlashes*config.RoundClearedFlashHalfFrames
	for range totalRoundCleared {
		ctx.Manager.Draw()
		opts := drawer.worldCalls[len(drawer.worldCalls)-1]
		if !opts.HideGhosts && !opts.FlashMaze {
			sawHold = true
		}
		if opts.HideGhosts && opts.FlashMaze {
			sawFlashWhite = true
		}
		if opts.HideGhosts && !opts.FlashMaze {
			sawFlashNormal = true
		}
		ctx.Manager.Update()
	}

	if !sawHold || !sawFlashWhite || !sawFlashNormal {
		t.Fatalf("expected hold + white/normal flash phases: hold=%v white=%v normal=%v", sawHold, sawFlashWhite, sawFlashNormal)
	}

	// Level 1 has no intermission -> should be in ReadyScene on Level 2 with score preserved.
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene after Level 1 clear, got %T", ctx.Manager.Current())
	}
	if gs.Level != 2 {
		t.Fatalf("Level = %d, want 2", gs.Level)
	}
	if gs.LevelComplete {
		t.Fatalf("expected LevelComplete=false on new level")
	}
	if gs.Score != config.PelletScore {
		t.Fatalf("Score lost across level advance: got %d, want %d", gs.Score, config.PelletScore)
	}
	if got, want := component.Transform.Get(player).Pos, component.Spawn.Get(player).StartPos; got != want {
		t.Fatalf("player Pos = %v, want reset to spawn %v", got, want)
	}
}

func TestClearingLevel2PlaysIntermissionThenReadyScene(t *testing.T) {
	ctx, input, drawer, player, _ := newSceneHarness(t, false) // Ms. Pac-Man mode
	gs := component.MustFindGameState(ctx.World)
	maze := component.MustFindMaze(ctx.World)

	// Start on Level 2 with 1 final power pellet remaining.
	gs.SetLevel(2)
	maze.Level = 2
	tr := component.Transform.Get(player)
	tr.Pos = config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	tr.Dir = config.Left
	component.PlayerInput.Get(player).Requested = config.Left
	maze.SetCell(10, 10, component.Power)
	gs.DotsRemaining = 1

	ctx.Manager.Start(scene.NewPlayingScene(ctx))

	// Eat final power pellet -> enters RoundClearedScene.
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.RoundClearedScene); !ok {
		t.Fatalf("expected RoundClearedScene after eating last power pellet, got %T", ctx.Manager.Current())
	}

	// Complete RoundClearedScene -> enters IntermissionScene Act 1.
	totalRoundCleared := ctx.RoundClearedHoldFrames + 2*ctx.RoundClearedFlashes*config.RoundClearedFlashHalfFrames
	for range totalRoundCleared {
		ctx.Manager.Update()
	}

	inter, ok := ctx.Manager.Current().(*scene.IntermissionScene)
	if !ok {
		t.Fatalf("expected IntermissionScene after Level 2 clear, got %T", ctx.Manager.Current())
	}
	if inter.Act() != 1 {
		t.Fatalf("intermission Act = %d, want 1", inter.Act())
	}
	if gs.Level != 3 || maze.Level != 3 {
		t.Fatalf("expected world advanced to Level 3, got gs.Level=%d maze.Level=%d", gs.Level, maze.Level)
	}
	if maze.GetIndex() != 1 {
		t.Fatalf("Ms. Pac-Man Level 3 maze index = %d, want 1", maze.GetIndex())
	}

	// Verify IntermissionScene renders and honors pause.
	ctx.Manager.Draw()
	if len(drawer.intermissionCalls) != 1 || drawer.intermissionCalls[0].Title != "ACT 1" {
		t.Fatalf("expected Act 1 intermission view, got %+v", drawer.intermissionCalls)
	}

	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update() // pauses
	input.state = system.InputState{}
	for range 10 {
		ctx.Manager.Update() // stays paused
	}
	if _, ok := ctx.Manager.Current().(*scene.IntermissionScene); !ok {
		t.Fatalf("intermission ended while paused")
	}

	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update() // unpauses and advances 1 frame
	input.state = system.InputState{}

	// Advance until intermission finishes and transitions to ReadyScene.
	for range 500 {
		if _, ok := ctx.Manager.Current().(*scene.ReadyScene); ok {
			break
		}
		ctx.Manager.Draw()
		ctx.Manager.Update()
	}
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene after IntermissionScene finished, got %T", ctx.Manager.Current())
	}
}

func TestOptionalIntermissionCanBeSkipped(t *testing.T) {
	ctx, _, _, _, _ := newSceneHarness(t, true)
	ctx.IntermissionFactory = func(_ *scene.Context, _ int) scene.Scene {
		return nil
	}
	gs := component.MustFindGameState(ctx.World)
	gs.SetLevel(2)
	gs.LevelComplete = true

	ctx.Manager.Start(scene.NewRoundClearedScene(ctx))
	totalRoundCleared := ctx.RoundClearedHoldFrames + 2*ctx.RoundClearedFlashes*config.RoundClearedFlashHalfFrames
	for range totalRoundCleared {
		ctx.Manager.Update()
	}

	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene when IntermissionFactory returns nil, got %T", ctx.Manager.Current())
	}
	if gs.Level != 3 {
		t.Fatalf("Level = %d, want 3", gs.Level)
	}
}

func TestPlayingToGhostEatenFreezeSceneAndBack(t *testing.T) {
	ctx, _, drawer, player, ghosts := newSceneHarness(t, true)
	gs := component.MustFindGameState(ctx.World)

	blinky := ghosts[0]
	component.Ghost.Get(blinky).State = config.Frightened
	gs.IsFrightMode = true
	gs.FrightFrames = 300

	pos := config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(blinky).Pos = pos

	ctx.Manager.Start(scene.NewPlayingScene(ctx))

	// Frame 0: collision happens in PlayingScene -> transitions immediately to GhostEatenFreezeScene.
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.GhostEatenFreezeScene); !ok {
		t.Fatalf("expected GhostEatenFreezeScene on ghost eat, got %T", ctx.Manager.Current())
	}
	ctx.Manager.Draw()
	if last := drawer.worldCalls[len(drawer.worldCalls)-1]; !last.HidePlayer || !last.ShowGhostScore {
		t.Fatalf("expected HidePlayer=true and ShowGhostScore=true, got %+v", last)
	}

	// Next 29 frames keep score visible (total 30 drawn frames = 0.5 seconds).
	for i := range config.GhostEatenPause - 1 {
		ctx.Manager.Update()
		if _, ok := ctx.Manager.Current().(*scene.GhostEatenFreezeScene); !ok {
			t.Fatalf("left GhostEatenFreezeScene early at frame %d", i+1)
		}
		ctx.Manager.Draw()
	}

	// Final tick expires the 0.5s freeze and returns to PlayingScene.
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.PlayingScene); !ok {
		t.Fatalf("expected return to PlayingScene after 0.5s freeze, got %T", ctx.Manager.Current())
	}
	if gs.GhostEatenPause != 0 || gs.GhostEatenEntity != donburi.Null || gs.GhostEatenScore != 0 {
		t.Fatalf("expected cleared ghost-eaten state, got pause=%d entity=%v score=%d",
			gs.GhostEatenPause, gs.GhostEatenEntity, gs.GhostEatenScore)
	}
}

func TestRealSpritesheetLevelProgressionIfPresent(t *testing.T) {
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

	ctx, _, _, _, _ := newSceneHarness(t, false) // Ms. Pac-Man
	ctx.IsPixelLit = isLit
	gs := component.MustFindGameState(ctx.World)
	maze := component.MustFindMaze(ctx.World)

	// Load Level 1 from real spritesheet (maze index 0).
	gs.DotsRemaining = maze.LoadLevel(1, isLit)
	level1Dots := gs.DotsRemaining
	if maze.GetIndex() != 0 || level1Dots == 0 {
		t.Fatalf("unexpected Level 1 setup: index=%d dots=%d", maze.GetIndex(), level1Dots)
	}

	// Clear Level 1 -> Level 2 (still maze index 0, no intermission).
	gs.DotsRemaining = 0
	gs.LevelComplete = true
	ctx.Manager.Start(scene.NewRoundClearedScene(ctx))
	totalRoundCleared := ctx.RoundClearedHoldFrames + 2*ctx.RoundClearedFlashes*config.RoundClearedFlashHalfFrames
	for range totalRoundCleared {
		ctx.Manager.Update()
	}
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene after Level 1, got %T", ctx.Manager.Current())
	}
	if gs.Level != 2 || maze.GetIndex() != 0 || gs.DotsRemaining != level1Dots {
		t.Fatalf("unexpected Level 2 state: level=%d index=%d dots=%d", gs.Level, maze.GetIndex(), gs.DotsRemaining)
	}

	// Clear Level 2 -> Intermission Act 1 -> Level 3 (maze index 1, repopulated dots).
	gs.DotsRemaining = 0
	gs.LevelComplete = true
	ctx.Manager.Start(scene.NewRoundClearedScene(ctx))
	for range totalRoundCleared {
		ctx.Manager.Update()
	}
	if _, ok := ctx.Manager.Current().(*scene.IntermissionScene); !ok {
		t.Fatalf("expected IntermissionScene after Level 2, got %T", ctx.Manager.Current())
	}
	if gs.Level != 3 || maze.GetIndex() != 1 || gs.DotsRemaining == 0 {
		t.Fatalf("unexpected Level 3 state: level=%d index=%d dots=%d", gs.Level, maze.GetIndex(), gs.DotsRemaining)
	}
}

func TestGhostEatsPlayerPlaysDeathAnimationAndRespawns(t *testing.T) {
	ctx, input, drawer, player, ghosts := newSceneHarness(t, true)
	gs := component.MustFindGameState(ctx.World)
	gs.Lives = 3
	gs.Score = 500
	gs.DotsEaten = 20
	gs.DotsRemaining = 80

	blinky := ghosts[0]
	component.Ghost.Get(blinky).State = config.Chase
	hitPos := config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = hitPos
	component.Transform.Get(blinky).Pos = config.Vector{X: hitPos.X + 4, Y: hitPos.Y}

	ctx.Manager.Start(scene.NewPlayingScene(ctx))

	// Collision frame transitions immediately into PacmanDyingScene and freezes actors.
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.PacmanDyingScene); !ok {
		t.Fatalf("expected PacmanDyingScene after ghost eats player, got %T", ctx.Manager.Current())
	}
	if got := component.Transform.Get(player).Pos; got != hitPos {
		t.Fatalf("player moved on death frame: got %v, want %v", got, hitPos)
	}

	// Space pause halts the death animation timer.
	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update()
	if !gs.Paused {
		t.Fatalf("expected pause during PacmanDyingScene")
	}
	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update() // unpauses and advances 1 frame
	input.state = system.InputState{}

	// Run through remaining death frames and verify hold -> 11 non-looping frames -> post-blank.
	seenHold := false
	seenPostBlank := false
	maxDeathFrame := -1
	totalDeath := ctx.PlayerDyingHoldFrames + config.PlayerDyingFrameCount*ctx.PlayerDyingFrameTicks + ctx.PlayerDyingPostFrames
	for range totalDeath - 1 {
		ctx.Manager.Draw()
		opts := drawer.worldCalls[len(drawer.worldCalls)-1]
		switch {
		case !opts.HideGhosts && !opts.PlayerDying && !opts.HidePlayer:
			seenHold = true
		case opts.PlayerDying && opts.HideGhosts:
			if opts.PlayerDeathFrame < maxDeathFrame {
				t.Fatalf("death animation looped backwards: %d -> %d", maxDeathFrame, opts.PlayerDeathFrame)
			}
			maxDeathFrame = opts.PlayerDeathFrame
		case opts.HidePlayer && opts.HideGhosts:
			seenPostBlank = true
		}
		ctx.Manager.Update()
	}

	if !seenHold || !seenPostBlank || maxDeathFrame != config.PlayerDyingFrameCount-1 {
		t.Fatalf("incomplete death phases: hold=%v postBlank=%v maxFrame=%d (want %d)",
			seenHold, seenPostBlank, maxDeathFrame, config.PlayerDyingFrameCount-1)
	}

	// Respawned into ReadyScene with 2 lives left, preserved dots/score, and reset positions.
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene after losing 1 life, got %T", ctx.Manager.Current())
	}
	if gs.Lives != 2 || gs.GameOver || gs.PlayerDying {
		t.Fatalf("unexpected state after respawn: Lives=%d GameOver=%v PlayerDying=%v", gs.Lives, gs.GameOver, gs.PlayerDying)
	}
	if gs.Score != 500 || gs.DotsEaten != 20 || gs.DotsRemaining != 80 {
		t.Fatalf("progress lost on respawn: Score=%d DotsEaten=%d DotsRemaining=%d", gs.Score, gs.DotsEaten, gs.DotsRemaining)
	}
	if got, want := component.Transform.Get(player).Pos, component.Spawn.Get(player).StartPos; got != want {
		t.Fatalf("player did not respawn at start: got %v, want %v", got, want)
	}
	if got, want := component.Transform.Get(blinky).Pos, component.Spawn.Get(blinky).StartPos; got != want {
		t.Fatalf("Blinky did not reset to start: got %v, want %v", got, want)
	}
}

func TestLosingLastLifeTransitionsToGameOverScene(t *testing.T) {
	ctx, _, drawer, player, ghosts := newSceneHarness(t, false)
	gs := component.MustFindGameState(ctx.World)
	gs.Lives = 1

	blinky := ghosts[0]
	component.Ghost.Get(blinky).State = config.Scatter
	pos := config.Vector{X: 12 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(blinky).Pos = pos

	ctx.Manager.Start(scene.NewPlayingScene(ctx))
	ctx.Manager.Update()

	if _, ok := ctx.Manager.Current().(*scene.PacmanDyingScene); !ok {
		t.Fatalf("expected PacmanDyingScene, got %T", ctx.Manager.Current())
	}

	totalDeath := ctx.PlayerDyingHoldFrames + config.PlayerDyingFrameCount*ctx.PlayerDyingFrameTicks + ctx.PlayerDyingPostFrames
	for range totalDeath {
		ctx.Manager.Update()
	}

	if _, ok := ctx.Manager.Current().(*scene.GameOverScene); !ok {
		t.Fatalf("expected GameOverScene after losing last life, got %T", ctx.Manager.Current())
	}
	if gs.Lives != 0 || !gs.GameOver {
		t.Fatalf("expected Lives=0 GameOver=true, got Lives=%d GameOver=%v", gs.Lives, gs.GameOver)
	}

	// GameOverScene stays frozen permanently and renders GAME OVER with actors hidden.
	frameBefore := gs.FrameCount
	for range 20 {
		ctx.Manager.Update()
		ctx.Manager.Draw()
	}
	if _, ok := ctx.Manager.Current().(*scene.GameOverScene); !ok {
		t.Fatalf("GameOverScene unexpectedly transitioned to %T", ctx.Manager.Current())
	}
	if gs.FrameCount != frameBefore {
		t.Fatalf("FrameCount advanced during GameOverScene: %d -> %d", frameBefore, gs.FrameCount)
	}
	last := drawer.worldCalls[len(drawer.worldCalls)-1]
	if !last.ShowGameOver || !last.HidePlayer || !last.HideGhosts {
		t.Fatalf("unexpected GameOver draw options: %+v", last)
	}
}
