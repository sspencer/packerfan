package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// The benchmark builds the real pacfan-headless binary once and drives it over
// stdin/stdout pipes, measuring the full cost a client such as the Python
// environment pays per step (process I/O, JSON, simulation).

var (
	buildOnce sync.Once
	buildDir  string
	buildBin  string
	buildErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
	os.Exit(code)
}

func headlessBinary(b *testing.B) string {
	b.Helper()
	buildOnce.Do(func() {
		buildDir, buildErr = os.MkdirTemp("", "pacfan-headless-bench-*")
		if buildErr != nil {
			return
		}
		buildBin = filepath.Join(buildDir, "pacfan-headless")
		out, err := exec.Command("go", "build", "-o", buildBin, ".").CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build: %v\n%s", err, out)
		}
	})
	if buildErr != nil {
		b.Fatal(buildErr)
	}
	return buildBin
}

type benchStepResponse struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error"`
	Ticks      int    `json:"ticks"`
	Terminated bool   `json:"terminated"`
	Truncated  bool   `json:"truncated"`
	State      struct {
		Player struct {
			Legal [4]bool `json:"legal"`
		} `json:"player"`
	} `json:"state"`
}

// BenchmarkHeadlessProcess measures one step request/response round trip
// against a pacfan-headless subprocess in each step mode, using the same
// legal-move policy as sim.BenchmarkEnvStep.
func BenchmarkHeadlessProcess(b *testing.B) {
	bin := headlessBinary(b)
	modes := []struct{ name, req string }{
		{"frames4", `"mode":"frames","frames":4`},
		{"tile", `"mode":"tile"`},
		{"intersection", `"mode":"intersection"`},
	}
	for _, m := range modes {
		b.Run(m.name, func(b *testing.B) {
			// Use the placeholder art so results are comparable across machines.
			cmd := exec.Command(bin, "--art", filepath.Join("..", "..", "assets", "game.png"))
			stdin, err := cmd.StdinPipe()
			if err != nil {
				b.Fatal(err)
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				b.Fatal(err)
			}
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				b.Fatal(err)
			}
			defer func() {
				_, _ = io.WriteString(stdin, `{"cmd":"close"}`+"\n")
				_ = stdin.Close()
				_ = cmd.Wait()
			}()

			w := bufio.NewWriter(stdin)
			r := bufio.NewReaderSize(stdout, 64*1024)
			var resp benchStepResponse
			rpc := func(line string) {
				if _, err := w.WriteString(line); err != nil {
					b.Fatal(err)
				}
				if err := w.Flush(); err != nil {
					b.Fatal(err)
				}
				out, err := r.ReadBytes('\n')
				if err != nil {
					b.Fatal(err)
				}
				resp = benchStepResponse{}
				if err := json.Unmarshal(out, &resp); err != nil {
					b.Fatal(err)
				}
				if !resp.OK {
					b.Fatalf("server error: %s", resp.Error)
				}
			}

			reset := `{"cmd":"reset","seed":42}` + "\n"
			rpc(reset)
			totalTicks := 0
			b.ResetTimer()
			for i := range b.N {
				action := 4
				for k := range 4 {
					if d := (i/8 + k) % 4; resp.State.Player.Legal[d] {
						action = d
						break
					}
				}
				rpc(fmt.Sprintf(`{"cmd":"step","action":%d,%s}`+"\n", action, m.req))
				totalTicks += resp.Ticks
				if resp.Terminated || resp.Truncated {
					rpc(reset)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(totalTicks)/float64(b.N), "ticks/step")
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "steps/s")
		})
	}
}
