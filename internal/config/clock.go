package config

// FixedStepClock accumulates real frame deltas and emits the exact number of
// fixed-rate simulation ticks needed to run at ArcadeFPS (60.606061 Hz)
// regardless of monitor refresh rate.
type FixedStepClock struct {
	step        float64
	maxDelta    float64
	accumulator float64
}

// NewArcadeClock returns a FixedStepClock paced to config.ArcadeFPS (60.606061 Hz).
func NewArcadeClock() *FixedStepClock {
	return NewFixedStepClock(ArcadeFPS)
}

// NewFixedStepClock creates a fixed-timestep accumulator for the given tick rate in Hz.
func NewFixedStepClock(hz float64) *FixedStepClock {
	if hz <= 0 {
		hz = ArcadeFPS
	}
	return &FixedStepClock{
		step:     1.0 / hz,
		maxDelta: 0.25, // clamp large pauses (e.g. window drag/sleep) to prevent spiral of death
	}
}

// Advance adds dt seconds to the accumulator and returns how many fixed
// simulation steps should execute on this frame.
func (c *FixedStepClock) Advance(dt float64) int {
	if dt <= 0 {
		return 0
	}
	if dt > c.maxDelta {
		dt = c.maxDelta
	}
	c.accumulator += dt

	steps := 0
	for c.accumulator >= c.step {
		c.accumulator -= c.step
		steps++
	}
	return steps
}
