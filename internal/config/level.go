package config

//go:generate stringer -type=BonusSymbol
type BonusSymbol int

const (
	BonusCherries BonusSymbol = iota
	BonusStrawberry
	BonusPeach
	BonusApple
	BonusGrapes
	BonusGalaxian
	BonusBell
	BonusKey
)

type LevelSpecTable struct {
	LevelMin          uint8
	LevelMax          uint8 // inclusive
	BonusSymbol       BonusSymbol
	BonusPoints       uint16
	PacmanSpeed       uint32
	PacmanFrightSpeed uint32
	GhostSpeed        uint32
	GhostTunnelSpeed  uint32
	GhostFrightSpeed  uint32
	Elroy1DotsLeft    int
	Elroy1Speed       uint32
	Elroy2DotsLeft    int
	Elroy2Speed       uint32
	FrightenedFrames  int
	FrightenedFlashes int
}

const (
	Speed20    uint32 = 0x00008888
	Speed40    uint32 = 0x0000AAAA
	Speed45    uint32 = 0x0000B555
	Speed50    uint32 = 0x0000B6B6
	Speed55    uint32 = 0x0000B6B7
	Speed60    uint32 = 0x0000EEEE
	Speed65    uint32 = 0x0000EEEF
	Speed75    uint32 = 0x0000FFFD
	Speed80    uint32 = 0x0000FFFF
	Speed85    uint32 = 0x80000001 // start of fast mode
	Speed90    uint32 = 0x80000101
	Speed95    uint32 = 0x80000421
	Speed100   uint32 = 0x80001111
	Speed105   uint32 = 0x80002492
	SpeedEaten uint32 = 0x8000FFFF // 2 px/frame (2x base speed) for eaten ghost eyes

	SpeedNA      uint32 = 0
	FrightTimeNA        = -1
	FlashesNA           = -1
)

func SecondsToFrames(x float64) int {
	return int(x * ArcadeFPS)
}

type GhostModePhase struct {
	State          GhostState
	DurationFrames int
}

type GhostModePhaseTable struct {
	LevelMin int
	LevelMax int
	Phases   [8]GhostModePhase
}

var pacmanLevelTable = []LevelSpecTable{
	{LevelMin: 1, LevelMax: 1, BonusSymbol: BonusCherries, BonusPoints: 100, PacmanSpeed: Speed80, GhostSpeed: Speed75, GhostTunnelSpeed: Speed40, Elroy1DotsLeft: 20, Elroy1Speed: Speed80, Elroy2DotsLeft: 10, Elroy2Speed: Speed85, PacmanFrightSpeed: Speed90, GhostFrightSpeed: Speed50, FrightenedFrames: SecondsToFrames(6), FrightenedFlashes: 5},
	{LevelMin: 2, LevelMax: 2, BonusSymbol: BonusStrawberry, BonusPoints: 300, PacmanSpeed: Speed90, GhostSpeed: Speed85, GhostTunnelSpeed: Speed45, Elroy1DotsLeft: 30, Elroy1Speed: Speed90, Elroy2DotsLeft: 15, Elroy2Speed: Speed95, PacmanFrightSpeed: Speed95, GhostFrightSpeed: Speed55, FrightenedFrames: SecondsToFrames(5), FrightenedFlashes: 5},
	{LevelMin: 3, LevelMax: 3, BonusSymbol: BonusPeach, BonusPoints: 500, PacmanSpeed: Speed90, GhostSpeed: Speed85, GhostTunnelSpeed: Speed45, Elroy1DotsLeft: 40, Elroy1Speed: Speed90, Elroy2DotsLeft: 20, Elroy2Speed: Speed95, PacmanFrightSpeed: Speed95, GhostFrightSpeed: Speed55, FrightenedFrames: SecondsToFrames(4), FrightenedFlashes: 5},
	{LevelMin: 4, LevelMax: 4, BonusSymbol: BonusPeach, BonusPoints: 500, PacmanSpeed: Speed90, GhostSpeed: Speed85, GhostTunnelSpeed: Speed45, Elroy1DotsLeft: 40, Elroy1Speed: Speed90, Elroy2DotsLeft: 20, Elroy2Speed: Speed95, PacmanFrightSpeed: Speed95, GhostFrightSpeed: Speed55, FrightenedFrames: SecondsToFrames(3), FrightenedFlashes: 5},
	{LevelMin: 5, LevelMax: 5, BonusSymbol: BonusApple, BonusPoints: 700, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 40, Elroy1Speed: Speed100, Elroy2DotsLeft: 20, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(2), FrightenedFlashes: 5},
	{LevelMin: 6, LevelMax: 6, BonusSymbol: BonusApple, BonusPoints: 700, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 50, Elroy1Speed: Speed100, Elroy2DotsLeft: 25, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(5), FrightenedFlashes: 5},
	{LevelMin: 7, LevelMax: 8, BonusSymbol: BonusGrapes, BonusPoints: 1000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 50, Elroy1Speed: Speed100, Elroy2DotsLeft: 25, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(2), FrightenedFlashes: 5},
	{LevelMin: 9, LevelMax: 9, BonusSymbol: BonusGalaxian, BonusPoints: 2000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 60, Elroy1Speed: Speed100, Elroy2DotsLeft: 30, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 10, LevelMax: 10, BonusSymbol: BonusGalaxian, BonusPoints: 2000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 60, Elroy1Speed: Speed100, Elroy2DotsLeft: 30, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(5), FrightenedFlashes: 5},
	{LevelMin: 11, LevelMax: 11, BonusSymbol: BonusBell, BonusPoints: 3000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 60, Elroy1Speed: Speed100, Elroy2DotsLeft: 30, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(2), FrightenedFlashes: 5},
	{LevelMin: 12, LevelMax: 12, BonusSymbol: BonusBell, BonusPoints: 3000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 80, Elroy1Speed: Speed100, Elroy2DotsLeft: 40, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 13, LevelMax: 13, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 80, Elroy1Speed: Speed100, Elroy2DotsLeft: 40, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 14, LevelMax: 14, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 80, Elroy1Speed: Speed100, Elroy2DotsLeft: 40, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(3), FrightenedFlashes: 5},
	{LevelMin: 15, LevelMax: 16, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 100, Elroy1Speed: Speed100, Elroy2DotsLeft: 50, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 17, LevelMax: 17, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 100, Elroy1Speed: Speed100, Elroy2DotsLeft: 50, Elroy2Speed: Speed105, PacmanFrightSpeed: SpeedNA, GhostFrightSpeed: SpeedNA, FrightenedFrames: FrightTimeNA, FrightenedFlashes: FlashesNA},
	{LevelMin: 18, LevelMax: 18, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 100, Elroy1Speed: Speed100, Elroy2DotsLeft: 50, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: SecondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 19, LevelMax: 20, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 120, Elroy1Speed: Speed100, Elroy2DotsLeft: 60, Elroy2Speed: Speed105, PacmanFrightSpeed: SpeedNA, GhostFrightSpeed: SpeedNA, FrightenedFrames: FrightTimeNA, FrightenedFlashes: FlashesNA},
	{LevelMin: 21, LevelMax: 255, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed90, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 120, Elroy1Speed: Speed100, Elroy2DotsLeft: 60, Elroy2Speed: Speed105, PacmanFrightSpeed: SpeedNA, GhostFrightSpeed: SpeedNA, FrightenedFrames: FrightTimeNA, FrightenedFlashes: FlashesNA},
}

var ghostModePhaseTable = []GhostModePhaseTable{
	{
		LevelMin: 1, LevelMax: 1,
		Phases: [8]GhostModePhase{
			{State: Scatter, DurationFrames: SecondsToFrames(7)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(7)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(5)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(5)},
			{State: Chase, DurationFrames: -1},
		},
	},
	{
		LevelMin: 2, LevelMax: 4,
		Phases: [8]GhostModePhase{
			{State: Scatter, DurationFrames: SecondsToFrames(7)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(7)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(5)},
			{State: Chase, DurationFrames: SecondsToFrames(1033)},
			{State: Scatter, DurationFrames: 1},
			{State: Chase, DurationFrames: -1},
		},
	},
	{
		LevelMin: 5, LevelMax: 255,
		Phases: [8]GhostModePhase{
			{State: Scatter, DurationFrames: SecondsToFrames(5)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(5)},
			{State: Chase, DurationFrames: SecondsToFrames(20)},
			{State: Scatter, DurationFrames: SecondsToFrames(5)},
			{State: Chase, DurationFrames: SecondsToFrames(1037)},
			{State: Scatter, DurationFrames: 1},
			{State: Chase, DurationFrames: -1},
		},
	},
}

// GetLevelSpec looks up the arcade timing and speed specification for a level (1..255).
func GetLevelSpec(level int) LevelSpecTable {
	if level < 1 || level > 255 {
		panic("invalid level")
	}

	for _, spec := range pacmanLevelTable {
		if level >= int(spec.LevelMin) && level <= int(spec.LevelMax) {
			return spec
		}
	}

	return pacmanLevelTable[len(pacmanLevelTable)-1]
}

// GetGhostWaveState returns the scheduled global Scatter/Chase ghost wave for the
// given level and elapsed frame count.
func GetGhostWaveState(level, frameCount int) GhostState {
	var phases [8]GhostModePhase
	for _, pt := range ghostModePhaseTable {
		if level >= pt.LevelMin && level <= pt.LevelMax {
			phases = pt.Phases
			break
		}
	}

	ticks := 0
	for _, phase := range phases {
		if phase.DurationFrames == -1 {
			return phase.State
		}

		ticks += phase.DurationFrames
		if ticks >= frameCount {
			return phase.State
		}
	}

	return Chase
}

// IsFirstScatterWave reports whether waveFrames is still within the opening
// Scatter phase for level (used for Ms. Pac-Man's random Blinky/Pinky scatter).
func IsFirstScatterWave(level, waveFrames int) bool {
	for _, pt := range ghostModePhaseTable {
		if level >= pt.LevelMin && level <= pt.LevelMax {
			return waveFrames <= pt.Phases[0].DurationFrames
		}
	}
	return false
}

var msPacmanBonusTable = [7]struct {
	Symbol BonusSymbol
	Points int
}{
	{BonusCherries, 100},
	{BonusStrawberry, 200},
	{BonusPeach, 500},
	{BonusApple, 700},     // Pretzel on Ms. Pac-Man spritesheet (index 3)
	{BonusGrapes, 1000},   // Apple on Ms. Pac-Man spritesheet (index 4)
	{BonusGalaxian, 2000}, // Pear on Ms. Pac-Man spritesheet (index 5)
	{BonusBell, 5000},     // Banana on Ms. Pac-Man spritesheet (index 6)
}

// MsPacmanBonus returns the bonus fruit symbol and point value for Ms. Pac-Man.
// Levels 1..7 follow the fixed sequence (100..5000); Level 8+ selects randomly
// among the 7 items using randByte.
func MsPacmanBonus(level int, randByte byte) (BonusSymbol, int) {
	if level >= 1 && level <= len(msPacmanBonusTable) {
		entry := msPacmanBonusTable[level-1]
		return entry.Symbol, entry.Points
	}
	entry := msPacmanBonusTable[int(randByte)%len(msPacmanBonusTable)]
	return entry.Symbol, entry.Points
}

// IntermissionForLevel returns the 1-based arcade intermission act number (1, 2, or 3)
// to play after completing completedLevel, or 0 if gameplay proceeds directly to the next level.
func IntermissionForLevel(completedLevel int, pacmanMode bool) int {
	switch completedLevel {
	case 2:
		return 1
	case 5:
		return 2
	case 9, 13, 17:
		return 3
	default:
		if !pacmanMode && completedLevel > 17 && (completedLevel-9)%4 == 0 {
			return 3
		}
		return 0
	}
}
