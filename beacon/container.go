package beacon

// Container represents a single container belonging to a service. All fields
// are considered to be immutable with exceptions for the state fields.
type Container struct {
	// The globally unique identifier of the container. This is typically set
	// by the underlying container implementation.
	ID string

	// The name of the service to which the container belongs. Determined by
	// the event listener.
	Service string

	// The label metadata attached to the container.
	Labels map[string]string

	// The hostname to use when communicating with the container. This is
	// typically the private IP of the instance host.
	Hostname string

	// Network port bindings.
	Bindings []*Binding
}
