package sim_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pacfan/assets"
	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/sheet"
	"pacfan/internal/sim"
	"pacfan/internal/system"
)

func TestReplayRecordAndVerifyRoundTrip(t *testing.T) {
	mazes := loadTestMazes(t)
	art := sim.ArtInfo{File: "game.png", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}

	for _, pacmanMode := range []bool{false, true} {
		name := "MsPacman"
		if pacmanMode {
			name = "Pacman"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sub", "ep.pfr")
			env := sim.NewEnv(sim.Config{
				Mazes:      mazes,
				PacmanMode: pacmanMode,
				StartLevel: 1,
				Seed:       13,
			})
			meta := json.RawMessage(`{"run_id":"test-run","episode":1}`)
			rec, err := sim.NewFileRecorder(env, path, art, meta)
			if err != nil {
				t.Fatalf("NewFileRecorder: %v", err)
			}

			rng := rand.New(rand.NewSource(99))
			var perTickHashes []uint64
			for step := range 200 {
				legal := env.LegalActions()
				action := sim.ActionLeft
				for tries := range 4 {
					cand := (step + tries + rng.Intn(4)) % 4
					if legal[cand] {
						action = sim.Action(cand)
						break
					}
				}
				res := env.Step(action, sim.StepTile, 0)
				perTickHashes = append(perTickHashes, sim.HashWorld(env.World()))
				if res.Terminated {
					break
				}
			}
			wantHash := sim.HashWorld(env.World())
			wantTicks := env.Tick()
			wantScore := component.MustFindGameState(env.World()).Score

			if err := rec.Close(); err != nil {
				t.Fatalf("Recorder.Close: %v", err)
			}

			rep, err := sim.ReadReplayFile(path)
			if err != nil {
				t.Fatalf("ReadReplayFile: %v", err)
			}
			if rep.TotalTicks != wantTicks {
				t.Fatalf("rep.TotalTicks = %d, want %d", rep.TotalTicks, wantTicks)
			}
			if rep.Footer == nil || rep.Footer.Score != wantScore || rep.Footer.Ticks != wantTicks {
				t.Fatalf("unexpected footer: %+v", rep.Footer)
			}
			if len(rep.Checkpoints) == 0 {
				t.Fatalf("expected at least one hash checkpoint")
			}

			// Replay in a fresh Env -> exact final hash and zero warnings.
			fresh := sim.NewEnv(sim.Config{Mazes: mazes})
			result, err := fresh.Replay(rep, art)
			if err != nil {
				t.Fatalf("Env.Replay: %v", err)
			}
			if result.ArtSHA256Warn {
				t.Errorf("unexpected ArtSHA256Warn for identical art: %s", result.Warning)
			}
			if result.FinalHash != wantHash || result.Ticks != wantTicks || result.Score != wantScore {
				t.Fatalf("replay mismatch: got hash=%016x ticks=%d score=%d, want hash=%016x ticks=%d score=%d",
					result.FinalHash, result.Ticks, result.Score, wantHash, wantTicks, wantScore)
			}
			_ = perTickHashes

			// Art SHA-256-only mismatch warns but succeeds when maze_hash matches.
			otherArt := sim.ArtInfo{File: "game-alt.png", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
			warnRes, err := fresh.Replay(rep, otherArt)
			if err != nil {
				t.Fatalf("Env.Replay with different art SHA256 failed: %v", err)
			}
			if !warnRes.ArtSHA256Warn || warnRes.Warning == "" {
				t.Errorf("expected ArtSHA256Warn=true when art SHA256 differs")
			}

			// Maze hash mismatch is rejected outright.
			badMazeRep := *rep
			badMazeRep.Header.MazeHash = "0000000000000000000000000000000000000000000000000000000000000000"
			if _, err := fresh.Replay(&badMazeRep, art); err == nil || !strings.Contains(err.Error(), "maze_hash mismatch") {
				t.Fatalf("expected maze_hash mismatch error, got %v", err)
			}

			// Corrupted hash checkpoint reports the exact tick.
			badHashRep := *rep
			badHashRep.Checkpoints = append([]sim.HashCheckpoint(nil), rep.Checkpoints...)
			badHashRep.Checkpoints[0].Hash ^= 0xdeadbeef
			if _, err := fresh.Replay(&badHashRep, art); err == nil || !strings.Contains(err.Error(), "hash mismatch at tick") {
				t.Fatalf("expected hash mismatch error, got %v", err)
			}
		})
	}
}

func TestRecorderWritesBufferAndHandlesEmptyReplay(t *testing.T) {
	mazes := loadTestMazes(t)
	env := sim.NewEnv(sim.Config{Mazes: mazes, PacmanMode: true, StartLevel: 1, Seed: 7})

	var buf bytes.Buffer
	rec, err := sim.NewRecorder(env, &buf, nil, sim.ArtInfo{File: "game.png"}, nil)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	if err := rec.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	rep, err := sim.ReadReplay(&buf)
	if err != nil {
		t.Fatalf("ReadReplay: %v", err)
	}
	if rep.TotalTicks != 0 || rep.Footer == nil || rep.Footer.End != "closed" {
		t.Fatalf("unexpected empty replay: ticks=%d footer=%+v", rep.TotalTicks, rep.Footer)
	}
}

// greedyBFSAction picks a legal action using shortest-path BFS toward the
// nearest frightened ghost, fruit, or edible dot while avoiding dangerous ghosts.
func greedyBFSAction(env *sim.Env, avoidDist int) sim.Action {
	w := env.World()
	maze := component.MustFindMaze(w)
	gs := component.MustFindGameState(w)
	pTile := component.Transform.Get(env.PlayerEntry()).Pos.ToCenterTile()

	type dangerGhost struct {
		x, y int
	}
	var dangers []dangerGhost
	var edible [][2]int
	for _, gEntry := range env.GhostEntries() {
		g := component.Ghost.Get(gEntry)
		gt := component.Transform.Get(gEntry).Pos.ToCenterTile()
		switch g.State {
		case config.Chase, config.Scatter:
			dangers = append(dangers, dangerGhost{gt.X, gt.Y})
		case config.Frightened:
			edible = append(edible, [2]int{gt.X, gt.Y})
		}
	}

	isDanger := func(x, y int) bool {
		for _, d := range dangers {
			dx := d.x - x
			if dx < 0 {
				dx = -dx
			}
			dy := d.y - y
			if dy < 0 {
				dy = -dy
			}
			if dx+dy <= avoidDist {
				return true
			}
		}
		return false
	}

	// BFS from player tile to find first step toward best target.
	type node struct {
		x, y     int
		firstDir config.Direction
		dist     int
	}
	var visited [config.GameHeight][config.GameWidth]bool
	queue := make([]node, 0, 256)
	visited[pTile.Y][pTile.X] = true

	bestSafeDir := sim.ActionNoop
	bestSafeScore := -1
	for d := range config.DirCount {
		dir := config.Direction(d)
		if !system.CanPlayerMove(maze, pTile.X, pTile.Y, dir) {
			continue
		}
		nx, ny := stepTileWrap(pTile.X, pTile.Y, dir)
		minGhostDist := 999
		for _, dg := range dangers {
			dx := dg.x - nx
			if dx < 0 {
				dx = -dx
			}
			dy := dg.y - ny
			if dy < 0 {
				dy = -dy
			}
			if dx+dy < minGhostDist {
				minGhostDist = dx + dy
			}
		}
		if minGhostDist > bestSafeScore {
			bestSafeScore = minGhostDist
			bestSafeDir = sim.Action(dir)
		}
		if isDanger(nx, ny) {
			continue
		}
		visited[ny][nx] = true
		queue = append(queue, node{x: nx, y: ny, firstDir: dir, dist: 1})
	}

	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		// Prefer frightened ghost, then fruit, then power/dot.
		for _, eg := range edible {
			if cur.x == eg[0] && cur.y == eg[1] {
				return sim.Action(cur.firstDir)
			}
		}
		if gs.FruitActive {
			ft := gs.FruitPos.ToCenterTile()
			if cur.x == ft.X && cur.y == ft.Y {
				return sim.Action(cur.firstDir)
			}
		}
		cell := maze.GetCell(cur.x, cur.y)
		if cell == component.Dot || cell == component.Power {
			return sim.Action(cur.firstDir)
		}
		for d := range config.DirCount {
			dir := config.Direction(d)
			if !system.CanPlayerMove(maze, cur.x, cur.y, dir) {
				continue
			}
			nx, ny := stepTileWrap(cur.x, cur.y, dir)
			if visited[ny][nx] || isDanger(nx, ny) {
				continue
			}
			visited[ny][nx] = true
			queue = append(queue, node{x: nx, y: ny, firstDir: cur.firstDir, dist: cur.dist + 1})
		}
	}
	return bestSafeDir
}

func stepTileWrap(x, y int, dir config.Direction) (int, int) {
	switch dir {
	case config.Right:
		x++
	case config.Left:
		x--
	case config.Up:
		y--
	case config.Down:
		y++
	}
	if x < 0 {
		x = config.GameWidth - 1
	} else if x >= config.GameWidth {
		x = 0
	}
	return x, y
}

func TestGenerateFixtures(t *testing.T) {
	if os.Getenv("GENERATE_FIXTURES") != "1" {
		t.Skip("set GENERATE_FIXTURES=1 to regenerate testdata/*.pfr fixtures")
	}
	s, err := sheet.Load(assets.FS, assets.GameAssetFallbackPath)
	if err != nil {
		t.Fatalf("load fallback sheet: %v", err)
	}
	mazes := component.ParseMazeSet(s.IsPixelLit)
	art := sim.ArtInfo{File: s.Name(), SHA256: s.SHA256()}

	type spec struct {
		file            string
		pacman          bool
		seed            int64
		endOnLevelClear bool
		maxSteps        int
		avoidDist       int
	}
	specs := []spec{
		{"testdata/pacman_play.pfr", true, 7, false, 600, 1},
		{"testdata/mspacman_play.pfr", false, 7, false, 600, 1},
		{"testdata/mspacman_level_clear.pfr", false, 11, true, 1200, 3},
	}

	for _, sp := range specs {
		env := sim.NewEnv(sim.Config{
			Mazes:           mazes,
			PacmanMode:      sp.pacman,
			StartLevel:      1,
			Seed:            sp.seed,
			EndOnLevelClear: sp.endOnLevelClear,
		})
		rec, err := sim.NewFileRecorder(env, sp.file, art, json.RawMessage(`{"fixture":true}`))
		if err != nil {
			t.Fatalf("NewFileRecorder(%s): %v", sp.file, err)
		}
		var mask sim.EventMask
		for range sp.maxSteps {
			a := greedyBFSAction(env, sp.avoidDist)
			res := env.Step(a, sim.StepTile, 0)
			mask |= res.Events
			if res.Terminated || res.Truncated {
				break
			}
		}
		if err := rec.Close(); err != nil {
			t.Fatalf("Close(%s): %v", sp.file, err)
		}
		gs := component.MustFindGameState(env.World())
		t.Logf("generated %s: ticks=%d score=%d level=%d lives=%d events=%v",
			sp.file, env.Tick(), gs.Score, gs.Level, gs.Lives, mask.Strings())
	}
}

func TestVerifyCheckedInFixtures(t *testing.T) {
	s, err := sheet.Load(assets.FS, assets.GameAssetFallbackPath)
	if err != nil {
		t.Fatalf("load fallback sheet: %v", err)
	}
	mazes := component.ParseMazeSet(s.IsPixelLit)
	art := sim.ArtInfo{File: s.Name(), SHA256: s.SHA256()}

	fixtures, err := filepath.Glob("testdata/*.pfr")
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatalf("no checked-in .pfr fixtures found in internal/sim/testdata")
	}

	for _, path := range fixtures {
		t.Run(filepath.Base(path), func(t *testing.T) {
			rep, err := sim.ReadReplayFile(path)
			if err != nil {
				t.Fatalf("ReadReplayFile(%s): %v", path, err)
			}
			env := sim.NewEnv(sim.Config{Mazes: mazes})
			res, err := env.Replay(rep, art)
			if err != nil {
				t.Fatalf("Replay(%s) failed: %v", path, err)
			}
			if res.ArtSHA256Warn {
				t.Errorf("unexpected art SHA256 warning on tracked fixture: %s", res.Warning)
			}
			if rep.Footer != nil {
				if res.Ticks != rep.Footer.Ticks || res.Score != rep.Footer.Score || res.Level != rep.Footer.Level {
					t.Errorf("footer mismatch: got ticks=%d score=%d level=%d, want %+v",
						res.Ticks, res.Score, res.Level, *rep.Footer)
				}
			}
		})
	}
}
