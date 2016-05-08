package container

// Event is used by the Listener to notify Beacon of
// container actions.
type Event struct {
	Action    Action
	Container *Container
}
