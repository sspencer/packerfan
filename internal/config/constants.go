package config

const (
	GameWidth                   = 28
	GameHeight                  = 31
	TopOffset                   = 3
	ScreenWidth                 = GameWidth
	ScreenHeight                = GameHeight + 5
	Tile                        = 8
	Sprite                      = 16
	Half                        = Tile / 2
	PelletScore                 = 10
	PelletPause                 = 1
	PowerScore                  = 50
	PowerPause                  = 3
	GhostEatenPause             = 30
	ReadyDuration               = 122
	RoundClearedHoldFrames      = 61
	RoundClearedFlashHalfFrames = 14
	RoundClearedFlashes         = 4
	PlayerDyingHoldFrames       = 61
	PlayerDyingFrameCount       = 11
	PlayerDyingFrameTicks       = 7
	PlayerDyingPostFrames       = 30
	TargetFPS                   = 61
	ArcadeFPS                   = 60.606061
)

var ghostScores = [...]int{200, 400, 800, 1600}

// GhostScore returns the points awarded for the nth ghost eaten during a
// single power pellet (1->200, 2->400, 3->800, 4+->1600).
func GhostScore(eatenCount int) int {
	if eatenCount <= 0 {
		return 0
	}
	idx := eatenCount - 1
	if idx >= len(ghostScores) {
		idx = len(ghostScores) - 1
	}
	return ghostScores[idx]
}
