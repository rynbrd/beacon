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

	// Network port bindings.
	Bindings []*Binding
}

// Equal returns true if this container is equal to another.
func (c *Container) Equal(b *Container) bool {
	if c == nil && b == nil {
		return true
	} else if c == nil || b == nil {
		return false
	}
	if c.ID != b.ID || c.Service != b.Service || len(c.Labels) != len(b.Labels) || len(c.Bindings) != len(b.Bindings) {
		return false
	}
	for name, val1 := range c.Labels {
		if val2, ok := b.Labels[name]; !ok || val1 != val2 {
			return false
		}
	}
	for n, binding1 := range c.Bindings {
		if !binding1.Equal(b.Bindings[n]) {
			return false
		}
	}
	return true
}
