package api

import (
	"fmt"

	"pacfan/internal/component"
	"pacfan/internal/config"
	"pacfan/internal/sheet"
	"pacfan/internal/sim"
)

// SessionConfig configures an API Session.
type SessionConfig struct {
	Sheet    *sheet.Sheet
	Mazes    *component.MazeSet
	Defaults sim.Config
}

// Session owns one sim.Env and translates NDJSON requests into simulation calls.
type Session struct {
	art      ArtIdentity
	mazes    *component.MazeSet
	defaults sim.Config
	env      *sim.Env
	hello    HelloResponse
	hasReset bool
}

// NewSession constructs a Session with a cached hello payload and an underlying sim.Env.
func NewSession(cfg SessionConfig) *Session {
	defaults := cfg.Defaults
	defaults.Mazes = cfg.Mazes
	if defaults.StartLevel <= 0 {
		defaults.StartLevel = 1
	}

	var art ArtIdentity
	if cfg.Sheet != nil {
		art = ArtIdentity{
			File:   cfg.Sheet.Name(),
			SHA256: cfg.Sheet.SHA256(),
		}
	}

	mazeSummaries := make([]MazeSummary, component.NumMazes)
	var mazeHash string
	if cfg.Mazes != nil {
		mazeHash = cfg.Mazes.Hash()
		for i := range component.NumMazes {
			tmpl := cfg.Mazes.Template(i)
			mazeSummaries[i] = MazeSummary{
				Index: tmpl.Index,
				Dots:  tmpl.Dots,
				Rows:  tmpl.Rows(),
			}
		}
	}

	hello := HelloResponse{
		OK:       true,
		Protocol: ProtocolVersion,
		Art:      art,
		MazeHash: mazeHash,
		Width:    config.GameWidth,
		Height:   config.GameHeight,
		Mazes:    mazeSummaries,
		Actions:  sim.ActionNames,
		Ghosts:   GhostNames,
	}

	return &Session{
		art:      art,
		mazes:    cfg.Mazes,
		defaults: defaults,
		env:      sim.NewEnv(defaults),
		hello:    hello,
	}
}

// Env returns the session's underlying simulation environment.
func (s *Session) Env() *sim.Env {
	return s.env
}

// Handle processes a single Request and returns the JSON-serializable response
// plus a boolean indicating whether the session should close.
func (s *Session) Handle(req *Request) (any, bool) {
	switch req.Cmd {
	case "hello":
		return s.hello, false
	case "reset":
		return s.handleReset(req), false
	case "step":
		return s.handleStep(req), false
	case "observe":
		return s.handleObserve(req), false
	case "close":
		s.Close()
		return CloseResponse{OK: true}, true
	default:
		return ErrorResponse{OK: false, Error: fmt.Sprintf("unknown command %q", req.Cmd)}, false
	}
}

// Close releases any resources held by the session (such as open replay recorders).
func (s *Session) Close() error {
	return nil
}

func (s *Session) handleReset(req *Request) any {
	pacman := s.defaults.PacmanMode
	if req.Pacman != nil {
		pacman = *req.Pacman
	}
	level := s.defaults.StartLevel
	if req.Level != nil {
		level = *req.Level
	}
	seed := s.defaults.Seed
	if req.Seed != nil {
		seed = *req.Seed
	}
	episodicLife := s.defaults.EpisodicLife
	if req.EpisodicLife != nil {
		episodicLife = *req.EpisodicLife
	}
	endOnLevelClear := s.defaults.EndOnLevelClear
	if req.EndOnLevelClear != nil {
		endOnLevelClear = *req.EndOnLevelClear
	}
	maxTicks := s.defaults.MaxTicks
	if req.MaxTicks != nil {
		maxTicks = *req.MaxTicks
	}

	s.env.Reset(&sim.ResetOptions{
		PacmanMode:      &pacman,
		StartLevel:      &level,
		Seed:            &seed,
		EpisodicLife:    &episodicLife,
		EndOnLevelClear: &endOnLevelClear,
		MaxTicks:        &maxTicks,
	})
	s.hasReset = true

	maze := component.MustFindMaze(s.env.World())
	return ResetResponse{
		OK:        true,
		Tick:      s.env.Tick(),
		MazeIndex: maze.GetIndex(),
		State:     snapshotState(s.env),
	}
}

func (s *Session) handleStep(req *Request) any {
	if !s.hasReset {
		return ErrorResponse{OK: false, Error: "must call reset before step"}
	}

	action := sim.ActionNoop
	if req.Action.Set {
		action = req.Action.Action
	}

	mode, ok := sim.ParseStepMode(req.Mode)
	if !ok {
		return ErrorResponse{OK: false, Error: fmt.Sprintf("unknown step mode %q", req.Mode)}
	}

	res := s.env.Step(action, mode, req.Frames)

	events := res.Events.Strings()
	if events == nil {
		events = []string{}
	}
	eaten := make([][2]int, len(res.Eaten))
	for i, c := range res.Eaten {
		eaten[i] = [2]int{c.X, c.Y}
	}

	resp := StepResponse{
		OK:         true,
		Tick:       res.Tick,
		Ticks:      res.Ticks,
		Reward:     res.Reward,
		Terminated: res.Terminated,
		Truncated:  res.Truncated,
		Events:     events,
		Eaten:      eaten,
		State:      snapshotState(s.env),
	}
	if res.LevelChanged {
		idx := res.MazeIndex
		resp.MazeIndex = &idx
	}
	return resp
}

func (s *Session) handleObserve(req *Request) any {
	if !s.hasReset {
		return ErrorResponse{OK: false, Error: "must call reset before observe"}
	}
	maze := component.MustFindMaze(s.env.World())
	resp := ObserveResponse{
		OK:        true,
		Tick:      s.env.Tick(),
		MazeIndex: maze.GetIndex(),
		State:     snapshotState(s.env),
	}
	if req.Maze {
		resp.Rows = maze.Rows()
	}
	return resp
}
