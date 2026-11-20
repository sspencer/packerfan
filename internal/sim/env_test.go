package sim_test

import (
	"math/rand"
	"sync"
	"testing"

	"pacfan/assets"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/sheet"
	"pacfan/internal/sim"
	"pacfan/internal/system"
)

var (
	testMazesOnce sync.Once
	testMazes     *component.MazeSet
	testMazesErr  error
)

func loadTestMazes(t testing.TB) *component.MazeSet {
	t.Helper()
	testMazesOnce.Do(func() {
		s, err := sheet.Load(assets.FS, assets.GameAssetFallbackPath)
		if err != nil {
			testMazesErr = err
			return
		}
		testMazes = component.ParseMazeSet(s.IsPixelLit)
	})
	if testMazesErr != nil {
		t.Fatalf("load test mazes: %v", testMazesErr)
	}
	return testMazes
}

func TestDeterminismAndSeedDivergence(t *testing.T) {
	mazes := loadTestMazes(t)

	for _, pacmanMode := range []bool{false, true} {
		name := "MsPacman"
		if pacmanMode {
			name = "Pacman"
		}
		t.Run(name, func(t *testing.T) {
			const totalTicks = 5500
			envA := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: pacmanMode, StartLevel: 1, Seed: 7})
			envB := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: pacmanMode, StartLevel: 1, Seed: 7})

			// Zoom and HighScore must not affect HashWorld.
			gsA := component.MustFindGameState(envA.World())
			gsB := component.MustFindGameState(envB.World())
			gsA.Zoom = 1
			gsA.HighScore = 0
			gsB.Zoom = 4
			gsB.HighScore = 99999

			if ha, hb := sim.HashWorld(envA.World()), sim.HashWorld(envB.World()); ha != hb {
				t.Fatalf("initial HashWorld differs: %x vs %x", ha, hb)
			}

			rng := rand.New(rand.NewSource(101))
			actions := make([]sim.Action, totalTicks)
			for i := range actions {
				if i%12 == 0 {
					actions[i] = sim.Action(rng.Intn(5))
				} else {
					actions[i] = actions[i-1]
				}
			}

			for tick := range totalTicks {
				a := actions[tick]
				termA, truncA := envA.TickOne(a)
				termB, truncB := envB.TickOne(a)
				if termA != termB || truncA != truncB {
					t.Fatalf("tick %d termination mismatch: (%v,%v) vs (%v,%v)", tick+1, termA, truncA, termB, truncB)
				}
				if ha, hb := sim.HashWorld(envA.World()), sim.HashWorld(envB.World()); ha != hb {
					t.Fatalf("tick %d HashWorld diverged: %x vs %x", tick+1, ha, hb)
				}
				if termA {
					// Reset both with same seed and continue ticking to reach 5,500+ ticks.
					seed := int64(7 + tick)
					envA.Reset(&sim.ResetOptions{Seed: &seed})
					envB.Reset(&sim.ResetOptions{Seed: &seed})
				}
			}

			// Different seeds must diverge when PRNG is exercised (e.g. Ms. Pac-Man scatter
			// random wandering or frightened mode in Pac-Man).
			env1 := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: pacmanMode, StartLevel: 1, Seed: 1})
			env2 := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: pacmanMode, StartLevel: 1, Seed: 9999})
			if pacmanMode {
				// Trigger frightened mode so classic Pac-Man ghosts sample the PRNG.
				for _, env := range []*sim.Env{env1, env2} {
					gs := component.MustFindGameState(env.World())
					gs.IsFrightMode = true
					gs.FrightFrames = 300
					for _, g := range env.GhostEntries() {
						gd := component.Ghost.Get(g)
						if gd.State == config.Scatter || gd.State == config.Chase {
							gd.State = config.Frightened
						}
					}
				}
			}
			diverged := false
			for tick := range 1000 {
				env1.TickOne(actions[tick])
				env2.TickOne(actions[tick])
				if sim.HashWorld(env1.World()) != sim.HashWorld(env2.World()) {
					diverged = true
					break
				}
			}
			if !diverged {
				t.Fatalf("expected different seeds to diverge within 1000 ticks")
			}
		})
	}
}

func TestScriptedGameResetToGameOver(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{
		Mazes:      mazes,
		PacmanMode: false,
		StartLevel: 1,
		Seed:       42,
	})

	rng := rand.New(rand.NewSource(77))
	var res sim.StepResult
	for step := range 5000 {
		legal := env.LegalActions()
		action := sim.ActionLeft
		for tries := range 4 {
			cand := (step + tries + rng.Intn(4)) % 4
			if legal[cand] {
				action = sim.Action(cand)
				break
			}
		}
		res = env.Step(action, sim.StepTile, 0)
		if res.Terminated {
			break
		}
	}

	gs := component.MustFindGameState(env.World())
	if !res.Terminated || !gs.GameOver || gs.Lives != 0 {
		t.Fatalf("expected game over: terminated=%v gameOver=%v lives=%d tick=%d", res.Terminated, gs.GameOver, gs.Lives, env.Tick())
	}
	if gs.Score <= 0 {
		t.Fatalf("expected positive score during scripted game, got %d", gs.Score)
	}
}

func TestRulesDotsAndPowerPellets(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1})
	gs := component.MustFindGameState(env.World())
	maze := component.MustFindMaze(env.World())
	initialDots := gs.DotsRemaining

	// Walk left across spawn corridor to eat at least one dot.
	var sawDot bool
	var eatenDots int
	for range 10 {
		res := env.Step(sim.ActionLeft, sim.StepTile, 0)
		if res.Events.Has(sim.EventDot) {
			sawDot = true
			eatenDots += len(res.Eaten)
			for _, c := range res.Eaten {
				if maze.GetCell(c.X, c.Y) != component.Empty {
					t.Fatalf("eaten tile %v is not Empty in maze", c)
				}
			}
		}
	}
	if !sawDot || gs.Score < config.PelletScore || gs.DotsRemaining != initialDots-eatenDots {
		t.Fatalf("dot eating failed: sawDot=%v score=%d dotsRemaining=%d initial=%d eaten=%d",
			sawDot, gs.Score, gs.DotsRemaining, initialDots, eatenDots)
	}

	// Place a power pellet directly to the right of Pac-Man and eat it.
	tr := component.Transform.Get(env.PlayerEntry())
	tile := tr.Pos.ToCenterTile()
	targetX, targetY := tile.X+1, tile.Y
	maze.SetCell(targetX, targetY, component.Power)
	tr.Pos = config.Vector{X: tile.X * config.Tile, Y: tile.Y * config.Tile}
	tr.Dir = config.Right
	scoreBefore := gs.Score

	res := env.Step(sim.ActionRight, sim.StepTile, 0)
	if !res.Events.Has(sim.EventPower) {
		t.Fatalf("expected EventPower, got mask=%b", res.Events)
	}
	if !gs.IsFrightMode || gs.FrightFrames <= 0 {
		t.Fatalf("expected IsFrightMode=true after eating power pellet")
	}
	if gs.Score-scoreBefore < config.PowerScore {
		t.Fatalf("score delta = %d, want >= %d", gs.Score-scoreBefore, config.PowerScore)
	}
}

func TestRuleGhostEatenFreezeRunsInOneStepAndEyesMove(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1})
	gs := component.MustFindGameState(env.World())

	// Enable fright mode and put frightened Blinky and already-eaten Pinky on the board.
	gs.IsFrightMode = true
	gs.FrightFrames = 300
	ghosts := env.GhostEntries()
	blinky := ghosts[component.Blinky]
	pinky := ghosts[component.Pinky]

	playerTr := component.Transform.Get(env.PlayerEntry())
	playerTr.Pos = config.Vector{X: 12 * config.Tile, Y: 23 * config.Tile}
	playerTr.Dir = config.Left

	component.Ghost.Get(blinky).State = config.Frightened
	component.Transform.Get(blinky).Pos = playerTr.Pos

	// Pinky is already eaten eyes heading back toward the ghost house.
	component.Actor.Get(pinky).IsEaten = true
	component.Ghost.Get(pinky).State = config.Eaten
	pinkyStart := config.Vector{X: 6 * config.Tile, Y: 14 * config.Tile}
	component.Transform.Get(pinky).Pos = pinkyStart
	component.Transform.Get(pinky).Dir = config.Right

	res := env.Step(sim.ActionLeft, sim.StepFrames, 1)
	if !res.Events.Has(sim.EventGhost) {
		t.Fatalf("expected EventGhost, got mask=%b", res.Events)
	}
	if res.Reward < 200 {
		t.Fatalf("expected reward >= 200 for eating ghost, got %d", res.Reward)
	}
	if res.Ticks < config.GhostEatenPause {
		t.Fatalf("expected Step to drain the %d-tick ghost freeze, got ticks=%d", config.GhostEatenPause, res.Ticks)
	}
	if gs.GhostEatenPause != 0 {
		t.Fatalf("expected GhostEatenPause == 0 after Step returns, got %d", gs.GhostEatenPause)
	}
	if got := component.Transform.Get(pinky).Pos; got == pinkyStart {
		t.Fatalf("expected eaten Pinky eyes to move during freeze, stayed at %v", got)
	}
}

func TestRuleFruitAndExtraLife(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1})
	gs := component.MustFindGameState(env.World())

	// Put score just under 10,000 and activate a 100-point fruit at player's tile.
	gs.Score = config.ExtraLifeScore - 50
	livesBefore := gs.Lives
	pos := component.Transform.Get(env.PlayerEntry()).Pos
	gs.FruitActive = true
	gs.FruitPos = pos
	gs.FruitPoints = 100
	gs.FruitTimer = 300

	res := env.Step(sim.ActionLeft, sim.StepFrames, 1)
	if !res.Events.Has(sim.EventFruit) {
		t.Fatalf("expected EventFruit, got mask=%b", res.Events)
	}
	if !res.Events.Has(sim.EventExtraLife) {
		t.Fatalf("expected EventExtraLife when crossing %d, got mask=%b", config.ExtraLifeScore, res.Events)
	}
	if gs.Lives != livesBefore+1 {
		t.Fatalf("Lives = %d, want %d", gs.Lives, livesBefore+1)
	}
}

func TestRuleDeathWithAndWithoutEpisodicLife(t *testing.T) {
	mazes := loadTestMazes(t)

	killPlayer := func(env *sim.Env) sim.StepResult {
		blinky := env.GhostEntries()[component.Blinky]
		component.Ghost.Get(blinky).State = config.Chase
		component.Transform.Get(blinky).Pos = component.Transform.Get(env.PlayerEntry()).Pos
		return env.Step(sim.ActionNoop, sim.StepFrames, 1)
	}

	t.Run("NonEpisodicLife", func(t *testing.T) {
		env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1, EpisodicLife: false})
		gs := component.MustFindGameState(env.World())

		// First death: loses 1 life, continues playing.
		res1 := killPlayer(env)
		if !res1.Events.Has(sim.EventDeath) || res1.Terminated || gs.Lives != config.InitialLives-1 {
			t.Fatalf("death 1: events=%b term=%v lives=%d", res1.Events, res1.Terminated, gs.Lives)
		}

		// Second death: loses 2nd life, continues.
		res2 := killPlayer(env)
		if !res2.Events.Has(sim.EventDeath) || res2.Terminated || gs.Lives != config.InitialLives-2 {
			t.Fatalf("death 2: events=%b term=%v lives=%d", res2.Events, res2.Terminated, gs.Lives)
		}

		// Third death: game over -> terminated.
		res3 := killPlayer(env)
		if !res3.Events.Has(sim.EventDeath) || !res3.Terminated || !gs.GameOver || gs.Lives != 0 {
			t.Fatalf("death 3: events=%b term=%v gameOver=%v lives=%d", res3.Events, res3.Terminated, gs.GameOver, gs.Lives)
		}
	})

	t.Run("EpisodicLife", func(t *testing.T) {
		env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1, EpisodicLife: true})
		gs := component.MustFindGameState(env.World())

		// Earn some score first.
		env.Step(sim.ActionLeft, sim.StepTile, 0)
		scoreBeforeDeath := gs.Score
		dotsBeforeDeath := gs.DotsRemaining

		// First death: terminated=true for the learner, lives=2.
		res1 := killPlayer(env)
		if !res1.Events.Has(sim.EventDeath) || !res1.Terminated || gs.Lives != 2 || gs.GameOver {
			t.Fatalf("episodic death 1: events=%b term=%v lives=%d gameOver=%v", res1.Events, res1.Terminated, gs.Lives, gs.GameOver)
		}

		// Reset continues the same game (score, dots, lives preserved).
		env.Reset(nil)
		if gs.Lives != 2 || gs.Score != scoreBeforeDeath || gs.DotsRemaining != dotsBeforeDeath {
			t.Fatalf("after episodic Reset: lives=%d score=%d dots=%d, want lives=2 score=%d dots=%d",
				gs.Lives, gs.Score, gs.DotsRemaining, scoreBeforeDeath, dotsBeforeDeath)
		}

		// Exhaust remaining lives -> GameOver, then Reset starts a brand new game.
		killPlayer(env)
		env.Reset(nil)
		res3 := killPlayer(env)
		if !res3.Terminated || !gs.GameOver || gs.Lives != 0 {
			t.Fatalf("final episodic death: term=%v gameOver=%v lives=%d", res3.Terminated, gs.GameOver, gs.Lives)
		}

		env.Reset(nil)
		if gs.Lives != config.InitialLives || gs.Score != 0 || env.Tick() != 0 || gs.GameOver {
			t.Fatalf("post-gameover Reset did not start fresh game: lives=%d score=%d tick=%d gameOver=%v",
				gs.Lives, gs.Score, env.Tick(), gs.GameOver)
		}
	})
}

func TestRuleLevelClearAndEndOnLevelClear(t *testing.T) {
	mazes := loadTestMazes(t)

	setupLastDot := func(env *sim.Env) {
		gs := component.MustFindGameState(env.World())
		maze := component.MustFindMaze(env.World())
		tr := component.Transform.Get(env.PlayerEntry())
		tile := tr.Pos.ToCenterTile()
		tr.Pos = config.Vector{X: tile.X * config.Tile, Y: tile.Y * config.Tile}
		tr.Dir = config.Left
		maze.SetCell(tile.X-1, tile.Y, component.Dot)
		gs.DotsRemaining = 1
	}

	// Ms. Pac-Man level 2 (maze 0) -> level 3 (maze 1).
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: false, StartLevel: 2, Seed: 1, EndOnLevelClear: false})
	setupLastDot(env)
	res := env.Step(sim.ActionLeft, sim.StepTile, 0)
	gs := component.MustFindGameState(env.World())
	if !res.Events.Has(sim.EventLevelClear) || !res.LevelChanged || res.Terminated {
		t.Fatalf("level clear: events=%b levelChanged=%v term=%v", res.Events, res.LevelChanged, res.Terminated)
	}
	if gs.Level != 3 || res.MazeIndex != 1 {
		t.Fatalf("after level clear: level=%d mazeIndex=%d, want level=3 mazeIndex=1", gs.Level, res.MazeIndex)
	}

	// With EndOnLevelClear=true, clearing the level terminates the episode.
	envEnd := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: false, StartLevel: 2, Seed: 1, EndOnLevelClear: true})
	setupLastDot(envEnd)
	resEnd := envEnd.Step(sim.ActionLeft, sim.StepTile, 0)
	if !resEnd.Events.Has(sim.EventLevelClear) || !resEnd.Terminated {
		t.Fatalf("EndOnLevelClear: events=%b term=%v", resEnd.Events, resEnd.Terminated)
	}
}

func TestRuleMaxTicksTruncation(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1, MaxTicks: 50})
	var res sim.StepResult
	for range 100 {
		res = env.Step(sim.ActionLeft, sim.StepFrames, 15)
		if res.Truncated || res.Terminated {
			break
		}
	}
	if !res.Truncated || res.Terminated {
		t.Fatalf("expected Truncated=true Terminated=false, got truncated=%v terminated=%v", res.Truncated, res.Terminated)
	}
	if env.Tick() != 50 {
		t.Fatalf("Tick() = %d, want 50", env.Tick())
	}
}

func TestRuleWallPushNeverHangsAndStepModes(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 1})

	// StepTile from spawn moves left to a centered tile.
	resTile := env.Step(sim.ActionLeft, sim.StepTile, 0)
	posAfterTile := component.Transform.Get(env.PlayerEntry()).Pos
	if !system.IsCentered(posAfterTile) {
		t.Fatalf("StepTile ended off-center at %v (ticks=%d)", posAfterTile, resTile.Ticks)
	}

	// StepIntersection advances to a centered decision tile.
	resInt := env.Step(sim.ActionLeft, sim.StepIntersection, 0)
	posAfterInt := component.Transform.Get(env.PlayerEntry()).Pos
	if !system.IsCentered(posAfterInt) || resInt.Ticks <= 0 {
		t.Fatalf("StepIntersection ended at %v ticks=%d", posAfterInt, resInt.Ticks)
	}

	// Push directly into a wall in every step mode; none may hang.
	tr := component.Transform.Get(env.PlayerEntry())
	tr.Pos = config.Vector{X: 1 * config.Tile, Y: 1 * config.Tile}
	tr.Dir = config.Left

	for _, mode := range []sim.StepMode{sim.StepFrames, sim.StepTile, sim.StepIntersection} {
		tr.Pos = config.Vector{X: 1 * config.Tile, Y: 1 * config.Tile}
		tr.Dir = config.Left
		res := env.Step(sim.ActionLeft, mode, 4)
		if res.Ticks <= 0 || res.Ticks > 64 {
			t.Fatalf("wall push in mode %v returned unexpected ticks=%d", mode, res.Ticks)
		}
	}
}

func BenchmarkEnvTick(b *testing.B) {
	mazes := loadTestMazes(b)
	env := sim.NewEnv(sim.Config{
		Mazes:      mazes,
		PacmanMode: false,
		StartLevel: 1,
		Seed:       42,
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		action := sim.Action(i % 4)
		term, _ := env.TickOne(action)
		if term {
			env.Reset(nil)
		}
	}
}

// benchAction picks a legal direction deterministically, rotating the
// preferred starting direction every call so the player explores the maze.
func benchAction(legal [config.DirCount]bool, i int) sim.Action {
	for k := range config.DirCount {
		d := (i + k) % config.DirCount
		if legal[d] {
			return sim.Action(d)
		}
	}
	return sim.ActionNoop
}

// BenchmarkEnvStep measures sim.Env.Step in each step mode with a legal-move
// policy. It reports ticks/step and ns/tick so modes can be compared.
func BenchmarkEnvStep(b *testing.B) {
	modes := []struct {
		name   string
		mode   sim.StepMode
		frames int
	}{
		{"frames4", sim.StepFrames, 4},
		{"tile", sim.StepTile, 0},
		{"intersection", sim.StepIntersection, 0},
	}
	mazes := loadTestMazes(b)
	for _, m := range modes {
		b.Run(m.name, func(b *testing.B) {
			env := sim.NewEnv(sim.Config{Mazes: mazes, StartLevel: 1, Seed: 42})
			totalTicks := 0
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				res := env.Step(benchAction(env.LegalActions(), i/8), m.mode, m.frames)
				totalTicks += res.Ticks
				if res.Terminated || res.Truncated {
					env.Reset(nil)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(totalTicks)/float64(b.N), "ticks/step")
			if totalTicks > 0 {
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(totalTicks), "ns/tick")
			}
		})
	}
}
