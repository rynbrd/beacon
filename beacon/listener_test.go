package beacon

import (
	"github.com/BlueDragonX/beacon/container"
)

// A test mock which implements the Listener inteface.
type MockListener struct {
	events    chan<- *container.Event
	listening chan<- bool
}

// NewMockListener creates a mock listener. If `listening` is non-nil then
// `true` will be sent to this channel on `Listen` and `false` on `Close`.
func NewMockListener(listening chan<- bool) *MockListener {
	return &MockListener{listening: listening}
}

// Emit a container event from the Listen channel.
func (m *MockListener) Emit(event *container.Event) {
	if m.events != nil {
		m.events <- event
	}
}

// Listen for events. Sets the internal events channel to `events` and sents
// `true` on `listening`.
func (m *MockListener) Listen(events chan<- *container.Event) {
	m.events = events
	if m.listening != nil {
		m.listening <- true
	}
}

// Close the listener. Send `false` on `listening`.
func (m *MockListener) Close() error {
	if m.listening != nil {
		m.listening <- false
	}
	return nil
}
