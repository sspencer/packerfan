package api_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"pacfan/assets"
	"pacfan/internal/api"
	"pacfan/internal/component"
	"pacfan/internal/sheet"
	"pacfan/internal/sim"
)

var (
	testAssetsOnce sync.Once
	testSheet      *sheet.Sheet
	testMazes      *component.MazeSet
	testAssetsErr  error
)

func loadTestAssets(t testing.TB) (*sheet.Sheet, *component.MazeSet) {
	t.Helper()
	testAssetsOnce.Do(func() {
		s, err := sheet.Load(assets.FS, assets.GameAssetFallbackPath)
		if err != nil {
			testAssetsErr = err
			return
		}
		testSheet = s
		testMazes = component.ParseMazeSet(s.IsPixelLit)
	})
	if testAssetsErr != nil {
		t.Fatalf("load test assets: %v", testAssetsErr)
	}
	return testSheet, testMazes
}

func newTestSession(t testing.TB) *api.Session {
	t.Helper()
	s, mazes := loadTestAssets(t)
	return api.NewSession(api.SessionConfig{
		Sheet: s,
		Mazes: mazes,
		Defaults: sim.Config{
			PacmanMode: false,
			StartLevel: 1,
			Seed:       7,
		},
	})
}

func TestNDJSONProtocolRoundTripAndErrors(t *testing.T) {
	s, mazes := loadTestAssets(t)
	session := newTestSession(t)
	recPath := filepath.Join(t.TempDir(), "recorded.pfr")

	requests := []string{
		`{"cmd":"hello"}`,
		`{"cmd":"step","action":1}`,
		`{"cmd":"observe","maze":true}`,
		`{not valid json`,
		`{"cmd":"bogus"}`,
		`{"cmd":"reset","pacman":true,"level":1,"seed":11,"record":"` + filepath.ToSlash(recPath) + `","meta":{"run_id":"api-test"}}`,
		`{"cmd":"step","action":99}`,
		`{"cmd":"step","action":"diagonal"}`,
		`{"cmd":"step","action":1,"mode":"frames","frames":10}`,
		`{"cmd":"step","action":"left","mode":"tile"}`,
		`{"cmd":"step","action":"noop","mode":"frames","frames":5}`,
		`{"cmd":"observe","maze":true}`,
		`{"cmd":"close"}`,
	}

	var out bytes.Buffer
	err := api.ServeNDJSON(strings.NewReader(strings.Join(requests, "\n")+"\n"), &out, session)
	if err != nil {
		t.Fatalf("ServeNDJSON: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(requests) {
		t.Fatalf("got %d responses, want %d:\n%s", len(lines), len(requests), out.String())
	}

	// 0: hello
	var hello api.HelloResponse
	if err := json.Unmarshal([]byte(lines[0]), &hello); err != nil {
		t.Fatalf("unmarshal hello: %v", err)
	}
	if !hello.OK || hello.Protocol != api.ProtocolVersion || hello.MazeHash != mazes.Hash() || len(hello.Mazes) != 5 {
		t.Fatalf("unexpected hello: %+v", hello)
	}
	if hello.Art.File != s.Name() || hello.Art.SHA256 != s.SHA256() {
		t.Fatalf("unexpected art identity: %+v", hello.Art)
	}

	// 1..4: errors (step before reset, observe before reset, malformed JSON, unknown cmd)
	for _, idx := range []int{1, 2, 3, 4, 6, 7} {
		var errResp api.ErrorResponse
		if err := json.Unmarshal([]byte(lines[idx]), &errResp); err != nil {
			t.Fatalf("line %d unmarshal error: %v", idx, err)
		}
		if errResp.OK || errResp.Error == "" {
			t.Fatalf("line %d expected error response, got %s", idx, lines[idx])
		}
	}

	// 5: reset
	var resetResp api.ResetResponse
	if err := json.Unmarshal([]byte(lines[5]), &resetResp); err != nil {
		t.Fatalf("unmarshal reset: %v", err)
	}
	if !resetResp.OK || resetResp.Tick != 0 || resetResp.MazeIndex != 4 {
		t.Fatalf("unexpected reset response: %+v", resetResp)
	}

	// 8, 9, 10: steps
	var stepResp api.StepResponse
	if err := json.Unmarshal([]byte(lines[8]), &stepResp); err != nil {
		t.Fatalf("unmarshal step 8: %v", err)
	}
	if !stepResp.OK || stepResp.Ticks != 10 || stepResp.MazeIndex != nil {
		t.Fatalf("unexpected step 8 response: %+v", stepResp)
	}

	// 11: observe
	var obsResp api.ObserveResponse
	if err := json.Unmarshal([]byte(lines[11]), &obsResp); err != nil {
		t.Fatalf("unmarshal observe: %v", err)
	}
	if !obsResp.OK || len(obsResp.Rows) != 31 {
		t.Fatalf("unexpected observe response: ok=%v rows=%d", obsResp.OK, len(obsResp.Rows))
	}

	// 12: close
	var closeResp api.CloseResponse
	if err := json.Unmarshal([]byte(lines[12]), &closeResp); err != nil {
		t.Fatalf("unmarshal close: %v", err)
	}
	if !closeResp.OK {
		t.Fatalf("expected ok=true on close")
	}

	// Verify the recorded replay file replays cleanly.
	rep, err := sim.ReadReplayFile(recPath)
	if err != nil {
		t.Fatalf("ReadReplayFile: %v", err)
	}
	if rep.TotalTicks != obsResp.Tick {
		t.Fatalf("recorded ticks = %d, want %d", rep.TotalTicks, obsResp.Tick)
	}
	fresh := sim.NewEnv(sim.Config{Mazes: mazes})
	if _, err := fresh.Replay(rep, sim.ArtInfo{File: s.Name(), SHA256: s.SHA256()}); err != nil {
		t.Fatalf("fresh.Replay: %v", err)
	}
}

func TestClientSideMazeReconstruction(t *testing.T) {
	for _, pacmanMode := range []bool{false, true} {
		name := "MsPacman"
		if pacmanMode {
			name = "Pacman"
		}
		t.Run(name, func(t *testing.T) {
			session := newTestSession(t)
			inR, inW := io.Pipe()
			outR, outW := io.Pipe()

			done := make(chan error, 1)
			go func() {
				done <- api.ServeNDJSON(inR, outW, session)
				_ = outW.Close()
			}()

			enc := json.NewEncoder(inW)
			dec := bufio.NewScanner(outR)
			dec.Buffer(make([]byte, 4096), 1024*1024)

			send := func(req any) []byte {
				t.Helper()
				if err := enc.Encode(req); err != nil {
					t.Fatalf("encode request: %v", err)
				}
				if !dec.Scan() {
					t.Fatalf("unexpected EOF reading response (err=%v)", dec.Err())
				}
				return append([]byte(nil), dec.Bytes()...)
			}

			// 1. hello -> cache all 5 static mazes client-side.
			var hello api.HelloResponse
			if err := json.Unmarshal(send(map[string]any{"cmd": "hello"}), &hello); err != nil {
				t.Fatalf("unmarshal hello: %v", err)
			}
			templates := make([][]string, len(hello.Mazes))
			for _, m := range hello.Mazes {
				templates[m.Index] = m.Rows
			}

			cloneGrid := func(idx int) [][]byte {
				rows := templates[idx]
				g := make([][]byte, len(rows))
				for y, r := range rows {
					g[y] = []byte(r)
				}
				return g
			}
			gridToRows := func(g [][]byte) []string {
				rows := make([]string, len(g))
				for y, r := range g {
					rows[y] = string(r)
				}
				return rows
			}

			// 2. reset -> select starting maze.
			var resetResp api.ResetResponse
			if err := json.Unmarshal(send(map[string]any{
				"cmd":    "reset",
				"pacman": pacmanMode,
				"level":  1,
				"seed":   42,
			}), &resetResp); err != nil {
				t.Fatalf("unmarshal reset: %v", err)
			}
			clientGrid := cloneGrid(resetResp.MazeIndex)

			// 3. Play until game over, checking client grid against observe {maze:true} on every step.
			rng := rand.New(rand.NewSource(7))
			lastLegal := resetResp.State.Player.Legal
			steps := 0
			for steps < 1500 {
				action := 1
				for tries := range 4 {
					cand := (steps + tries + rng.Intn(4)) % 4
					if lastLegal[cand] {
						action = cand
						break
					}
				}
				var stepResp api.StepResponse
				if err := json.Unmarshal(send(map[string]any{
					"cmd":    "step",
					"action": action,
					"mode":   "tile",
				}), &stepResp); err != nil {
					t.Fatalf("step %d unmarshal: %v", steps, err)
				}
				if !stepResp.OK {
					t.Fatalf("step %d not ok", steps)
				}

				if stepResp.MazeIndex != nil {
					clientGrid = cloneGrid(*stepResp.MazeIndex)
				} else {
					for _, pos := range stepResp.Eaten {
						clientGrid[pos[1]][pos[0]] = ' '
					}
				}

				var obsResp api.ObserveResponse
				if err := json.Unmarshal(send(map[string]any{
					"cmd":  "observe",
					"maze": true,
				}), &obsResp); err != nil {
					t.Fatalf("observe %d unmarshal: %v", steps, err)
				}
				if !slices.Equal(gridToRows(clientGrid), obsResp.Rows) {
					t.Fatalf("step %d (tick=%d): client-reconstructed maze diverged from observe", steps, stepResp.Tick)
				}

				lastLegal = stepResp.State.Player.Legal
				steps++
				if stepResp.Terminated || stepResp.Truncated {
					break
				}
			}
			if steps < 20 {
				t.Fatalf("game ended unexpectedly early at %d steps", steps)
			}

			send(map[string]any{"cmd": "close"})
			_ = inW.Close()
			if err := <-done; err != nil {
				t.Fatalf("ServeNDJSON error: %v", err)
			}
		})
	}
}

func BenchmarkNDJSONStep(b *testing.B) {
	session := newTestSession(b)
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	done := make(chan error, 1)
	go func() {
		done <- api.ServeNDJSON(inR, outW, session)
		_ = outW.Close()
	}()

	bw := bufio.NewWriter(inW)
	scanner := bufio.NewScanner(outR)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	roundTrip := func(line string) []byte {
		if _, err := bw.WriteString(line); err != nil {
			b.Fatalf("write: %v", err)
		}
		if err := bw.Flush(); err != nil {
			b.Fatalf("flush: %v", err)
		}
		if !scanner.Scan() {
			b.Fatalf("scan: %v", scanner.Err())
		}
		return scanner.Bytes()
	}

	roundTrip("{\"cmd\":\"reset\",\"seed\":7}\n")
	stepCmds := [4]string{
		"{\"cmd\":\"step\",\"action\":0,\"mode\":\"frames\",\"frames\":4}\n",
		"{\"cmd\":\"step\",\"action\":1,\"mode\":\"frames\",\"frames\":4}\n",
		"{\"cmd\":\"step\",\"action\":2,\"mode\":\"frames\",\"frames\":4}\n",
		"{\"cmd\":\"step\",\"action\":3,\"mode\":\"frames\",\"frames\":4}\n",
	}

	var totalRespBytes int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		resp := roundTrip(stepCmds[i&3])
		totalRespBytes += int64(len(resp))
		if bytes.Contains(resp, []byte(`"terminated":true`)) {
			roundTrip("{\"cmd\":\"reset\",\"seed\":7}\n")
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(totalRespBytes)/float64(b.N), "bytes/resp")

	roundTrip("{\"cmd\":\"close\"}\n")
	_ = inW.Close()
	if err := <-done; err != nil {
		b.Fatalf("ServeNDJSON: %v", err)
	}
}
