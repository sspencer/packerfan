package sim

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pacfan/internal/component"
)

const (
	// ReplayFormatVersion is the .pfr schema version written in the header.
	ReplayFormatVersion = 1
	// DefaultHashCheckpointInterval is the tick cadence for HashWorld checkpoints.
	DefaultHashCheckpointInterval = 600
	maxRunsPerActionLine          = 16
)

// ArtInfo identifies the artwork PNG associated with a replay.
type ArtInfo struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// ReplayOptions captures episode configuration flags stored in a replay header.
type ReplayOptions struct {
	EpisodicLife    bool `json:"episodic_life,omitempty"`
	EndOnLevelClear bool `json:"end_on_level_clear,omitempty"`
	MaxTicks        int  `json:"max_ticks,omitempty"`
}

// ReplayHeader is line 1 of a .pfr replay file.
type ReplayHeader struct {
	PFR      int             `json:"pfr"`
	Art      ArtInfo         `json:"art"`
	MazeHash string          `json:"maze_hash"`
	Pacman   bool            `json:"pacman"`
	Level    int             `json:"level"`
	Seed     int64           `json:"seed"`
	Options  ReplayOptions   `json:"options"`
	Meta     json.RawMessage `json:"meta,omitempty"`
}

// ReplayFooter is the final JSON line of a .pfr replay file.
type ReplayFooter struct {
	End    string `json:"end"`
	Ticks  int    `json:"ticks"`
	Score  int    `json:"score"`
	Level  int    `json:"level"`
	Deaths int    `json:"deaths"`
}

// ActionRun is a run-length encoded sequence of identical actions (<action>x<count>).
type ActionRun struct {
	Action Action
	Count  int
}

// HashCheckpoint records an expected HashWorld value at a specific tick.
type HashCheckpoint struct {
	Tick int
	Hash uint64
}

// Replay is an in-memory representation of a parsed .pfr file.
type Replay struct {
	Header      ReplayHeader
	Runs        []ActionRun
	Checkpoints []HashCheckpoint
	Footer      *ReplayFooter
	TotalTicks  int
}

// ReplayResult summarizes the outcome of replaying a .pfr file through Env.Replay.
type ReplayResult struct {
	Ticks         int
	Score         int
	Level         int
	FinalHash     uint64
	CheckpointsOK int
	ArtSHA256Warn bool
	Warning       string
}

// Recorder streams a .pfr replay file while attached to an Env.
type Recorder struct {
	env          *Env
	w            *bufio.Writer
	closer       io.Closer
	hashInterval int

	hasRun     bool
	runAction  Action
	runCount   int
	pendingRow []ActionRun

	ticks          int
	deaths         int
	lastCheckpoint int
	endReason      string
	closed         bool
	err            error
}

// NewRecorder writes the replay header to w and attaches the recorder to env's tick hook.
func NewRecorder(env *Env, w io.Writer, closer io.Closer, art ArtInfo, meta json.RawMessage) (*Recorder, error) {
	cfg := env.Config()
	var mazeHash string
	if cfg.Mazes != nil {
		mazeHash = cfg.Mazes.Hash()
	}

	header := ReplayHeader{
		PFR:      ReplayFormatVersion,
		Art:      art,
		MazeHash: mazeHash,
		Pacman:   cfg.PacmanMode,
		Level:    cfg.StartLevel,
		Seed:     cfg.Seed,
		Options: ReplayOptions{
			EpisodicLife:    cfg.EpisodicLife,
			EndOnLevelClear: cfg.EndOnLevelClear,
			MaxTicks:        cfg.MaxTicks,
		},
		Meta: meta,
	}

	bw := bufio.NewWriterSize(w, 8*1024)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(header); err != nil {
		if closer != nil {
			_ = closer.Close()
		}
		return nil, fmt.Errorf("replay: write header: %w", err)
	}

	rec := &Recorder{
		env:          env,
		w:            bw,
		closer:       closer,
		hashInterval: DefaultHashCheckpointInterval,
		pendingRow:   make([]ActionRun, 0, maxRunsPerActionLine),
	}
	env.SetTickHook(rec.onTick)
	return rec, nil
}

// NewFileRecorder creates path (including parent directories) and starts recording env to it.
func NewFileRecorder(env *Env, path string, art ArtInfo, meta json.RawMessage) (*Recorder, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("replay: mkdir %q: %w", dir, err)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("replay: create %q: %w", path, err)
	}
	return NewRecorder(env, f, f, art, meta)
}

func (r *Recorder) onTick(info TickInfo) {
	if r.closed || r.err != nil {
		return
	}

	r.ticks = info.Tick
	if info.Died {
		r.deaths++
	}

	if r.hasRun && r.runAction == info.Action {
		r.runCount++
	} else {
		if r.hasRun {
			r.pushRun(ActionRun{Action: r.runAction, Count: r.runCount})
		}
		r.hasRun = true
		r.runAction = info.Action
		r.runCount = 1
	}

	if r.hashInterval > 0 && info.Tick%r.hashInterval == 0 {
		r.flushRuns()
		r.writeCheckpoint(info.Tick)
	}

	if info.Terminated {
		gs := component.MustFindGameState(r.env.World())
		switch {
		case gs.GameOver:
			r.endReason = "game_over"
		case info.LevelCleared:
			r.endReason = "level_clear"
		case info.Died:
			r.endReason = "life_lost"
		default:
			r.endReason = "terminated"
		}
	} else if info.Truncated {
		r.endReason = "truncated"
	}
}

func (r *Recorder) pushRun(run ActionRun) {
	r.pendingRow = append(r.pendingRow, run)
	if len(r.pendingRow) >= maxRunsPerActionLine {
		r.flushActionLine()
	}
}

func (r *Recorder) flushRuns() {
	if r.hasRun {
		r.pendingRow = append(r.pendingRow, ActionRun{Action: r.runAction, Count: r.runCount})
		r.hasRun = false
		r.runCount = 0
	}
	r.flushActionLine()
}

func (r *Recorder) flushActionLine() {
	if len(r.pendingRow) == 0 || r.err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteByte('a')
	for _, run := range r.pendingRow {
		sb.WriteByte(' ')
		sb.WriteString(strconv.Itoa(int(run.Action)))
		sb.WriteByte('x')
		sb.WriteString(strconv.Itoa(run.Count))
	}
	sb.WriteByte('\n')
	if _, err := r.w.WriteString(sb.String()); err != nil {
		r.err = err
	}
	r.pendingRow = r.pendingRow[:0]
}

func (r *Recorder) writeCheckpoint(tick int) {
	if r.err != nil {
		return
	}
	h := HashWorld(r.env.World())
	if _, err := fmt.Fprintf(r.w, "h %d %016x\n", tick, h); err != nil {
		r.err = err
		return
	}
	r.lastCheckpoint = tick
}

// Close flushes remaining action runs, writes a final hash checkpoint (if the
// final tick was not already checkpointed) and footer, and closes the writer.
func (r *Recorder) Close() error {
	if r.closed {
		return r.err
	}
	r.closed = true
	r.env.SetTickHook(nil)

	r.flushRuns()
	if r.ticks > 0 && r.ticks != r.lastCheckpoint {
		r.writeCheckpoint(r.ticks)
	}

	if r.err == nil {
		gs := component.MustFindGameState(r.env.World())
		end := r.endReason
		if end == "" {
			end = "closed"
		}
		footer := ReplayFooter{
			End:    end,
			Ticks:  r.ticks,
			Score:  gs.Score,
			Level:  gs.Level,
			Deaths: r.deaths,
		}
		enc := json.NewEncoder(r.w)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(footer); err != nil {
			r.err = err
		}
	}

	if r.err == nil {
		r.err = r.w.Flush()
	}
	if r.closer != nil {
		if cerr := r.closer.Close(); cerr != nil && r.err == nil {
			r.err = cerr
		}
	}
	return r.err
}

// ReadReplay parses a .pfr replay stream.
func ReadReplay(r io.Reader) (*Replay, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	var rep Replay
	headerRead := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !headerRead {
			if err := json.Unmarshal([]byte(line), &rep.Header); err != nil {
				return nil, fmt.Errorf("replay line %d: invalid header: %w", lineNum, err)
			}
			if rep.Header.PFR != ReplayFormatVersion {
				return nil, fmt.Errorf("replay line %d: unsupported pfr version %d (want %d)", lineNum, rep.Header.PFR, ReplayFormatVersion)
			}
			headerRead = true
			continue
		}

		switch {
		case strings.HasPrefix(line, "a "):
			fields := strings.Fields(line[2:])
			for _, tok := range fields {
				actStr, cntStr, ok := strings.Cut(tok, "x")
				if !ok {
					return nil, fmt.Errorf("replay line %d: invalid action run %q", lineNum, tok)
				}
				actNum, err := strconv.Atoi(actStr)
				if err != nil || actNum < 0 || actNum > int(ActionNoop) {
					return nil, fmt.Errorf("replay line %d: invalid action %q", lineNum, actStr)
				}
				cnt, err := strconv.Atoi(cntStr)
				if err != nil || cnt <= 0 {
					return nil, fmt.Errorf("replay line %d: invalid run count %q", lineNum, cntStr)
				}
				rep.Runs = append(rep.Runs, ActionRun{Action: Action(actNum), Count: cnt})
				rep.TotalTicks += cnt
			}

		case strings.HasPrefix(line, "h "):
			fields := strings.Fields(line[2:])
			if len(fields) != 2 {
				return nil, fmt.Errorf("replay line %d: invalid hash checkpoint %q", lineNum, line)
			}
			tick, err := strconv.Atoi(fields[0])
			if err != nil || tick < 0 {
				return nil, fmt.Errorf("replay line %d: invalid checkpoint tick %q", lineNum, fields[0])
			}
			hash, err := strconv.ParseUint(fields[1], 16, 64)
			if err != nil {
				return nil, fmt.Errorf("replay line %d: invalid checkpoint hash %q", lineNum, fields[1])
			}
			rep.Checkpoints = append(rep.Checkpoints, HashCheckpoint{Tick: tick, Hash: hash})

		case strings.HasPrefix(line, "{"):
			var footer ReplayFooter
			if err := json.Unmarshal([]byte(line), &footer); err != nil {
				return nil, fmt.Errorf("replay line %d: invalid footer: %w", lineNum, err)
			}
			rep.Footer = &footer

		default:
			return nil, fmt.Errorf("replay line %d: unexpected record %q", lineNum, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("replay: scan: %w", err)
	}
	if !headerRead {
		return nil, fmt.Errorf("replay: empty file (missing header)")
	}
	return &rep, nil
}

// ReadReplayFile reads and parses a .pfr file from disk.
func ReadReplayFile(path string) (*Replay, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("replay: open %q: %w", path, err)
	}
	defer f.Close()
	return ReadReplay(f)
}

// Replay verifies r against e's MazeSet and replays every recorded tick,
// checking every hash checkpoint. A maze_hash mismatch is rejected outright;
// an art.sha256-only mismatch sets ReplayResult.ArtSHA256Warn.
func (e *Env) Replay(r *Replay, currentArt ArtInfo) (ReplayResult, error) {
	var res ReplayResult
	if r == nil {
		return res, fmt.Errorf("replay: nil replay")
	}

	if e.cfg.Mazes != nil && r.Header.MazeHash != "" && e.cfg.Mazes.Hash() != r.Header.MazeHash {
		return res, fmt.Errorf(
			"replay maze_hash mismatch: replay requires %s (maze_hash=%s), but loaded %s (maze_hash=%s)",
			r.Header.Art.File, r.Header.MazeHash, currentArt.File, e.cfg.Mazes.Hash(),
		)
	}

	if r.Header.Art.SHA256 != "" && currentArt.SHA256 != "" && r.Header.Art.SHA256 != currentArt.SHA256 {
		res.ArtSHA256Warn = true
		res.Warning = fmt.Sprintf(
			"artwork SHA-256 differs (replay=%s %s, loaded=%s %s) but maze_hash matches",
			r.Header.Art.File, r.Header.Art.SHA256[:min(12, len(r.Header.Art.SHA256))],
			currentArt.File, currentArt.SHA256[:min(12, len(currentArt.SHA256))],
		)
	}

	pacman := r.Header.Pacman
	level := r.Header.Level
	seed := r.Header.Seed
	episodic := r.Header.Options.EpisodicLife
	endClear := r.Header.Options.EndOnLevelClear
	maxTicks := r.Header.Options.MaxTicks

	e.Reset(&ResetOptions{
		PacmanMode:      &pacman,
		StartLevel:      &level,
		Seed:            &seed,
		EpisodicLife:    &episodic,
		EndOnLevelClear: &endClear,
		MaxTicks:        &maxTicks,
		ForceFullReset:  true,
	})

	cpMap := make(map[int]uint64, len(r.Checkpoints))
	for _, cp := range r.Checkpoints {
		cpMap[cp.Tick] = cp.Hash
	}

	if want, ok := cpMap[0]; ok {
		got := HashWorld(e.world)
		if got != want {
			return res, fmt.Errorf("replay hash mismatch at tick 0: got %016x, want %016x", got, want)
		}
		res.CheckpointsOK++
	}

	for _, run := range r.Runs {
		for range run.Count {
			e.TickOne(run.Action)
			tick := e.Tick()
			if want, ok := cpMap[tick]; ok {
				got := HashWorld(e.world)
				if got != want {
					return res, fmt.Errorf("replay hash mismatch at tick %d: got %016x, want %016x", tick, got, want)
				}
				res.CheckpointsOK++
			}
		}
	}

	gs := component.MustFindGameState(e.world)
	res.Ticks = e.Tick()
	res.Score = gs.Score
	res.Level = gs.Level
	res.FinalHash = HashWorld(e.world)

	if res.CheckpointsOK != len(r.Checkpoints) {
		return res, fmt.Errorf("replay checked %d of %d checkpoints (total ticks=%d)", res.CheckpointsOK, len(r.Checkpoints), res.Ticks)
	}

	return res, nil
}
