package sim_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"pacfan/internal/component"
	"pacfan/internal/sim"
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
