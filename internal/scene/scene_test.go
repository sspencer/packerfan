package scene_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/audio"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/event"
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
	attractCalls      []scene.AttractView
}

func (d *recordingDrawer) DrawWorld(_ donburi.World, opts scene.DrawOptions) {
	d.worldCalls = append(d.worldCalls, opts)
}

func (d *recordingDrawer) DrawIntermission(_ donburi.World, view scene.IntermissionView) {
	d.intermissionCalls = append(d.intermissionCalls, view)
}

func (d *recordingDrawer) DrawAttract(_ donburi.World, view scene.AttractView) {
	d.attractCalls = append(d.attractCalls, view)
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

func TestGameStartReadySceneTwoStageIntro(t *testing.T) {
	ctx, _, drawer, _, _ := newSceneHarness(t, true)
	ctx.PlayerOneDuration = 3
	ctx.ReadyDuration = 2
	ctx.Manager.Start(scene.NewGameStartReadyScene(ctx))

	// Stage 1 (3 frames): PLAYER ONE + READY! with actors hidden.
	for i := range 3 {
		ctx.Manager.Draw()
		opts := drawer.worldCalls[len(drawer.worldCalls)-1]
		if !opts.ShowPlayerOne || !opts.ShowReady || !opts.HidePlayer || !opts.HideGhosts {
			t.Fatalf("stage 1 frame %d: unexpected draw opts %+v", i, opts)
		}
		ctx.Manager.Update()
		if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
			t.Fatalf("left ReadyScene early during stage 1 at frame %d", i)
		}
	}

	// Stage 2 (2 frames): READY! only with actors visible.
	for i := range 2 {
		ctx.Manager.Draw()
		opts := drawer.worldCalls[len(drawer.worldCalls)-1]
		if opts.ShowPlayerOne || !opts.ShowReady || opts.HidePlayer || opts.HideGhosts {
			t.Fatalf("stage 2 frame %d: unexpected draw opts %+v", i, opts)
		}
		ctx.Manager.Update()
	}

	if _, ok := ctx.Manager.Current().(*scene.PlayingScene); !ok {
		t.Fatalf("expected PlayingScene after two-stage ready intro, got %T", ctx.Manager.Current())
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

func TestPreviouslyEatenGhostEyesKeepMovingDuringGhostEatenFreeze(t *testing.T) {
	ctx, _, _, player, ghosts := newSceneHarness(t, true)
	gs := component.MustFindGameState(ctx.World)

	// Blinky is about to be eaten; Pinky was already eaten earlier and is returning as eyes.
	blinky := ghosts[0]
	pinky := ghosts[2]
	component.Ghost.Get(blinky).State = config.Frightened
	gs.IsFrightMode = true
	gs.FrightFrames = 300

	eatPos := config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = eatPos
	component.Transform.Get(blinky).Pos = eatPos

	pinkyStart := config.Vector{X: 4 * config.Tile, Y: 10 * config.Tile}
	component.Actor.Get(pinky).IsEaten = true
	component.Ghost.Get(pinky).State = config.Eaten
	component.Transform.Get(pinky).Pos = pinkyStart
	component.Transform.Get(pinky).Dir = config.Right

	ctx.Manager.Start(scene.NewPlayingScene(ctx))
	ctx.Manager.Update() // collision -> GhostEatenFreezeScene

	if _, ok := ctx.Manager.Current().(*scene.GhostEatenFreezeScene); !ok {
		t.Fatalf("expected GhostEatenFreezeScene, got %T", ctx.Manager.Current())
	}

	// Step a few freeze frames: Blinky and player stay frozen, Pinky's eyes keep moving.
	for range 5 {
		ctx.Manager.Update()
	}
	if got := component.Transform.Get(blinky).Pos; got != eatPos {
		t.Fatalf("just-eaten Blinky moved during freeze: got %v, want %v", got, eatPos)
	}
	if got := component.Transform.Get(player).Pos; got != eatPos {
		t.Fatalf("player moved during freeze: got %v, want %v", got, eatPos)
	}
	if got := component.Transform.Get(pinky).Pos; got == pinkyStart {
		t.Fatalf("previously eaten Pinky eyes did not move during freeze (stuck at %v)", got)
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
	component.Transform.Get(blinky).Pos = config.Vector{X: hitPos.X + 3, Y: hitPos.Y}

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

func TestGameOverReturnsToAttractScreenAfterDelayOrTimeout(t *testing.T) {
	ctx, input, _, player, ghosts := newSceneHarness(t, false)
	gs := component.MustFindGameState(ctx.World)
	maze := component.MustFindMaze(ctx.World)

	// Simulate dying on Level 3 (Ms. Pac-Man maze index 1) with score & high score.
	gs.SetLevel(3)
	maze.Level = 3
	gs.Lives = 0
	gs.Score = 7500
	gs.PlayerScores[0] = 7500
	gs.HighScore = 9000
	gs.GameOver = true
	component.Transform.Get(player).Pos = config.Vector{X: 40, Y: 40}
	component.Transform.Get(ghosts[0]).Pos = config.Vector{X: 56, Y: 56}

	ctx.Manager.Start(scene.NewGameOverScene(ctx))

	// Pressing any key during the first 60 frames (< 1 second) must NOT leave GameOverScene.
	input.state = system.InputState{AnyKeyPressed: true}
	for i := range config.GameOverRestartDelay - 1 {
		ctx.Manager.Update()
		if _, ok := ctx.Manager.Current().(*scene.GameOverScene); !ok {
			t.Fatalf("left GameOverScene too early on frame %d (before 1s delay)", i+1)
		}
	}

	// At frame 61 without a key press, still in GameOverScene until timeout.
	input.state = system.InputState{AnyKeyPressed: false}
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.GameOverScene); !ok {
		t.Fatalf("left GameOverScene without key press before timeout")
	}

	// Pressing any key after 1s returns to AttractScene (where 1P/2P can be chosen).
	input.state = system.InputState{AnyKeyPressed: true}
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.AttractScene); !ok {
		t.Fatalf("expected transition to AttractScene on key press after 1s, got %T", ctx.Manager.Current())
	}

	// From AttractScene, pressing "2" starts a fresh 2-player game on Level 1 preserving HighScore.
	input.state = system.InputState{Start2PPressed: true}
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected transition to ReadyScene after selecting 2P, got %T", ctx.Manager.Current())
	}
	if gs.PlayerCount != 2 || gs.Level != 1 || maze.Level != 1 || maze.GetIndex() != 0 {
		t.Fatalf("expected 2P Level 1 (maze index 0), got players=%d level=%d mazeLevel=%d index=%d",
			gs.PlayerCount, gs.Level, maze.Level, maze.GetIndex())
	}
	if gs.Lives != config.InitialLives || gs.Score != 0 || gs.HighScore != 9000 || gs.GameOver {
		t.Fatalf("unexpected GameState after restart: Lives=%d Score=%d HighScore=%d GameOver=%v",
			gs.Lives, gs.Score, gs.HighScore, gs.GameOver)
	}
	if got, want := component.Transform.Get(player).Pos, component.Spawn.Get(player).StartPos; got != want {
		t.Fatalf("player did not reset to spawn: got %v, want %v", got, want)
	}

	// Also verify GameOverScene automatically returns to AttractScene after GameOverDuration with no input.
	input.state = system.InputState{}
	ctx.Manager.Start(scene.NewGameOverScene(ctx))
	for i := range config.GameOverDuration - 1 {
		ctx.Manager.Update()
		if _, ok := ctx.Manager.Current().(*scene.GameOverScene); !ok {
			t.Fatalf("auto-transitioned too early on frame %d", i+1)
		}
	}
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.AttractScene); !ok {
		t.Fatalf("expected automatic transition to AttractScene after GameOverDuration, got %T", ctx.Manager.Current())
	}
}

func TestAttractScene1PAnd2PStartAndDemoCycle(t *testing.T) {
	ctx, input, drawer, _, _ := newSceneHarness(t, true)
	ctx.DefaultPlayers = 2
	ctx.Manager.Start(scene.NewAttractScene(ctx))

	// Advance through cast reveal and demo chase phases.
	for range 400 {
		ctx.Manager.Draw()
		ctx.Manager.Update()
	}
	if len(drawer.attractCalls) != 400 {
		t.Fatalf("expected 400 attract draws, got %d", len(drawer.attractCalls))
	}
	last := drawer.attractCalls[len(drawer.attractCalls)-1]
	if len(last.Cast) != 4 || !last.ShowLegend || len(last.Sprites) == 0 || last.SelectedPlayer != 2 {
		t.Fatalf("unexpected attract view at frame 400: cast=%d legend=%v sprites=%d selected=%d",
			len(last.Cast), last.ShowLegend, len(last.Sprites), last.SelectedPlayer)
	}

	// Pressing "2" starts a 2-player game into the two-stage ReadyScene.
	input.state = system.InputState{Start2PPressed: true}
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene after pressing 2, got %T", ctx.Manager.Current())
	}
	gs := component.MustFindGameState(ctx.World)
	if gs.PlayerCount != 2 || gs.ActivePlayer != 0 {
		t.Fatalf("expected 2-player mode starting at Player 1, got count=%d active=%d", gs.PlayerCount, gs.ActivePlayer)
	}

	// Pressing "1" from AttractScene starts a 1-player game.
	ctx.Manager.Start(scene.NewAttractScene(ctx))
	input.state = system.InputState{Start1PPressed: true}
	ctx.Manager.Update()
	if gs.PlayerCount != 1 {
		t.Fatalf("expected 1-player mode after pressing 1, got count=%d", gs.PlayerCount)
	}
}

func TestTwoPlayerAlternatesOnDeathWithIndependentBoards(t *testing.T) {
	ctx, _, _, player, ghosts := newSceneHarness(t, true)
	maze := component.MustFindMaze(ctx.World)
	// Seed a simple board with two dots so Player 1 can eat one and leave the other.
	maze.SetCell(10, 10, component.Dot)
	maze.SetCell(11, 10, component.Dot)

	gs := component.MustFindGameState(ctx.World)
	system.StartNewGame(ctx.World, 2, nil)
	gs.DotsRemaining = 2
	gs.Lives = 2
	gs.Players[0].Lives = 2
	gs.Players[1].Lives = 1

	// Player 1 eats dot at (10,10) and earns 10 points.
	maze.SetCell(10, 10, component.Empty)
	gs.AddScore(10)
	gs.DotsEaten = 1
	gs.DotsRemaining = 1

	// Player 1 dies -> switches to Player 2 (first turn).
	blinky := ghosts[0]
	component.Ghost.Get(blinky).State = config.Chase
	pos := config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(blinky).Pos = pos

	ctx.Manager.Start(scene.NewPlayingScene(ctx))
	ctx.Manager.Update() // enters PacmanDyingScene
	totalDeath := ctx.PlayerDyingHoldFrames + config.PlayerDyingFrameCount*ctx.PlayerDyingFrameTicks + ctx.PlayerDyingPostFrames
	for range totalDeath {
		ctx.Manager.Update()
	}

	if _, ok := ctx.Manager.Current().(*scene.ReadyScene); !ok {
		t.Fatalf("expected ReadyScene for Player 2, got %T", ctx.Manager.Current())
	}
	if gs.ActivePlayer != 1 || gs.ActivePlayerName() != "PLAYER TWO" {
		t.Fatalf("expected ActivePlayer=1 (PLAYER TWO), got %d (%s)", gs.ActivePlayer, gs.ActivePlayerName())
	}
	if gs.Score != 0 || gs.PlayerScores[0] != 10 {
		t.Fatalf("expected P2 score=0 and P1 score=10, got score=%d scores=%v", gs.Score, gs.PlayerScores)
	}

	// Player 2 scores 50 points, then loses their last life -> switches back to Player 1 with P1's saved board!
	gs.AddScore(50)
	gs.Lives = 1
	component.Ghost.Get(blinky).State = config.Chase
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(blinky).Pos = pos
	ctx.Manager.Start(scene.NewPlayingScene(ctx))
	ctx.Manager.Update()
	for range totalDeath {
		ctx.Manager.Update()
	}

	if gs.ActivePlayer != 0 || gs.Score != 10 || gs.PlayerScores[1] != 50 {
		t.Fatalf("expected switch back to P1 (score 10, P2 score 50), got active=%d score=%d scores=%v",
			gs.ActivePlayer, gs.Score, gs.PlayerScores)
	}
	if maze.GetCell(10, 10) != component.Empty || maze.GetCell(11, 10) != component.Dot {
		t.Fatalf("P1 maze board was not preserved across P2 turn")
	}
}

type recordingAudio struct {
	played      []audio.SoundID
	loops       []audio.LoopID
	pauseStates []bool
	muted       bool
	stopped     int
}

func (a *recordingAudio) Play(id audio.SoundID)     { a.played = append(a.played, id) }
func (a *recordingAudio) SetLoop(loop audio.LoopID) { a.loops = append(a.loops, loop) }
func (a *recordingAudio) SetPaused(paused bool)     { a.pauseStates = append(a.pauseStates, paused) }
func (a *recordingAudio) SetMuted(muted bool)       { a.muted = muted }
func (a *recordingAudio) Muted() bool               { return a.muted }
func (a *recordingAudio) StopAll()                  { a.stopped++ }
func (a *recordingAudio) Unload()                   {}

func (a *recordingAudio) hasSound(want audio.SoundID) bool {
	for _, s := range a.played {
		if s == want {
			return true
		}
	}
	return false
}

func (a *recordingAudio) hasLoop(want audio.LoopID) bool {
	for _, l := range a.loops {
		if l == want {
			return true
		}
	}
	return false
}

func TestAudioEventsAndSceneLoopTransitions(t *testing.T) {
	ctx, input, _, player, ghosts := newSceneHarness(t, true)
	rec := &recordingAudio{}
	ctx.Audio = rec
	ctx.PlayerOneDuration = 1
	ctx.ReadyDuration = 1
	system.RegisterAudioEventHandlers(ctx.World, rec)

	// Attract -> start game plays Credit + Start jingle.
	ctx.Manager.Start(scene.NewAttractScene(ctx))
	input.state = system.InputState{Start1PPressed: true}
	ctx.Manager.Update()
	input.state = system.InputState{}
	gs := component.MustFindGameState(ctx.World)
	gs.DotsRemaining = 100
	if !rec.hasSound(audio.SoundCredit) || !rec.hasSound(audio.SoundStart) {
		t.Fatalf("expected Credit and Start sounds on game start, got %v", rec.played)
	}

	// Finish two-stage Ready -> PlayingScene activates dynamic siren loop.
	ctx.Manager.Update()
	ctx.Manager.Update()
	if _, ok := ctx.Manager.Current().(*scene.PlayingScene); !ok {
		t.Fatalf("expected PlayingScene, got %T", ctx.Manager.Current())
	}
	ctx.Manager.Update()
	if !rec.hasLoop(audio.LoopSiren0) {
		t.Fatalf("expected siren loop during PlayingScene, got %v", rec.loops)
	}

	// Eating dot, ghost, fruit, and extra life emit their respective SFX.
	gs.Score = 9990
	maze := component.MustFindMaze(ctx.World)
	maze.SetCell(1, 1, component.Dot)
	event.DotEaten.Publish(ctx.World, event.DotEatenEvent{X: 1, Y: 1})
	gs.FruitActive = true
	event.FruitEaten.Publish(ctx.World, event.FruitEatenEvent{Points: 100})
	component.Ghost.Get(ghosts[0]).State = config.Frightened
	event.GhostEaten.Publish(ctx.World, event.GhostEatenEvent{Entity: ghosts[0].Entity()})
	ctx.Pipeline.Update(ctx.World) // flushes events

	for _, want := range []audio.SoundID{audio.SoundChomp, audio.SoundEatFruit, audio.SoundEatGhost, audio.SoundExtraLife} {
		if !rec.hasSound(want) {
			t.Errorf("expected sound %v to play, played=%v", want, rec.played)
		}
	}

	// Entering PacmanDyingScene plays SoundDeath when hold frames finish.
	component.Actor.Get(player).IsEaten = true
	gs.PlayerDying = true
	ctx.Manager.Start(scene.NewPacmanDyingScene(ctx))
	for range ctx.PlayerDyingHoldFrames {
		ctx.Manager.Update()
	}
	if !rec.hasSound(audio.SoundDeath) {
		t.Fatalf("expected SoundDeath on death animation start, played=%v", rec.played)
	}

	// Entering IntermissionScene Act 2 plays SoundIntermission2.
	ctx.Manager.Start(scene.NewIntermissionScene(ctx, 2))
	if !rec.hasSound(audio.SoundIntermission2) {
		t.Fatalf("expected SoundIntermission2, played=%v", rec.played)
	}

	// 2-player turn switch ReadyScene shows banner with actors visible and does NOT replay SoundStart.
	rec.played = nil
	ctx.Manager.Start(scene.NewPlayerTurnReadyScene(ctx))
	ctx.Manager.Draw()
	if rec.hasSound(audio.SoundStart) {
		t.Fatalf("2P turn switch replayed SoundStart unexpectedly: %v", rec.played)
	}
}

func TestGhostEatenExtraLifeSoundPlaysImmediatelyOnCollisionFrame(t *testing.T) {
	ctx, _, _, player, ghosts := newSceneHarness(t, true)
	rec := &recordingAudio{}
	ctx.Audio = rec
	system.RegisterAudioEventHandlers(ctx.World, rec)

	gs := component.MustFindGameState(ctx.World)
	gs.Score = 9900
	gs.IsFrightMode = true
	gs.FrightFrames = 200

	blinky := ghosts[0]
	component.Ghost.Get(blinky).State = config.Frightened
	pos := config.Vector{X: 10 * config.Tile, Y: 10 * config.Tile}
	component.Transform.Get(player).Pos = pos
	component.Transform.Get(blinky).Pos = pos

	ctx.Manager.Start(scene.NewPlayingScene(ctx))
	ctx.Manager.Update() // Single tick: eats ghost (+200 -> 10100) and transitions to GhostEatenFreezeScene.

	if _, ok := ctx.Manager.Current().(*scene.GhostEatenFreezeScene); !ok {
		t.Fatalf("expected GhostEatenFreezeScene, got %T", ctx.Manager.Current())
	}
	if !rec.hasSound(audio.SoundEatGhost) || !rec.hasSound(audio.SoundExtraLife) {
		t.Fatalf("expected both SoundEatGhost and SoundExtraLife immediately on collision frame, played=%v", rec.played)
	}
}

func TestPauseSynchronizesAudioPauseAndResume(t *testing.T) {
	ctx, input, _, _, _ := newSceneHarness(t, true)
	rec := &recordingAudio{}
	ctx.Audio = rec
	ctx.Manager.Start(scene.NewPlayingScene(ctx))

	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update() // pauses
	if len(rec.pauseStates) == 0 || !rec.pauseStates[len(rec.pauseStates)-1] {
		t.Fatalf("expected SetPaused(true) on pause, got %v", rec.pauseStates)
	}

	input.state = system.InputState{PausePressed: true}
	ctx.Manager.Update() // unpauses
	if rec.pauseStates[len(rec.pauseStates)-1] {
		t.Fatalf("expected SetPaused(false) on unpause, got %v", rec.pauseStates)
	}

	// Pressing M toggles mute on and off.
	input.state = system.InputState{MutePressed: true}
	ctx.Manager.Update()
	if !rec.Muted() {
		t.Fatalf("expected Muted()=true after pressing M")
	}
	ctx.Manager.Update()
	if rec.Muted() {
		t.Fatalf("expected Muted()=false after pressing M again")
	}
}
