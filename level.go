package main

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
	Speed20  uint32 = 0x00008888
	Speed40  uint32 = 0x0000AAAA
	Speed45  uint32 = 0x0000B555
	Speed50  uint32 = 0x0000B6B6
	Speed55  uint32 = 0x0000B6B7
	Speed60  uint32 = 0x0000EEEE
	Speed65  uint32 = 0x0000EEEF
	Speed75  uint32 = 0x0000FFFD
	Speed80  uint32 = 0x0000FFFF
	Speed85  uint32 = 0x80000001 // start of fast mode
	Speed90  uint32 = 0x80000101
	Speed95  uint32 = 0x80000421
	Speed100 uint32 = 0x80001111
	Speed105 uint32 = 0x80002492

	SpeedNA      uint32 = 0
	FrightTimeNA        = -1
	FlashesNA           = -1
)

func secondsToFrames(x float64) int {
	return int(x * 60.606061)
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
	{LevelMin: 1, LevelMax: 1, BonusSymbol: BonusCherries, BonusPoints: 100, PacmanSpeed: Speed80, GhostSpeed: Speed75, GhostTunnelSpeed: Speed40, Elroy1DotsLeft: 20, Elroy1Speed: Speed80, Elroy2DotsLeft: 10, Elroy2Speed: Speed85, PacmanFrightSpeed: Speed90, GhostFrightSpeed: Speed50, FrightenedFrames: secondsToFrames(6), FrightenedFlashes: 5},
	{LevelMin: 2, LevelMax: 2, BonusSymbol: BonusStrawberry, BonusPoints: 300, PacmanSpeed: Speed90, GhostSpeed: Speed85, GhostTunnelSpeed: Speed45, Elroy1DotsLeft: 30, Elroy1Speed: Speed90, Elroy2DotsLeft: 15, Elroy2Speed: Speed95, PacmanFrightSpeed: Speed95, GhostFrightSpeed: Speed55, FrightenedFrames: secondsToFrames(5), FrightenedFlashes: 5},
	{LevelMin: 3, LevelMax: 3, BonusSymbol: BonusPeach, BonusPoints: 500, PacmanSpeed: Speed90, GhostSpeed: Speed85, GhostTunnelSpeed: Speed45, Elroy1DotsLeft: 40, Elroy1Speed: Speed90, Elroy2DotsLeft: 20, Elroy2Speed: Speed95, PacmanFrightSpeed: Speed95, GhostFrightSpeed: Speed55, FrightenedFrames: secondsToFrames(4), FrightenedFlashes: 5},
	{LevelMin: 4, LevelMax: 4, BonusSymbol: BonusPeach, BonusPoints: 500, PacmanSpeed: Speed90, GhostSpeed: Speed85, GhostTunnelSpeed: Speed45, Elroy1DotsLeft: 40, Elroy1Speed: Speed90, Elroy2DotsLeft: 20, Elroy2Speed: Speed95, PacmanFrightSpeed: Speed95, GhostFrightSpeed: Speed55, FrightenedFrames: secondsToFrames(3), FrightenedFlashes: 5},
	{LevelMin: 5, LevelMax: 5, BonusSymbol: BonusApple, BonusPoints: 700, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 40, Elroy1Speed: Speed100, Elroy2DotsLeft: 20, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(2), FrightenedFlashes: 5},
	{LevelMin: 6, LevelMax: 6, BonusSymbol: BonusApple, BonusPoints: 700, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 50, Elroy1Speed: Speed100, Elroy2DotsLeft: 25, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(5), FrightenedFlashes: 5},
	{LevelMin: 7, LevelMax: 8, BonusSymbol: BonusGrapes, BonusPoints: 1000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 50, Elroy1Speed: Speed100, Elroy2DotsLeft: 25, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(2), FrightenedFlashes: 5},
	{LevelMin: 9, LevelMax: 9, BonusSymbol: BonusGalaxian, BonusPoints: 2000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 60, Elroy1Speed: Speed100, Elroy2DotsLeft: 30, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 10, LevelMax: 10, BonusSymbol: BonusGalaxian, BonusPoints: 2000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 60, Elroy1Speed: Speed100, Elroy2DotsLeft: 30, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(5), FrightenedFlashes: 5},
	{LevelMin: 11, LevelMax: 11, BonusSymbol: BonusBell, BonusPoints: 3000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 60, Elroy1Speed: Speed100, Elroy2DotsLeft: 30, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(2), FrightenedFlashes: 5},
	{LevelMin: 12, LevelMax: 12, BonusSymbol: BonusBell, BonusPoints: 3000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 80, Elroy1Speed: Speed100, Elroy2DotsLeft: 40, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 13, LevelMax: 13, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 80, Elroy1Speed: Speed100, Elroy2DotsLeft: 40, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 14, LevelMax: 14, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 80, Elroy1Speed: Speed100, Elroy2DotsLeft: 40, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(3), FrightenedFlashes: 5},
	{LevelMin: 15, LevelMax: 16, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 100, Elroy1Speed: Speed100, Elroy2DotsLeft: 50, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 17, LevelMax: 17, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 100, Elroy1Speed: Speed100, Elroy2DotsLeft: 50, Elroy2Speed: Speed105, PacmanFrightSpeed: SpeedNA, GhostFrightSpeed: SpeedNA, FrightenedFrames: FrightTimeNA, FrightenedFlashes: FlashesNA},
	{LevelMin: 18, LevelMax: 18, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 100, Elroy1Speed: Speed100, Elroy2DotsLeft: 50, Elroy2Speed: Speed105, PacmanFrightSpeed: Speed100, GhostFrightSpeed: Speed60, FrightenedFrames: secondsToFrames(1), FrightenedFlashes: 3},
	{LevelMin: 19, LevelMax: 20, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed100, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 120, Elroy1Speed: Speed100, Elroy2DotsLeft: 60, Elroy2Speed: Speed105, PacmanFrightSpeed: SpeedNA, GhostFrightSpeed: SpeedNA, FrightenedFrames: FrightTimeNA, FrightenedFlashes: FlashesNA},
	{LevelMin: 21, LevelMax: 255, BonusSymbol: BonusKey, BonusPoints: 5000, PacmanSpeed: Speed90, GhostSpeed: Speed95, GhostTunnelSpeed: Speed50, Elroy1DotsLeft: 120, Elroy1Speed: Speed100, Elroy2DotsLeft: 60, Elroy2Speed: Speed105, PacmanFrightSpeed: SpeedNA, GhostFrightSpeed: SpeedNA, FrightenedFrames: FrightTimeNA, FrightenedFlashes: FlashesNA},
}

var ghostModePhaseTable = []GhostModePhaseTable{
	{
		LevelMin: 1, LevelMax: 1,
		Phases: [8]GhostModePhase{
			{State: Scatter, DurationFrames: secondsToFrames(7)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(7)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(5)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(5)},
			{State: Chase, DurationFrames: -1},
		},
	},
	{
		LevelMin: 2, LevelMax: 4,
		Phases: [8]GhostModePhase{
			{State: Scatter, DurationFrames: secondsToFrames(7)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(7)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(5)},
			{State: Chase, DurationFrames: secondsToFrames(1033)},
			{State: Scatter, DurationFrames: 1},
			{State: Chase, DurationFrames: -1},
		},
	},
	{
		LevelMin: 5, LevelMax: 255,
		Phases: [8]GhostModePhase{
			{State: Scatter, DurationFrames: secondsToFrames(5)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(5)},
			{State: Chase, DurationFrames: secondsToFrames(20)},
			{State: Scatter, DurationFrames: secondsToFrames(5)},
			{State: Chase, DurationFrames: secondsToFrames(1037)},
			{State: Scatter, DurationFrames: 1},
			{State: Chase, DurationFrames: -1},
		},
	},
}

func getLevelSpec(level int) LevelSpecTable {
	if level < 1 || level > 255 {
		panic("invalid level")
	}

	return pacmanLevelTable[level-1]
}

func (g *Game) setLevel(level int) {
	g.level = level
	g.spec = getLevelSpec(level) // eventually change pacman/mspacman spec
}

func (g *Game) getGhostState() GhostState {
	level := g.level
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
		if ticks >= g.frameCount {
			return phase.State
		}
	}

	return Chase
}
