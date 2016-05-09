package beacon

// New creates a Beacon which receieves events from the `runtime` and uses
// `routes` to queue them into appropriate backends. New does not start the
// Beacon.
func New(runtime Runtime, routes []*Route) Beacon {
	return nil
}

// Beacon processes events from a runtime into one or more backends. It
// maintains state for container it sees and provides methods for interacting
// with that state.
type Beacon interface {
	// Run the beacon. This blocks until Close is called.
	Run() error

	// Close the beacon. Stops a running beacon and cleans up any resources
	// attached to it.
	Close() error

	// Containers retrieves the list of containers that Beacon has discovered.
	// An optional filter may be provided in order to limit the containers
	// returned.
	Containers(filter Filter) ([]*Container, error)
}
