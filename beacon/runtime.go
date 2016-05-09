package beacon

// Runtime generates events from a container runtime.
type Runtime interface {
	// EmitEvents returns an event channel which emits container events. This
	// is called once by Beacon to receive events.
	EmitEvents() (<-chan *Event, error)

	// Close closes any open channels returned by Events. It also cleans up any
	// other resources. This is called once by Beacon when its Close is called.
	Close() error
}
