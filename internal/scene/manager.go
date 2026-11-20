package scene

// Scene defines a high-level game phase managed by Manager.
type Scene interface {
	Enter()
	Update()
	Draw()
	Exit()
}

// Manager owns the active Scene and performs clean Exit/Enter transitions.
type Manager struct {
	current Scene
	next    Scene
}

// NewManager creates a Scene Manager and immediately enters the initial scene if non-nil.
func NewManager(initial Scene) *Manager {
	m := &Manager{}
	if initial != nil {
		m.Start(initial)
	}
	return m
}

// Start transitions to initial and immediately invokes Enter (and Exit on any prior scene).
func (m *Manager) Start(initial Scene) {
	if initial == nil {
		return
	}
	m.Transition(initial)
	m.applyPendingTransitions()
}

// Transition schedules next to become the active scene.
func (m *Manager) Transition(next Scene) {
	m.next = next
}

// Current returns the currently active scene.
func (m *Manager) Current() Scene {
	return m.current
}

// Update advances the active scene by one frame and applies any pending scene transitions.
func (m *Manager) Update() {
	m.applyPendingTransitions()
	if m.current != nil {
		m.current.Update()
	}
	m.applyPendingTransitions()
}

// Draw renders the active scene.
func (m *Manager) Draw() {
	if m.current != nil {
		m.current.Draw()
	}
}

func (m *Manager) applyPendingTransitions() {
	for m.next != nil {
		next := m.next
		m.next = nil
		if m.current != nil {
			m.current.Exit()
		}
		m.current = next
		m.current.Enter()
	}
}
