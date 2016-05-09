package beacon

// Backend recieves events routed to it by Beacon.
type Backend interface {
	// ProcessEvent instructs the backend to handle an event. This is called in
	// the main event processing loop and so should not block. If ProcessEvent
	// panics it will fail through the Beacon Run function.
	ProcessEvent(event *Event) error

	// Close frees any resources associated with the backend.
	Close() error
}
