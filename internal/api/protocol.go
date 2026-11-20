package api

import (
	"encoding/json"
	"fmt"

	"pacfan/internal/config"
	"pacfan/internal/sim"
)

// ProtocolVersion is the wire protocol schema version reported in hello.
const ProtocolVersion = 1

// ActionInput unmarshals an action from either a JSON integer (0..4) or a
// canonical wire string ("right", "left", "up", "down", "noop").
type ActionInput struct {
	Action sim.Action
	Set    bool
}

func (a *ActionInput) UnmarshalJSON(data []byte) error {
	var num int
	if err := json.Unmarshal(data, &num); err == nil {
		if num < 0 || num > int(sim.ActionNoop) {
			return fmt.Errorf("action %d out of range [0..4]", num)
		}
		a.Action = sim.Action(num)
		a.Set = true
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		parsed, ok := sim.ParseAction(str)
		if !ok {
			return fmt.Errorf("unknown action %q", str)
		}
		a.Action = parsed
		a.Set = true
		return nil
	}

	return fmt.Errorf("action must be an integer 0..4 or one of right, left, up, down, noop")
}

// Request is the incoming NDJSON envelope.
type Request struct {
	Cmd string `json:"cmd"`

	// reset fields
	Pacman          *bool           `json:"pacman,omitempty"`
	Level           *int            `json:"level,omitempty"`
	Seed            *int64          `json:"seed,omitempty"`
	EpisodicLife    *bool           `json:"episodic_life,omitempty"`
	EndOnLevelClear *bool           `json:"end_on_level_clear,omitempty"`
	MaxTicks        *int            `json:"max_ticks,omitempty"`
	Record          string          `json:"record,omitempty"`
	Meta            json.RawMessage `json:"meta,omitempty"`

	// step fields
	Action ActionInput `json:"action"`
	Mode   string      `json:"mode,omitempty"`
	Frames int         `json:"frames,omitempty"`

	// observe fields
	Maze bool `json:"maze,omitempty"`
}

// ErrorResponse is returned for malformed requests or command errors.
type ErrorResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// ArtIdentity describes the game PNG used to parse the maze layouts.
type ArtIdentity struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// MazeSummary describes one static maze layout sent once in hello.
type MazeSummary struct {
	Index int      `json:"index"`
	Dots  int      `json:"dots"`
	Rows  []string `json:"rows"`
}

// HelloResponse carries all static environment metadata once per connection.
type HelloResponse struct {
	OK       bool          `json:"ok"`
	Protocol int           `json:"protocol"`
	Art      ArtIdentity   `json:"art"`
	MazeHash string        `json:"maze_hash"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Mazes    []MazeSummary `json:"mazes"`
	Actions  [5]string     `json:"actions"`
	Ghosts   [4]string     `json:"ghosts"`
}

// PlayerState is the player snapshot inside StateSnapshot.
type PlayerState struct {
	X     int                   `json:"x"`
	Y     int                   `json:"y"`
	Dir   int                   `json:"dir"`
	Legal [config.DirCount]bool `json:"legal"`
}

// GhostStateSnapshot is a single ghost snapshot inside StateSnapshot.
type GhostStateSnapshot struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Dir   int    `json:"dir"`
	State string `json:"state"`
}

// FruitStateSnapshot describes the bonus fruit state.
type FruitStateSnapshot struct {
	Active bool   `json:"active"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Symbol string `json:"symbol,omitempty"`
	Points int    `json:"points,omitempty"`
	Timer  int    `json:"timer,omitempty"`
}

// StateSnapshot is the compact per-step gameplay state (no maze grid).
type StateSnapshot struct {
	Level         int                   `json:"level"`
	Score         int                   `json:"score"`
	Lives         int                   `json:"lives"`
	DotsRemaining int                   `json:"dots_remaining"`
	Wave          string                `json:"wave"`
	FrightFrames  int                   `json:"fright_frames"`
	GhostsEaten   int                   `json:"ghosts_eaten"`
	Player        PlayerState           `json:"player"`
	Ghosts        [4]GhostStateSnapshot `json:"ghosts"`
	Fruit         FruitStateSnapshot    `json:"fruit"`
}

// ResetResponse is returned by the reset command.
type ResetResponse struct {
	OK        bool          `json:"ok"`
	Tick      int           `json:"tick"`
	MazeIndex int           `json:"maze_index"`
	State     StateSnapshot `json:"state"`
}

// StepResponse is returned by the step command.
type StepResponse struct {
	OK         bool          `json:"ok"`
	Tick       int           `json:"tick"`
	Ticks      int           `json:"ticks"`
	Reward     int           `json:"reward"`
	Terminated bool          `json:"terminated"`
	Truncated  bool          `json:"truncated"`
	Events     []string      `json:"events"`
	Eaten      [][2]int      `json:"eaten"`
	MazeIndex  *int          `json:"maze_index,omitempty"`
	State      StateSnapshot `json:"state"`
}

// ObserveResponse is returned by the observe command.
type ObserveResponse struct {
	OK        bool          `json:"ok"`
	Tick      int           `json:"tick"`
	MazeIndex int           `json:"maze_index"`
	Rows      []string      `json:"rows,omitempty"`
	State     StateSnapshot `json:"state"`
}

// CloseResponse is returned by the close command.
type CloseResponse struct {
	OK bool `json:"ok"`
}

// GhostNames lists the canonical ghost names in component.ActorID order.
var GhostNames = [4]string{"blinky", "pinky", "inky", "clyde"}

func ghostStateString(s config.GhostState) string {
	switch s {
	case config.Scatter:
		return "scatter"
	case config.Chase:
		return "chase"
	case config.Frightened:
		return "frightened"
	case config.Eaten:
		return "eaten"
	case config.EnteringHouse:
		return "entering_house"
	case config.InHouse:
		return "in_house"
	case config.LeavingHouse:
		return "leaving_house"
	default:
		return s.String()
	}
}

func bonusSymbolString(sym config.BonusSymbol) string {
	switch sym {
	case config.BonusCherries:
		return "cherries"
	case config.BonusStrawberry:
		return "strawberry"
	case config.BonusPeach:
		return "peach"
	case config.BonusApple:
		return "apple"
	case config.BonusGrapes:
		return "grapes"
	case config.BonusGalaxian:
		return "galaxian"
	case config.BonusBell:
		return "bell"
	case config.BonusKey:
		return "key"
	default:
		return sym.String()
	}
}

func snapshotState(env *sim.Env) StateSnapshot {
	gs := env.GameState()
	playerTr := env.PlayerTransform()

	snap := StateSnapshot{
		Level:         gs.Level,
		Score:         gs.Score,
		Lives:         gs.Lives,
		DotsRemaining: gs.DotsRemaining,
		Wave:          ghostStateString(gs.GhostState),
		FrightFrames:  gs.FrightFrames,
		GhostsEaten:   gs.NumGhostsEaten,
		Player: PlayerState{
			X:     playerTr.Pos.X,
			Y:     playerTr.Pos.Y,
			Dir:   int(playerTr.Dir),
			Legal: env.LegalActions(),
		},
	}

	for i := range 4 {
		tr := env.GhostTransform(i)
		g := env.GhostData(i)
		snap.Ghosts[i] = GhostStateSnapshot{
			X:     tr.Pos.X,
			Y:     tr.Pos.Y,
			Dir:   int(tr.Dir),
			State: ghostStateString(g.State),
		}
	}

	if gs.FruitActive {
		snap.Fruit = FruitStateSnapshot{
			Active: true,
			X:      gs.FruitPos.X,
			Y:      gs.FruitPos.Y,
			Symbol: bonusSymbolString(gs.FruitSymbol),
			Points: gs.FruitPoints,
			Timer:  gs.FruitTimer,
		}
	}

	return snap
}
