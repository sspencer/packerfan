package main

const MaxEvents = 256

//go:generate stringer -type=EventType
type EventType int

const (
	PlayerAteDot EventType = iota
	PlayerAtePowerUp
	GhostStateChange
)

type Event struct {
	Type EventType
	ID   int
	X    int
	Y    int
}

// EventStack stores events in LIFO order to match the original stack semantics.
type EventStack struct {
	events [MaxEvents]Event
	top    int
}

func NewEventStack() *EventStack {
	return &EventStack{
		top: 0,
	}
}

func (s *EventStack) Push(event Event) bool {
	if s.IsFull() {
		return false
	}

	s.events[s.top] = event
	s.top++
	return true
}

func (s *EventStack) Pop() (Event, bool) {
	if s.IsEmpty() {
		return Event{}, false
	}

	s.top--
	return s.events[s.top], true
}

func (s *EventStack) IsEmpty() bool {
	return s.top == 0
}

func (s *EventStack) IsFull() bool {
	return s.top >= MaxEvents
}
